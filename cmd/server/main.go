package main

import (
	"Test/config"
	"Test/internal/auth"
	"Test/internal/handler"
	"Test/internal/middleware"
	"Test/internal/repository"
	"Test/internal/service"
	"Test/internal/storage"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.LoadConfig("./config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := storage.New(cfg)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate("file://migrations", cfg); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	userRepo := repository.NewUserRepo(db.DB)
	taskRepo := repository.NewTaskRepo(db.DB)

	userService := service.NewUserService(userRepo, taskRepo)
	jwtService := middleware.NewJWTService(cfg.JWT.SecretKey)

	userHandler := handler.NewUserHandler(userService)
	authHandler := auth.NewAuthHandler(userRepo, cfg)

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	api := router.Group("/api")
	{
		api.POST("/register", authHandler.RegisterHandler)
		api.POST("/login", authHandler.LoginHandler)

		authorized := api.Group("")
		authorized.Use(middleware.AuthMiddleware(jwtService))
		{
			users := authorized.Group("/users")
			{
				users.GET("/:id/status", userHandler.GetUserStatus)
				users.POST("/:id/task/complete", userHandler.CompleteTask)
				users.POST("/:id/referrer", userHandler.SetReferrer)
				users.GET("/leaderboard", userHandler.GetLeaderboard)
			}
		}
	}

	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("start server: %v", err)
	}
}
