package corehttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type createBusinessRequest struct {
	UserID       int64           `json:"user_id" binding:"required,gt=0"`
	BusinessName string          `json:"business_name" binding:"required"`
	Description  *string         `json:"description"`
	BusinessData json.RawMessage `json:"business_data"`
}

type updateBusinessRequest struct {
	UserID       int64           `json:"user_id" binding:"required,gt=0"`
	BusinessID   int64           `json:"business_id" binding:"required,gt=0"`
	BusinessName string          `json:"business_name" binding:"required"`
	Description  json.RawMessage `json:"description"`
	BusinessData json.RawMessage `json:"business_data"`
}

type deleteBusinessRequest struct {
	UserID     int64 `json:"user_id" binding:"required,gt=0"`
	BusinessID int64 `json:"business_id" binding:"required,gt=0"`
}

type businessMemberRequest struct {
	UserID       int64  `json:"user_id" binding:"required,gt=0"`
	BusinessID   int64  `json:"business_id" binding:"required,gt=0"`
	MemberUserID int64  `json:"member_user_id" binding:"required,gt=0"`
	Role         string `json:"role"`
}

type businessMemberDeleteRequest struct {
	UserID       int64 `json:"user_id" binding:"required,gt=0"`
	BusinessID   int64 `json:"business_id" binding:"required,gt=0"`
	MemberUserID int64 `json:"member_user_id" binding:"required,gt=0"`
}

