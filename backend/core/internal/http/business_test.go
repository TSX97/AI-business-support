package corehttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type businessTestDB struct {
	calls int
}

func (db *businessTestDB) QueryRow(context.Context, string, ...any) pgx.Row {
	db.calls++
	return fakeRow{err: pgx.ErrNoRows}
}

func TestBusinessEndpointsRequireSessionBeforeDatabaseAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/business",
			body:   `{"user_id":42,"business_name":"Northwind","business_data":{}}`,
		},
		{
			name:   "list",
			method: http.MethodGet,
			path:   "/business?user_id=42",
		},
		{
			name:   "update",
			method: http.MethodPut,
			path:   "/business",
			body:   `{"user_id":42,"business_id":51,"business_name":"Northwind","description":null,"business_data":{}}`,
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			path:   "/business",
			body:   `{"user_id":42,"business_id":51}`,
		},
		{
			name:   "add member",
			method: http.MethodPost,
			path:   "/business/member",
			body:   `{"user_id":42,"business_id":51,"member_user_id":77}`,
		},
		{
			name:   "remove member",
			method: http.MethodDelete,
			path:   "/business/member",
			body:   `{"user_id":42,"business_id":51,"member_user_id":77}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &businessTestDB{}
			router := NewRouter(Dependencies{Postgres: database, Redis: redisClient})
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Session-ID", "missing-session")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, response.Code, response.Body.String())
			}
			if database.calls != 0 {
				t.Fatalf("database must not be accessed without a valid session; got %d query calls", database.calls)
			}
		})
	}
}
