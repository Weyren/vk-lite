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

	db := infra.NewPostgres(cfg)
	redisClient := infra.NewRedis(cfg)
	events := infra.NewRabbitPublisher(cfg)

	userRepo := repo.NewUserRepo(db)
	authSvc := service.NewAuthService(userRepo, cfg)

	authH := handler.NewAuthHandler(authSvc)
	socialH := handler.NewSocialHandler(db, redisClient, events)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	public := r.Group("/api/v1")
	public.POST("/users", authH.Register)
	public.POST("/auth/register", authH.Register)
	public.POST("/auth/login", authH.Login)

	protected := r.Group("/api/v1")
	protected.Use(middleware.JWT(cfg.JWTSecret))
	protected.GET("/users/:id", socialH.GetUser)
	protected.POST("/users/:id/follow", socialH.ToggleFollow)
	protected.POST("/posts", socialH.CreatePost)
	protected.POST("/posts/:id/like", socialH.ToggleLike)
	protected.GET("/feed", socialH.GetFeed)

	r.GET("/healthz", func(c *gin.Context) { c.String(200, "OK") })
	r.GET("/ready", func(c *gin.Context) { c.String(200, "OK") })

	addr := ":" + cfg.Port
	log.Printf("starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("gin run error: %v", err)
	}
}
