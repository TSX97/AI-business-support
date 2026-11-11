package corehttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type createChatRequest struct {
	UserID    int64  `json:"user_id" binding:"required,gt=0"`
	SessionID string `json:"session_id" binding:"required"`
	ChatName  string `json:"chat_name" binding:"required"`
}

type renameChatRequest struct {
	NewChatName string `json:"new_chat_name" binding:"required"`
}

type chatResponse struct {
	Success bool       `json:"success"`
	Data    chatRecord `json:"data"`
}

type chatRecord struct {
	ChatID    int64     `json:"chat_id"`
	UserID    int64     `json:"user_id"`
	CompanyID *int64    `json:"company_id,omitempty"`
	ChatName  *string   `json:"chat_name"`
	CreatedAt time.Time `json:"created_at"`
}

type chatListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Chats      []chatListItem `json:"chats"`
		NextCursor *string        `json:"next_cursor"`
	} `json:"data"`
}

type chatListItem struct {
	chatRecord
	LastMessagePreview *string    `json:"last_message_preview"`
	LastMessageAt      *time.Time `json:"last_message_at"`
}

func (api *API) root(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "core"})
}

func (api *API) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (api *API) hello(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello from Core service!"})
}

func (api *API) createChat(c *gin.Context) {
	var request createChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "user_id, session_id, and chat_name are required; user_id must be positive")
		return
	}

	request.ChatName = strings.TrimSpace(request.ChatName)
	if !validChatName(request.ChatName) {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "chat_name must contain between 1 and 80 characters")
		return
	}
	if !validSessionID(request.SessionID) {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "session_id must be between 1 and 256 bytes without surrounding whitespace")
		return
	}
	if !api.requirePostgres(c) {
		return
	}
	if !api.requireSession(c, request.UserID, request.SessionID) {
		return
	}

	chat, err := insertChat(c, api.postgres, request.UserID, request.ChatName)
	if err != nil {
		slog.Error("failed to insert chat", "user_id", request.UserID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "chat_creation_failed", "could not create chat")
		return
	}

	c.JSON(http.StatusCreated, chatResponse{Success: true, Data: chat})
}

func (api *API) getChat(c *gin.Context) {
	chatID, ok := positiveID(c, c.Param("id"), "chat ID")
	if !ok {
		return
	}
	userID, ok := positiveID(c, c.Query("user_id"), "user_id")
	if !ok {
		return
	}
	if !api.requirePostgres(c) {
		return
	}
	if !api.requireSession(c, userID, c.GetHeader("X-Session-ID")) {
		return
	}

	chat, err := selectChatByID(c, api.postgres, chatID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && chat.UserID != userID) {
		writeAPIError(c, http.StatusNotFound, "chat_not_found", "chat not found")
		return
	}
	if err != nil {
		slog.Error("failed to retrieve chat", "chat_id", chatID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "chat_retrieval_failed", "could not retrieve chat")
		return
	}

	c.JSON(http.StatusOK, chatResponse{Success: true, Data: chat})
}

func (api *API) renameChat(c *gin.Context) {
	chatID, ok := positiveID(c, c.Param("id"), "chat ID")
	if !ok {
		return
	}
	userID, ok := positiveID(c, c.Query("user_id"), "user_id")
	if !ok {
		return
	}

	var request renameChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "new_chat_name is required")
		return
	}
	request.NewChatName = strings.TrimSpace(request.NewChatName)
	if !validChatName(request.NewChatName) {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "new_chat_name must contain between 1 and 80 characters")
		return
	}
	if !api.requirePostgres(c) {
		return
	}
	if !api.requireSession(c, userID, c.GetHeader("X-Session-ID")) {
		return
	}

	chat, err := updateChatTitle(c, api.postgres, chatID, userID, request.NewChatName)
	if errors.Is(err, pgx.ErrNoRows) {
		writeAPIError(c, http.StatusNotFound, "chat_not_found", "chat not found")
		return
	}
	if err != nil {
		slog.Error("failed to rename chat", "chat_id", chatID, "user_id", userID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "chat_update_failed", "could not rename chat")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "chat renamed successfully",
		"data":    chat,
	})
}

func (api *API) deleteChat(c *gin.Context) {
	chatID, ok := positiveID(c, c.Param("id"), "chat ID")
	if !ok {
		return
	}
	userID, ok := positiveID(c, c.Query("user_id"), "user_id")
	if !ok {
		return
	}
	if !api.requirePostgres(c) {
		return
	}
	if !api.requireSession(c, userID, c.GetHeader("X-Session-ID")) {
		return
	}

	deletedChatID, err := deleteChatByID(c, api.postgres, chatID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeAPIError(c, http.StatusNotFound, "chat_not_found", "chat not found")
		return
	}
	if err != nil {
		slog.Error("failed to delete chat", "chat_id", chatID, "user_id", userID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "chat_deletion_failed", "could not delete chat")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "chat deleted successfully",
		"data": gin.H{
			"chat_id": deletedChatID,
		},
	})
}

func (api *API) listChats(c *gin.Context) {
	userID, ok := positiveID(c, c.Query("user_id"), "user_id")
	if !ok {
		return
	}
	if !api.requirePostgres(c) {
		return
	}
	if !api.requireSession(c, userID, c.GetHeader("X-Session-ID")) {
		return
	}

	limit := 25
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			writeAPIError(c, http.StatusBadRequest, "invalid_request", "limit must be between 1 and 100")
			return
		}
		limit = parsedLimit
	}

	var beforeID *int64
	if cursor := c.Query("cursor"); cursor != "" {
		parsedCursor, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil || parsedCursor <= 0 {
			writeAPIError(c, http.StatusBadRequest, "invalid_request", "cursor must be a positive chat ID")
			return
		}
		beforeID = &parsedCursor
	}

	chats, nextCursor, err := selectChats(c, api.postgres, userID, beforeID, limit)
	if err != nil {
		slog.Error("failed to list chats", "user_id", userID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "chat_list_failed", "could not retrieve chats")
		return
	}

	response := chatListResponse{Success: true}
	response.Data.Chats = chats
	response.Data.NextCursor = nextCursor
	c.JSON(http.StatusOK, response)
}

func (api *API) requirePostgres(c *gin.Context) bool {
	if api.postgres != nil {
		return true
	}
	writeAPIError(c, http.StatusServiceUnavailable, "service_unavailable", "chat storage is unavailable")
	return false
}

func validChatName(name string) bool {
	length := utf8.RuneCountInString(name)
	return length > 0 && length <= 80
}

func positiveID(c *gin.Context, value, field string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err == nil && id > 0 {
		return id, true
	}
	writeAPIError(c, http.StatusBadRequest, "invalid_request", field+" must be a positive integer")
	return 0, false
}

func writeAPIError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func writeSessionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errSessionNotFound):
		writeAPIError(c, http.StatusUnauthorized, "invalid_session", errSessionNotFound.Error())
	case errors.Is(err, errSessionUserMismatch):
		writeAPIError(c, http.StatusForbidden, "session_user_mismatch", errSessionUserMismatch.Error())
	default:
		slog.Error("failed to validate session", "error", err)
		writeAPIError(c, http.StatusServiceUnavailable, "session_store_unavailable", "could not validate session")
	}
}
