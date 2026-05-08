// cmd/vk-lite/main.go
package main

import (
	"log"

	"github.com/Weyren/vk-lite/internal/handler"
	"github.com/Weyren/vk-lite/internal/infra"
	"github.com/Weyren/vk-lite/internal/middleware"
	"github.com/Weyren/vk-lite/internal/repo"
	"github.com/Weyren/vk-lite/internal/service"
	"github.com/Weyren/vk-lite/pkg/utils"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := utils.NewConfig()

	// инфраструктура
	db := infra.NewPostgres(cfg)
	//redisClient := infra.NewRedis(cfg) // пока не используется, но уже готов

	// репозитории
	userRepo := repo.NewUserRepo(db)

	// сервисы
	authSvc := service.NewAuthService(userRepo, cfg)

	// хендлеры
	authH := handler.NewAuthHandler(authSvc)

	// роутер Gin
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// публичные роуты
	public := r.Group("/api/v1")
	public.POST("/auth/register", authH.Register)
	public.POST("/auth/login", authH.Login)

	// защищённые роуты (пока пусто)
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWT(cfg.JWTSecret))
	// сюда дальше будут post‑/like‑/feed‑хендлеры

	// health‑checks
	r.GET("/healthz", func(c *gin.Context) { c.String(200, "OK") })
	r.GET("/ready", func(c *gin.Context) { c.String(200, "OK") })

	addr := ":" + cfg.Port
	log.Printf("starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("gin run error: %v", err)
	}
}
