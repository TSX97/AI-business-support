package corehttp

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

type postgresDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Dependencies struct {
	Postgres postgresDB
	Redis    *redis.Client
	NATS     *nats.Conn
}

type API struct {
	postgres postgresDB
	redis    *redis.Client
	nats     *nats.Conn
}

func NewRouter(dependencies Dependencies) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	api := &API{
		postgres: dependencies.Postgres,
		redis:    dependencies.Redis,
		nats:     dependencies.NATS,
	}

	router.GET("/", api.root)
	router.GET("/health", api.health)
	router.POST("/api", api.hello)

	router.POST("/chat", api.createChat)
	router.GET("/chat/:id", api.getChat)
	router.PUT("/chat/:id", api.renameChat)
	router.DELETE("/chat/:id", api.deleteChat)
	router.GET("/chats", api.listChats)

	return router
}
