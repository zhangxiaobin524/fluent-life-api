package main

import (
	"log"

	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/handlers"
	"fluent-life-backend/internal/hub"
	"fluent-life-backend/internal/middleware"
	"fluent-life-backend/internal/models"
	"fluent-life-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 自动迁移数据库
	if err := models.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 设置 Gin 模式
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 路由
	r := gin.New()

	// 全局中间件
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"}, "服务运行正常")
	})

	// 初始化 WebSocket Hub
	roomHub := hub.NewRoomHub()
	go roomHub.Run()

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(db, cfg)
	userHandler := handlers.NewUserHandler(db)
	trainingHandler := handlers.NewTrainingHandler(db, cfg)
	aiHandler := handlers.NewAIHandler(db, cfg)
	communityHandler := handlers.NewCommunityHandler(db)
	achievementHandler := handlers.NewAchievementHandler(db)
	practiceRoomHandler := handlers.NewPracticeRoomHandler(db, roomHub)
	wsHandler := handlers.NewWebSocketHandler(roomHub, db, cfg.JWTSecret)

	// API 路由组
	v1 := r.Group("/api/v1")
	{
		// 认证相关（无需认证）
		auth := v1.Group("/auth")
		{
			auth.POST("/send-code", authHandler.SendCode)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// 需要认证的路由
		authenticated := v1.Group("")
		authenticated.Use(middleware.Auth(cfg.JWTSecret))
		{
			// 用户相关
			users := authenticated.Group("/users")
			{
				users.GET("/profile", userHandler.GetProfile)
				users.PUT("/profile", userHandler.UpdateProfile)
				users.GET("/stats", userHandler.GetStats)
			}

			// 训练记录
			training := authenticated.Group("/training")
			{
				training.POST("/records", trainingHandler.CreateRecord)
				training.GET("/records", trainingHandler.GetRecords)
				training.GET("/stats", trainingHandler.GetStats)
				training.GET("/meditation-progress", trainingHandler.GetMeditationProgress)
			}

			// AI 导师
			ai := authenticated.Group("/ai")
			{
				ai.POST("/chat", aiHandler.Chat)
				ai.GET("/conversation", aiHandler.GetConversation)
				ai.POST("/analyze-speech", aiHandler.AnalyzeSpeech)
			}

			// 社区
			community := authenticated.Group("/community")
			{
				community.GET("/posts", communityHandler.GetPosts)
				community.POST("/posts", communityHandler.CreatePost)
				community.GET("/posts/:id", communityHandler.GetPost)
				community.POST("/posts/:id/like", communityHandler.ToggleLike)
				community.GET("/posts/:id/comments", communityHandler.GetComments)
				community.POST("/posts/:id/comments", communityHandler.CreateComment)
			}

			// 成就系统
			achievements := authenticated.Group("/achievements")
			{
				achievements.GET("", achievementHandler.GetAchievements)
			}

			// 对练房
			practiceRooms := authenticated.Group("/practice-rooms")
			{
				practiceRooms.POST("", practiceRoomHandler.CreateRoom)
				practiceRooms.GET("", practiceRoomHandler.GetRooms)
				practiceRooms.GET("/:id", practiceRoomHandler.GetRoom)
				practiceRooms.POST("/:id/join", practiceRoomHandler.JoinRoom)
				practiceRooms.POST("/:id/leave", practiceRoomHandler.LeaveRoom)
			}
		}

		// WebSocket 连接（独立处理，不经过认证中间件，因为它自己处理 token 验证）
		v1.GET("/ws", wsHandler.HandleWebSocket)
	}

	// 启动服务器
	addr := ":" + cfg.Port
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
