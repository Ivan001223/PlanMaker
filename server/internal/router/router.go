package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kaoyan/server/internal/handler"
	"github.com/kaoyan/server/internal/middleware"
	"github.com/kaoyan/server/internal/pkg/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Router struct {
	userHandler  *handler.UserHandler
	planHandler  *handler.PlanHandler
	chatHandler  *handler.ChatHandler
	pdfHandler   *handler.PDFHandler
	notifHandler *handler.NotificationHandler
	db           *gorm.DB
	logger       *zap.Logger
}

func NewRouter(
	userHandler *handler.UserHandler,
	planHandler *handler.PlanHandler,
	chatHandler *handler.ChatHandler,
	pdfHandler *handler.PDFHandler,
	notifHandler *handler.NotificationHandler,
	db *gorm.DB,
	logger *zap.Logger,
) *Router {
	return &Router{
		userHandler:  userHandler,
		planHandler:  planHandler,
		chatHandler:  chatHandler,
		pdfHandler:   pdfHandler,
		notifHandler: notifHandler,
		db:           db,
		logger:       logger,
	}
}

func (r *Router) Setup(engine *gin.Engine) {
	// 全局中间件
	engine.Use(
		middleware.CORSMiddleware(),
		middleware.LoggerMiddleware(r.logger),
		middleware.RecoveryMiddleware(r.logger),
	)

	// 全局限流器
	apiLimiter := middleware.NewRateLimiter(10, 30)
	strictLimiter := middleware.NewRateLimiter(1, 5)

	// 健康检查
	engine.GET("/health", func(c *gin.Context) {
		status := "healthy"
		details := gin.H{}

		sqlDB, err := r.db.DB()
		if err != nil || sqlDB.Ping() != nil {
			status = "degraded"
			details["db"] = "unreachable"
		} else {
			details["db"] = "ok"
		}

		if status == "healthy" {
			response.OK(c, gin.H{"status": status, "components": details})
		} else {
			c.JSON(503, response.Response{Code: 503, Message: status, Data: gin.H{"components": details}})
		}
	})

	// API v1
	v1 := engine.Group("/api/v1")
	v1.Use(middleware.RateLimitMiddleware(apiLimiter))
	{
		// 认证 (无需登录，严格限流)
		auth := v1.Group("/auth")
		auth.Use(middleware.RateLimitMiddleware(strictLimiter))
		{
			auth.POST("/login", r.userHandler.Login)
		}

		// 内部回调 (需要 API Key 保护)
		internal := v1.Group("/internal")
		internal.Use(middleware.InternalAPIMiddleware())
		{
			internal.POST("/textbooks/parse-callback", r.pdfHandler.ParseCallback)
		}

		// 以下接口需要 JWT 认证
		authorized := v1.Group("")
		authorized.Use(middleware.AuthMiddleware())
		{
			// 用户
			user := authorized.Group("/user")
			{
				user.GET("/profile", r.userHandler.GetProfile)
				user.PUT("/profile", r.userHandler.UpdateProfile)
				user.GET("/preference", r.userHandler.GetPreference)
				user.PUT("/preference", r.userHandler.UpdatePreference)
			}

			// 学习计划
			plans := authorized.Group("/plans")
			{
				plans.POST("/generate", middleware.RateLimitMiddleware(strictLimiter), r.planHandler.GeneratePlan)
				plans.GET("", r.planHandler.ListPlans)
				plans.GET("/:id", r.planHandler.GetPlan)
				plans.POST("/:id/refresh", r.planHandler.RefreshPlan)
				plans.GET("/today", r.planHandler.GetTodayTasks)
				plans.GET("/week", r.planHandler.GetWeekTasks)
			}

			// 任务
			tasks := authorized.Group("/tasks")
			{
				tasks.PATCH("/:id/status", r.planHandler.UpdateTaskStatus)
			}

			// AI 对话
			chat := authorized.Group("/chat")
			{
				chat.POST("/sessions", r.chatHandler.CreateSession)
				chat.GET("/sessions", r.chatHandler.ListSessions)
				chat.GET("/sessions/:id", r.chatHandler.GetSession)
				chat.POST("/sessions/:id/messages", middleware.RateLimitMiddleware(strictLimiter), r.chatHandler.SendMessage)
			}

			// 教材/PDF
			textbooks := authorized.Group("/textbooks")
			{
				textbooks.POST("", middleware.RateLimitMiddleware(strictLimiter), r.pdfHandler.UploadPDF)
				textbooks.GET("", r.pdfHandler.ListTextbooks)
			}

			// 通知
			notifications := authorized.Group("/notifications")
			{
				notifications.GET("", r.notifHandler.ListNotifications)
				notifications.POST("/generate", r.notifHandler.GenerateNotifications)
			}
		}
	}
}
