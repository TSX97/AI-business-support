package corehttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"

	"github.com/jackc/pgx/v5"
)

type chatDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func insertChat(ctx context.Context, db chatDB, userID int64, title string) (chatRecord, error) {
	var chat chatRecord
	var companyID sql.NullInt64
	var chatName sql.NullString
	err := db.QueryRow(
		ctx,
		`INSERT INTO core.chats (user_id, title) VALUES ($1, $2) RETURNING id, user_id, company_id, title, created_at`,
		userID,
		title,
	).Scan(&chat.ChatID, &chat.UserID, &companyID, &chatName, &chat.CreatedAt)
	setNullableChatFields(&chat, companyID, chatName)
	return chat, err
}

func selectChatByID(ctx context.Context, db chatDB, chatID int64) (chatRecord, error) {
	var chat chatRecord
	var companyID sql.NullInt64
	var chatName sql.NullString
	err := db.QueryRow(ctx, `SELECT * FROM core.chats WHERE id = $1`, chatID).
		Scan(&chat.ChatID, &chat.UserID, &companyID, &chatName, &chat.CreatedAt)
	setNullableChatFields(&chat, companyID, chatName)
	return chat, err
}

func updateChatTitle(ctx context.Context, db chatDB, chatID, userID int64, title string) (chatRecord, error) {
	var chat chatRecord
	var companyID sql.NullInt64
	var chatName sql.NullString
	err := db.QueryRow(
		ctx,
		`UPDATE core.chats SET title = $3 WHERE id = $1 AND user_id = $2 RETURNING id, user_id, company_id, title, created_at`,
		chatID,
		userID,
		title,
	).Scan(&chat.ChatID, &chat.UserID, &companyID, &chatName, &chat.CreatedAt)
	setNullableChatFields(&chat, companyID, chatName)
	return chat, err
}

func deleteChatByID(ctx context.Context, db chatDB, chatID, userID int64) (int64, error) {
	var deletedChatID int64
	err := db.QueryRow(
		ctx,
		`DELETE FROM core.chats WHERE id = $1 AND user_id = $2 RETURNING id`,
		chatID,
		userID,
	).Scan(&deletedChatID)
	return deletedChatID, err
}

func selectChats(ctx context.Context, db chatDB, userID int64, beforeID *int64, limit int) ([]chatListItem, *string, error) {
	var encodedChats []byte
	err := db.QueryRow(
		ctx,
		`SELECT COALESCE(json_agg(row_to_json(chat_row)), '[]'::json)
		 FROM (
		   SELECT
		     c.id AS chat_id,
		     c.user_id,
		     c.company_id,
		     c.title AS chat_name,
		     c.created_at,
		     last_message.preview AS last_message_preview,
		     last_message.created_at AS last_message_at
		   FROM core.chats AS c
		   LEFT JOIN LATERAL (
		     SELECT left(m.content, 200) AS preview, m.created_at
		     FROM core.messages AS m
		     WHERE m.chat_id = c.id
		     ORDER BY m.id DESC
		     LIMIT 1
		   ) AS last_message ON true
		   WHERE c.user_id = $1 AND ($2::BIGINT IS NULL OR c.id < $2)
		   ORDER BY c.id DESC
		   LIMIT $3
		 ) AS chat_row`,
		userID,
		beforeID,
		limit+1,
	).Scan(&encodedChats)
	if err != nil {
		return nil, nil, err
	}

	chats, err := decodeChatList(encodedChats)
	if err != nil {
		return nil, nil, err
	}

	var nextCursor *string
	if len(chats) > limit {
		chats = chats[:limit]
		cursor := strconv.FormatInt(chats[len(chats)-1].ChatID, 10)
		nextCursor = &cursor
	}
	return chats, nextCursor, nil
}

func setNullableChatFields(chat *chatRecord, companyID sql.NullInt64, chatName sql.NullString) {
	if companyID.Valid {
		chat.CompanyID = &companyID.Int64
	}
	if chatName.Valid {
		chat.ChatName = &chatName.String
	}
}

func decodeChatList(data []byte) ([]chatListItem, error) {
	chats := make([]chatListItem, 0)
	err := json.Unmarshal(data, &chats)
	return chats, err
}