type businessRecord struct {
	BusinessID   int64           `json:"business_id"`
	UserID       int64           `json:"user_id"`
	BusinessName string          `json:"business_name"`
	Description  *string         `json:"description"`
	BusinessData json.RawMessage `json:"business_data"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type businessResponse struct {
	Success bool           `json:"success"`
	Data    businessRecord `json:"data"`
}

type businessListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Businesses []businessRecord `json:"businesses"`
		NextCursor *string          `json:"next_cursor"`
	} `json:"data"`
}

func (api *API) createBusiness(c *gin.Context) {
	var request createBusinessRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "user_id and business_name are required")
		return
	}
	request.BusinessName = strings.TrimSpace(request.BusinessName)
	if request.UserID > math.MaxInt32 {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "user_id exceeds the current PostgreSQL INTEGER range")
		return
	}
	if !validBusinessName(request.BusinessName) {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "business_name must contain between 1 and 120 characters")
		return
	}
	if request.Description != nil && utf8.RuneCountInString(*request.Description) > 1000 {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "description must not exceed 1000 characters")
		return
	}
	businessData, err := normalizeJSONObject(request.BusinessData)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "business_data must be a JSON object")
		return
	}
	request.BusinessData = businessData
	if !api.requirePostgres(c) {
		return
	}
	if !api.requireSession(c, request.UserID, c.GetHeader("X-Session-ID")) {
		return
	}

	business, err := insertBusiness(c, api.postgres, request.UserID, request.BusinessName, request.Description, businessData)
	if err != nil {
		slog.Error("failed to create business", "user_id", request.UserID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "business_creation_failed", "could not create business")
		return
	}
	c.JSON(http.StatusCreated, businessResponse{Success: true, Data: business})
}

func (api *API) listBusinesses(c *gin.Context) {
	userID, ok := positiveID(c, c.Query("user_id"), "user_id")
	if !ok || userID > math.MaxInt32 {
		if ok {
			writeAPIError(c, http.StatusBadRequest, "invalid_request", "user_id exceeds the current PostgreSQL INTEGER range")
		}
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
	cursor, err := parseBusinessCursor(c.Query("cursor"))
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "cursor is invalid")
		return
	}

	businesses, nextCursor, err := selectBusinesses(c, api.postgres, userID, cursor, limit)
	if err != nil {
		slog.Error("failed to list businesses", "user_id", userID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "business_list_failed", "could not retrieve businesses")
		return
	}
	response := businessListResponse{Success: true}
	response.Data.Businesses = businesses
	response.Data.NextCursor = nextCursor
	c.JSON(http.StatusOK, response)
}

func (api *API) updateBusiness(c *gin.Context) {
	var request updateBusinessRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "user_id, business_id, business_name, description, and business_data are required")
		return
	}
	request.BusinessName = strings.TrimSpace(request.BusinessName)
	if !validBusinessName(request.BusinessName) {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "business_name must contain between 1 and 120 characters")
		return
	}
	description, validDescription := parseBusinessDescription(request.Description)
	if request.UserID > math.MaxInt32 || request.BusinessID > math.MaxInt32 {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "user_id and business_id must fit the current PostgreSQL INTEGER range")
		return
	}
	if !validDescription {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "description must be a string or null")
		return
	}
	if description != nil && utf8.RuneCountInString(*description) > 1000 {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "description must not exceed 1000 characters")
		return
	}
	if len(request.BusinessData) == 0 {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "business_data is required")
		return
	}
	businessData, err := normalizeJSONObject(request.BusinessData)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "business_data must be a JSON object")
		return
	}
	request.BusinessData = businessData
	if !api.requirePostgres(c) {
		return
	}
	if !api.requireSession(c, request.UserID, c.GetHeader("X-Session-ID")) {
		return
	}

	business, err := updateBusinessRecord(c, api.postgres, request, description)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeAPIError(c, http.StatusNotFound, "business_not_found", "business not found")
	case errors.Is(err, errBusinessNotOwned):
		writeAPIError(c, http.StatusForbidden, "business_forbidden", "only the business owner can update it")
	case err != nil:
		slog.Error("failed to update business", "business_id", request.BusinessID, "user_id", request.UserID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "business_update_failed", "could not update business")
	default:
		c.JSON(http.StatusOK, businessResponse{Success: true, Data: business})
	}
}

func (api *API) deleteBusiness(c *gin.Context) {
	var request deleteBusinessRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.UserID > math.MaxInt32 || request.BusinessID > math.MaxInt32 {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "user_id and business_id are required and must be positive")
		return
	}
	if !api.requirePostgres(c) {
		return
	}
	if !api.requireSession(c, request.UserID, c.GetHeader("X-Session-ID")) {
		return
	}

	deletedID, err := deleteBusinessRecord(c, api.postgres, request.BusinessID, request.UserID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeAPIError(c, http.StatusNotFound, "business_not_found", "business not found")
	case errors.Is(err, errBusinessNotOwned):
		writeAPIError(c, http.StatusForbidden, "business_forbidden", "only the business owner can delete it")
	case err != nil:
		slog.Error("failed to delete business", "business_id", request.BusinessID, "user_id", request.UserID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "business_deletion_failed", "could not delete business")
	default:
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "business deleted successfully",
			"data":    gin.H{"business_id": deletedID},
		})
	}
}

func (api *API) addBusinessMember(c *gin.Context) {
	var request businessMemberRequest
	if err := c.ShouldBindJSON(&request); err != nil ||
		request.UserID > math.MaxInt32 || request.BusinessID > math.MaxInt32 || request.MemberUserID > math.MaxInt32 {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "user_id, business_id, and member_user_id are required and must be positive")
		return
	}
	if request.Role == "" {
		request.Role = "member"
	}
	if request.Role != "member" && request.Role != "admin" {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "role must be member or admin")
		return
	}
	if !api.requirePostgres(c) {
		return
	}
	if !api.requireSession(c, request.UserID, c.GetHeader("X-Session-ID")) {
		return
	}

	err := addBusinessMemberRecord(c, api.postgres, request)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeAPIError(c, http.StatusNotFound, "business_not_found", "business not found")
	case errors.Is(err, errBusinessNotOwned):
		writeAPIError(c, http.StatusForbidden, "business_forbidden", "only the business owner can add members")
	case isUniqueViolation(err):
		writeAPIError(c, http.StatusConflict, "member_already_exists", "user is already a member of this business")
	case isForeignKeyViolation(err):
		writeAPIError(c, http.StatusNotFound, "user_not_found", "member user not found")
	case err != nil:
		slog.Error("failed to add business member", "business_id", request.BusinessID, "member_user_id", request.MemberUserID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "business_member_add_failed", "could not add business member")
	default:
		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "member added successfully",
			"data": gin.H{
				"business_id":    request.BusinessID,
				"member_user_id": request.MemberUserID,
				"role":           request.Role,
			},
		})
	}
}

func (api *API) removeBusinessMember(c *gin.Context) {
	var request businessMemberDeleteRequest
	if err := c.ShouldBindJSON(&request); err != nil ||
		request.UserID > math.MaxInt32 || request.BusinessID > math.MaxInt32 || request.MemberUserID > math.MaxInt32 {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "user_id, business_id, and member_user_id are required and must be positive")
		return
	}
	if request.UserID == request.MemberUserID {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "the business owner cannot remove themselves")
		return
	}
	if !api.requirePostgres(c) {
		return
	}
	if !api.requireSession(c, request.UserID, c.GetHeader("X-Session-ID")) {
		return
	}

	err := removeBusinessMemberRecord(c, api.postgres, request)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeAPIError(c, http.StatusNotFound, "business_or_member_not_found", "business or member not found")
	case errors.Is(err, errBusinessNotOwned):
		writeAPIError(c, http.StatusForbidden, "business_forbidden", "only the business owner can remove members")
	case err != nil:
		slog.Error("failed to remove business member", "business_id", request.BusinessID, "member_user_id", request.MemberUserID, "error", err)
		writeAPIError(c, http.StatusInternalServerError, "business_member_remove_failed", "could not remove business member")
	default:
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "member removed successfully",
			"data": gin.H{
				"business_id":    request.BusinessID,
				"member_user_id": request.MemberUserID,
			},
		})
	}
}

func validBusinessName(name string) bool {
	length := utf8.RuneCountInString(name)
	return length > 0 && length <= 120
}

func normalizeJSONObject(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(`{}`), nil
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, errors.New("value must be a JSON object")
	}
	return raw, nil
}

func parseBusinessDescription(raw json.RawMessage) (*string, bool) {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil, false
	}
	if strings.TrimSpace(string(raw)) == "null" {
		return nil, true
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false
	}
	return &value, true
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23503"
}
