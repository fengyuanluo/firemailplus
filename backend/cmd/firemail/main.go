package main

import (
	"context"
	"log"

	"firemail/internal/config"
	"firemail/internal/database"
	"firemail/internal/handlers"
	"firemail/internal/middleware"
	"firemail/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量 - 优先加载.env.local，然后是.env
	if err := godotenv.Load(".env.local"); err != nil {
		// 如果.env.local不存在，尝试加载.env
		if err := godotenv.Load(".env"); err != nil {
			log.Println("Warning: No .env file found, using system environment variables")
		} else {
			log.Println("Loaded configuration from .env file")
		}
	} else {
		log.Println("Loaded configuration from .env.local file")
	}

	// 初始化配置
	cfg := config.Load()

	// 初始化数据库
	db, err := database.Initialize(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 设置Gin模式
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	router := gin.New()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS(cfg.CORS.Origins))

	// 初始化处理器
	h := handlers.New(db, cfg)

	// 设置全局认证服务（用于向后兼容）
	log.Println("Setting global auth service...")
	middleware.SetGlobalAuthService(h.GetAuthService())
	log.Println("Global auth service set successfully")

	// 启动SSE服务
	if err := h.StartSSEService(); err != nil {
		log.Fatalf("Failed to start SSE service: %v", err)
	}

	// 验证软删除查询行为
	if err := services.ValidateSoftDeleteQueries(db); err != nil {
		log.Printf("Warning: Soft delete validation failed: %v", err)
	}

	// 启动自动备份服务
	if err := h.StartBackupService(context.Background()); err != nil {
		log.Printf("Warning: Failed to start backup service: %v", err)
	}

	// 启动软删除自动清理服务（保留30天）
	if err := h.StartSoftDeleteCleanup(context.Background(), 30); err != nil {
		log.Printf("Warning: Failed to start soft delete cleanup service: %v", err)
	}

	// 启动临时附件自动清理服务（保留24小时）
	if err := h.StartTemporaryAttachmentCleanup(context.Background(), 24); err != nil {
		log.Printf("Warning: Failed to start temporary attachment cleanup service: %v", err)
	}

	// 启动定时邮件服务
	if err := h.StartScheduledEmailService(context.Background()); err != nil {
		log.Printf("Warning: Failed to start scheduled email service: %v", err)
	}

	// 设置路由
	setupRoutes(router, h)

	// 启动服务器
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("FireMail server starting on %s", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(router *gin.Engine, h *handlers.Handler) {
	// 健康检查
	router.GET("/health", h.HealthCheck)

	// API路由组
	api := router.Group("/api/v1")
	{
		// 认证路由
		auth := api.Group("/auth")
		{
			auth.POST("/login", h.Login)
			auth.POST("/logout", h.Logout)
			auth.GET("/me", h.AuthRequired(), h.GetCurrentUser)
		}

		// OAuth2认证路由
		oauth := api.Group("/oauth")
		{
			// OAuth2初始化端点（不需要认证，因为是OAuth流程的开始）
			oauth.GET("/gmail", h.InitGmailOAuth)
			oauth.GET("/outlook", h.InitOutlookOAuth)

			// OAuth2回调处理端点（不需要认证，因为是OAuth流程的一部分）
			oauth.GET("/:provider/callback", h.HandleOAuth2Callback)

			// OAuth2账户创建端点（需要认证）
			oauth.POST("/create-account", h.AuthRequired(), h.CreateOAuth2Account)

			// 手动OAuth2配置端点（需要认证）
			oauth.POST("/manual-config", h.AuthRequired(), h.CreateManualOAuth2Account)

			// 注意：移除了通用OAuth端点以避免路由冲突
			// 如果需要支持其他provider，请添加具体的路由
		}

		// 邮件账户管理路由（需要认证）
		accounts := api.Group("/accounts")
		accounts.Use(h.AuthRequired())
		{
			accounts.GET("", h.GetEmailAccounts)
			accounts.POST("", h.CreateEmailAccount)
			accounts.POST("/custom", h.CreateCustomEmailAccount) // 自定义邮箱创建端点
			accounts.GET("/:id", h.GetEmailAccount)
			accounts.PUT("/:id", h.UpdateEmailAccount)
			accounts.DELETE("/:id", h.DeleteEmailAccount)
			accounts.POST("/:id/test", h.TestEmailAccount)
			accounts.POST("/:id/sync", h.SyncEmailAccount)
		}

		// 提供商配置路由（需要认证）
		providers := api.Group("/providers")
		providers.Use(h.AuthRequired())
		{
			providers.GET("", h.GetProviders)
			providers.GET("/detect", h.DetectProvider)
		}

		// 邮件管理路由（需要认证）
		emails := api.Group("/emails")
		emails.Use(h.AuthRequired())
		{
			emails.GET("", h.GetEmails)
			emails.GET("/search", h.SearchEmails)
			emails.GET("/:id", h.GetEmail)
			emails.PATCH("/:id", h.UpdateEmail)
			emails.POST("/send", h.SendEmail)
			emails.DELETE("/:id", h.DeleteEmail)
			emails.PUT("/:id/read", h.MarkEmailAsRead)
			emails.PUT("/:id/unread", h.MarkEmailAsUnread)
			emails.PUT("/:id/star", h.ToggleEmailStar)
			emails.PUT("/:id/move", h.MoveEmail)
			emails.PUT("/:id/archive", h.ArchiveEmail)
			emails.POST("/:id/reply", h.ReplyEmail)
			emails.POST("/:id/reply-all", h.ReplyAllEmail)
			emails.POST("/:id/forward", h.ForwardEmail)
			emails.POST("/batch", h.BatchEmailOperations)
		}

		// 邮件文件夹路由（需要认证）
		folders := api.Group("/folders")
		folders.Use(h.AuthRequired())
		{
			folders.GET("", h.GetFolders)
			folders.POST("", h.CreateFolder)
			folders.GET("/:id", h.GetFolder)
			folders.PUT("/:id", h.UpdateFolder)
			folders.DELETE("/:id", h.DeleteFolder)
			folders.PUT("/:id/mark-read", h.MarkFolderAsRead)
			folders.PUT("/:id/sync", h.SyncFolder)
		}

		// 附件处理路由（需要认证）
		// 创建附件存储配置
		attachmentStorageConfig := &services.AttachmentStorageConfig{
			BaseDir:      "attachments",
			MaxFileSize:  25 * 1024 * 1024, // 25MB
			CompressText: true,
			CreateDirs:   true,
			ChecksumType: "md5",
		}

		// 创建附件存储
		attachmentStorage := services.NewLocalFileStorage(attachmentStorageConfig)

		// 创建附件服务
		attachmentService := services.NewAttachmentService(h.GetDB(), attachmentStorage, h.GetProviderFactory())

		// 创建附件处理器
		attachmentHandler := handlers.NewAttachmentHandler(attachmentService, h.GetDB())

		// 注册附件路由
		attachmentHandler.RegisterRoutes(api)

		// SSE路由（SSE端点有自己的认证逻辑）
		sse := api.Group("/sse")
		{
			sse.GET("", h.HandleSSE)                    // 主SSE端点，支持token参数
			sse.GET("/events", h.HandleSSE)             // 保持向后兼容
			sse.GET("/stats", h.AuthRequired(), h.GetSSEStats)     // 统计需要认证
			sse.POST("/test", h.AuthRequired(), h.SendTestEvent)   // 测试需要认证
		}
	}

	// 静态文件服务
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")

	// 前端路由（如果有）
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "FireMail",
		})
	})

	// SSE演示页面
	router.GET("/sse-demo", func(c *gin.Context) {
		c.HTML(200, "sse-demo.html", gin.H{
			"title": "FireMail SSE Demo",
		})
	})
}
