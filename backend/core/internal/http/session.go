package corehttp

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var (
	errSessionNotFound     = errors.New("session does not exist or has expired")
	errSessionUserMismatch = errors.New("session does not belong to user_id")
	errSessionUnavailable  = errors.New("session store is unavailable")
)

func (api *API) validateSession(ctx context.Context, userID int64, sessionID string) error {
	if !validSessionID(sessionID) {
		return errSessionNotFound
	}
	if api.redis == nil {
		return errSessionUnavailable
	}

	sessionUserID, err := api.redis.Get(ctx, "session:"+sessionID).Result()
	if errors.Is(err, redis.Nil) {
		return errSessionNotFound
	}
	if err != nil {
		return errors.Join(errSessionUnavailable, err)
	}
	if sessionUserID != strconv.FormatInt(userID, 10) {
		return errSessionUserMismatch
	}
	return nil
}

func (api *API) requireSession(c *gin.Context, userID int64, sessionID string) bool {
	if err := api.validateSession(c.Request.Context(), userID, sessionID); err != nil {
		writeSessionError(c, err)
		return false
	}
	return true
}

func validSessionID(sessionID string) bool {
	return sessionID != "" && len(sessionID) <= 256 && strings.TrimSpace(sessionID) == sessionID
}
