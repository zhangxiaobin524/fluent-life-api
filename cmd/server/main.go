package main

import (
	"log"
	"os"

	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/handlers"
	"fluent-life-backend/internal/hub"
	"fluent-life-backend/internal/middleware"
	"fluent-life-backend/internal/models"
	"fluent-life-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func main() {
	// 确保所有日志输出到标准输出和标准错误（都会被重定向到日志文件）
	// 这样 log.Printf 和 zap logger 都会输出到同一个文件
	// 使用无缓冲输出，确保日志立即写入
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	// 测试日志输出
	log.Printf("========== 后端服务启动 ==========")
	os.Stdout.Sync()
	
	// 同时设置标准错误输出到标准输出，确保所有日志都在一个文件
	// 注意：这需要在重定向之前设置，但我们已经用 2>&1 重定向了
	
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
	userHandler := handlers.NewUserHandler(db, cfg)
	trainingHandler := handlers.NewTrainingHandler(db, cfg)
	aiHandler := handlers.NewAIHandler(db, cfg)
	communityHandler := handlers.NewCommunityHandler(db)
	achievementHandler := handlers.NewAchievementHandler(db)
	practiceRoomHandler := handlers.NewPracticeRoomHandler(db, roomHub)
	wsHandler := handlers.NewWebSocketHandler(roomHub, db, cfg.JWTSecret)
	followHandler := handlers.NewFollowHandler(db)
	collectionHandler := handlers.NewCollectionHandler(db)

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
				users.GET("/:id", userHandler.GetUserProfileByID)
			}

			// 训练记录
			training := authenticated.Group("/training")
			{
				training.POST("/records", trainingHandler.CreateRecord)
				training.GET("/records", trainingHandler.GetRecords)
				training.GET("/stats", trainingHandler.GetStats)
				training.GET("/meditation-progress", trainingHandler.GetMeditationProgress)
				training.GET("/weekly-stats", trainingHandler.GetWeeklyStats)
				training.GET("/skill-levels", trainingHandler.GetSkillLevels)
				training.GET("/recommendations", trainingHandler.GetRecommendations)
				training.GET("/progress-trend", trainingHandler.GetProgressTrend)
				training.GET("/learning-partner-stats", trainingHandler.GetLearningPartnerStats)
				training.GET("/learning-partners", trainingHandler.GetLearningPartners)
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
				community.DELETE("/posts/:id", communityHandler.DeletePost)
				community.GET("/users/:id/posts", communityHandler.GetUserPosts)
				community.DELETE("/comments/:id", communityHandler.DeleteComment)
				community.POST("/comments/:id/like", communityHandler.ToggleCommentLike)
				community.GET("/users/:id/follow-status", communityHandler.CheckUserFollowStatus)
				community.GET("/posts/:id/collection-status", communityHandler.CheckPostCollectionStatus)
				community.GET("/users/:id/follow-count", communityHandler.GetUserFollowCount)
				community.GET("/posts/:id/collection-count", communityHandler.GetPostCollectionCount)
			}

			// 关注功能
			follow := authenticated.Group("/follow")
			{
				follow.POST("/users/:id", followHandler.FollowUser)
				follow.DELETE("/users/:id", followHandler.UnfollowUser)
				follow.GET("/users/:id/status", followHandler.CheckFollowStatus)
				follow.GET("/users/:id/followers", followHandler.GetFollowers)
				follow.GET("/users/:id/following", followHandler.GetFollowing)
			}

			// 收藏功能
			collection := authenticated.Group("/collection")
			{
				collection.POST("/posts/:id", collectionHandler.CollectPost)
				collection.DELETE("/posts/:id", collectionHandler.UncollectPost)
				collection.GET("/posts", collectionHandler.GetCollectedPosts)
				collection.GET("/posts/:id/status", collectionHandler.CheckCollectionStatus)
				collection.POST("/toggle/:id", collectionHandler.ToggleCollectPost) // 新增的切换收藏状态路由
			}

			// 成就系统
			achievements := authenticated.Group("/achievements")
			{
				achievements.GET("", achievementHandler.GetAchievements)
			}

			// 对练房
			practiceRooms := authenticated.Group("/practice-rooms")
			{
				log.Printf("[路由注册] 注册对练房路由: POST /api/v1/practice-rooms")
				practiceRooms.POST("", practiceRoomHandler.CreateRoom)
				log.Printf("[路由注册] 注册对练房路由: GET /api/v1/practice-rooms")
				practiceRooms.GET("", practiceRoomHandler.GetRooms)
				log.Printf("[路由注册] 注册对练房路由: GET /api/v1/practice-rooms/:id")
				practiceRooms.GET("/:id", practiceRoomHandler.GetRoom)
				log.Printf("[路由注册] 注册对练房路由: POST /api/v1/practice-rooms/:id/join")
				practiceRooms.POST("/:id/join", practiceRoomHandler.JoinRoom)
				log.Printf("[路由注册] 注册对练房路由: POST /api/v1/practice-rooms/:id/leave")
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
