package corehttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type fakeChatDB struct {
	called    bool
	userID    int64
	chatName  string
	companyID sql.NullInt64
}

func (db *fakeChatDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.called = true
	if !strings.Contains(query, "INSERT INTO core.chats (user_id, title)") {
		return fakeRow{err: pgx.ErrNoRows}
	}
	db.userID = args[0].(int64)
	db.chatName = args[1].(string)
	return fakeRow{
		chatID:    103,
		userID:    db.userID,
		companyID: db.companyID,
		title:     sql.NullString{String: db.chatName, Valid: true},
		createdAt: time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC),
	}
}

type fakeRow struct {
	chatID    int64
	userID    int64
	companyID sql.NullInt64
	title     sql.NullString
	createdAt time.Time
	err       error
}

func (row fakeRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*dest[0].(*int64) = row.chatID
	*dest[1].(*int64) = row.userID
	*dest[2].(*sql.NullInt64) = row.companyID
	*dest[3].(*sql.NullString) = row.title
	*dest[4].(*time.Time) = row.createdAt
	return nil
}

type fakeGetChatDB struct {
	called bool
	query  string
	args   []any
	row    pgx.Row
}

func (db *fakeGetChatDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.called = true
	db.query = query
	db.args = args
	return db.row
}

type fakeGetChatRow struct {
	chatID    int64
	userID    int64
	companyID sql.NullInt64
	title     sql.NullString
	createdAt time.Time
	err       error
}

type fakeJSONRow struct {
	data []byte
	err  error
}

type fakeDeleteChatRow struct {
	chatID int64
	err    error
}

func (row fakeDeleteChatRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*dest[0].(*int64) = row.chatID
	return nil
}

func (row fakeJSONRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*dest[0].(*[]byte) = row.data
	return nil
}

func (row fakeGetChatRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*dest[0].(*int64) = row.chatID
	*dest[1].(*int64) = row.userID
	*dest[2].(*sql.NullInt64) = row.companyID
	*dest[3].(*sql.NullString) = row.title
	*dest[4].(*time.Time) = row.createdAt
	return nil
}

func TestCreateChatValidatesSessionAndInsertsChat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Set(context.Background(), "session:session-abc", "42", 0).Err(); err != nil {
		t.Fatal(err)
	}

	database := &fakeChatDB{}
	router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
	request := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(
		`{"user_id":42,"session_id":"session-abc","chat_name":" sales "}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	if !database.called {
		t.Fatal("expected chat insert to be called")
	}
	if database.userID != 42 || database.chatName != "sales" {
		t.Fatalf("unexpected insert args: user_id=%d chat_name=%q", database.userID, database.chatName)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			ChatID   int64  `json:"chat_id"`
			UserID   int64  `json:"user_id"`
			ChatName string `json:"chat_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success || body.Data.ChatID != 103 || body.Data.UserID != 42 || body.Data.ChatName != "sales" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestRenameChatValidatesSessionAndUpdatesOwnedChat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Set(context.Background(), "session:session-abc", "42", 0).Err(); err != nil {
		t.Fatal(err)
	}

	database := &fakeGetChatDB{row: fakeGetChatRow{
		chatID:    103,
		userID:    42,
		title:     sql.NullString{String: "sales-q4", Valid: true},
		createdAt: time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC),
	}}
	router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
	request := httptest.NewRequest(http.MethodPut, "/chat/103?user_id=42", strings.NewReader(
		`{"new_chat_name":"sales-q4"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Session-ID", "session-abc")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if !strings.Contains(database.query, "UPDATE core.chats SET title") {
		t.Fatalf("expected update query, got %q", database.query)
	}
	if len(database.args) != 3 || database.args[0] != int64(103) ||
		database.args[1] != int64(42) || database.args[2] != "sales-q4" {
		t.Fatalf("unexpected update args: %#v", database.args)
	}
}

func TestListChatsValidatesSessionAndReturnsCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Set(context.Background(), "session:session-abc", "42", 0).Err(); err != nil {
		t.Fatal(err)
	}

	database := &fakeGetChatDB{row: fakeJSONRow{data: []byte(`[
		{"chat_id":103,"user_id":42,"company_id":null,"chat_name":"newest","created_at":"2026-11-11T10:00:00Z","last_message_preview":null,"last_message_at":null},
		{"chat_id":102,"user_id":42,"company_id":null,"chat_name":"older","created_at":"2026-11-10T10:00:00Z","last_message_preview":"hello","last_message_at":"2026-11-10T10:01:00Z"}
	]`)}}
	router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
	request := httptest.NewRequest(http.MethodGet, "/chats?user_id=42&limit=1", nil)
	request.Header.Set("X-Session-ID", "session-abc")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if len(database.args) != 3 || database.args[0] != int64(42) ||
		database.args[1] != (*int64)(nil) || database.args[2] != 2 {
		t.Fatalf("unexpected list query args: %#v", database.args)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Chats      []chatListItem `json:"chats"`
			NextCursor *string        `json:"next_cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success || len(body.Data.Chats) != 1 || body.Data.Chats[0].ChatID != 103 ||
		body.Data.NextCursor == nil || *body.Data.NextCursor != "103" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestDeleteChatValidatesSessionAndDeletesOwnedChat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Set(context.Background(), "session:session-abc", "42", 0).Err(); err != nil {
		t.Fatal(err)
	}

	database := &fakeGetChatDB{row: fakeDeleteChatRow{chatID: 103}}
	router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
	request := httptest.NewRequest(http.MethodDelete, "/chat/103?user_id=42", nil)
	request.Header.Set("X-Session-ID", "session-abc")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if database.query != `DELETE FROM core.chats WHERE id = $1 AND user_id = $2 RETURNING id` {
		t.Fatalf("unexpected delete query: %q", database.query)
	}
	if len(database.args) != 2 || database.args[0] != int64(103) || database.args[1] != int64(42) {
		t.Fatalf("unexpected delete args: %#v", database.args)
	}
	if !strings.Contains(response.Body.String(), `"chat_id":103`) {
		t.Fatalf("expected deleted chat ID in response, got %s", response.Body.String())
	}
}

func TestDeleteChatRejectsMissingSessionWithoutDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	database := &fakeGetChatDB{row: fakeDeleteChatRow{chatID: 103}}
	router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
	request := httptest.NewRequest(http.MethodDelete, "/chat/103?user_id=42", nil)
	request.Header.Set("X-Session-ID", "missing-session")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
	if database.called {
		t.Fatal("chat delete must not run when the session is missing")
	}
}

func TestGetChatValidatesSessionAndSelectsChat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Set(context.Background(), "session:session-abc", "42", 0).Err(); err != nil {
		t.Fatal(err)
	}

	createdAt := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	database := &fakeGetChatDB{row: fakeGetChatRow{
		chatID:    103,
		userID:    42,
		companyID: sql.NullInt64{Int64: 7, Valid: true},
		title:     sql.NullString{String: "sales", Valid: true},
		createdAt: createdAt,
	}}
	router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
	request := httptest.NewRequest(http.MethodGet, "/chat/103?user_id=42", nil)
	request.Header.Set("X-Session-ID", "session-abc")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if !database.called {
		t.Fatal("expected chat select to be called")
	}
	if database.query != `SELECT * FROM core.chats WHERE id = $1` {
		t.Fatalf("unexpected query: %q", database.query)
	}
	if len(database.args) != 1 || database.args[0] != int64(103) {
		t.Fatalf("unexpected query args: %#v", database.args)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			ChatID    int64     `json:"chat_id"`
			UserID    int64     `json:"user_id"`
			CompanyID int64     `json:"company_id"`
			ChatName  string    `json:"chat_name"`
			CreatedAt time.Time `json:"created_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success || body.Data.ChatID != 103 || body.Data.UserID != 42 ||
		body.Data.CompanyID != 7 || body.Data.ChatName != "sales" || !body.Data.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestGetChatRejectsMissingSessionWithoutSelect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	database := &fakeGetChatDB{row: fakeGetChatRow{err: pgx.ErrNoRows}}
	router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
	request := httptest.NewRequest(http.MethodGet, "/chat/103?user_id=42", nil)
	request.Header.Set("X-Session-ID", "missing-session")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
	if database.called {
		t.Fatal("chat select must not run when the session is missing")
	}
}

func TestGetChatHidesChatOwnedByDifferentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Set(context.Background(), "session:session-abc", "42", 0).Err(); err != nil {
		t.Fatal(err)
	}

	database := &fakeGetChatDB{row: fakeGetChatRow{
		chatID: 103,
		userID: 99,
		title:  sql.NullString{String: "private", Valid: true},
	}}
	router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
	request := httptest.NewRequest(http.MethodGet, "/chat/103?user_id=42", nil)
	request.Header.Set("X-Session-ID", "session-abc")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestCreateChatRejectsMissingSessionWithoutInsert(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	database := &fakeChatDB{}
	router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
	request := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(
		`{"user_id":42,"session_id":"missing-session","chat_name":"sales"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
	if database.called {
		t.Fatal("chat insert must not run for a missing session")
	}
}

func TestCreateChatRejectsSessionForDifferentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Set(context.Background(), "session:session-abc", "99", 0).Err(); err != nil {
		t.Fatal(err)
	}

	database := &fakeChatDB{}
	router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
	request := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(
		`{"user_id":42,"session_id":"session-abc","chat_name":"sales"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.Code)
	}
	if database.called {
		t.Fatal("chat insert must not run when the session belongs to another user")
	}
}
