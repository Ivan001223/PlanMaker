package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/kaoyan/server/config"
	"github.com/kaoyan/server/internal/handler"
	"github.com/kaoyan/server/internal/middleware"
	"github.com/kaoyan/server/internal/model"
	"github.com/kaoyan/server/internal/pkg/llm"
	"github.com/kaoyan/server/internal/pkg/wechat"
	"github.com/kaoyan/server/internal/repository"
	"github.com/kaoyan/server/internal/router"
	"github.com/kaoyan/server/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	var logger *zap.Logger
	if cfg.App.Env == "production" {
		logger, _ = zap.NewProduction()
	} else {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	logger.Info("启动考研规划服务",
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
	)

	db, err := initDB(cfg)
	if err != nil {
		logger.Fatal("数据库连接失败", zap.Error(err))
	}
	logger.Info("数据库连接成功")

	if cfg.App.Env != "production" {
		if err := db.AutoMigrate(
			&model.User{},
			&model.UserPreference{},
			&model.StudyPlan{},
			&model.StudyTask{},
			&model.ChatSession{},
			&model.ChatMessage{},
			&model.Notification{},
			&model.Textbook{},
		); err != nil {
			logger.Warn("AutoMigrate失败", zap.Error(err))
		}
	}

	middleware.InitJWT(cfg.JWT.Secret)
	middleware.SetInternalAPIKey(cfg.InternalAPIKey)

	if cfg.App.CORSAllowedOrigins != "" {
		origins := strings.Split(cfg.App.CORSAllowedOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		middleware.SetCORSOrigins(origins)
	}

	aiClient := llm.NewClient(cfg.LLM.APIKey, cfg.LLM.BaseURL)
	wechatClient := wechat.NewClient(cfg.WeChat.AppID, cfg.WeChat.AppSecret)

	userRepo := repository.NewUserRepo(db)
	planRepo := repository.NewPlanRepo(db)
	taskRepo := repository.NewTaskRepo(db)
	chatRepo := repository.NewChatRepo(db)
	notifRepo := repository.NewNotificationRepo(db)

	userService := service.NewUserService(userRepo, wechatClient, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	plannerService := service.NewPlannerService(planRepo, taskRepo, userRepo, aiClient, cfg.LLM.PlanningModel, logger)
	chatService := service.NewChatService(chatRepo, aiClient, cfg.LLM.DefaultModel, logger)
	pdfService := service.NewPDFService(db, cfg.PDFService.BaseURL, logger)
	notifService := service.NewNotificationService(notifRepo, logger)

	userHandler := handler.NewUserHandler(userService)
	planHandler := handler.NewPlanHandler(plannerService)
	chatHandler := handler.NewChatHandler(chatService)
	pdfHandler := handler.NewPDFHandler(pdfService)
	notifHandler := handler.NewNotificationHandler(notifService)

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()

	r := router.NewRouter(userHandler, planHandler, chatHandler, pdfHandler, notifHandler, db, logger)
	r.Setup(engine)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go startDailyCron(ctx, plannerService, logger)

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{Addr: addr, Handler: engine}

	go func() {
		logger.Info("HTTP 服务启动", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务启动失败", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("收到关闭信号，优雅关闭中...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("优雅关闭失败", zap.Error(err))
	}
	logger.Info("服务已关闭")
}

func startDailyCron(ctx context.Context, plannerService *service.PlannerService, logger *zap.Logger) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		logger.Info("计划刷新定时任务已注册", zap.Time("next_run", next), zap.Duration("wait", next.Sub(now)))

		select {
		case <-ctx.Done():
			logger.Info("定时任务已停止")
			return
		case <-time.After(next.Sub(now)):
		}

		logger.Info("开始执行每日计划刷新")
		plans, err := plannerService.GetActivePlans()
		if err != nil {
			logger.Error("获取活跃计划失败", zap.Error(err))
			continue
		}

		const maxConcurrency = 10
		sem := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup
		var successCount, failCount int64

		for _, plan := range plans {
			sem <- struct{}{}
			wg.Add(1)

			go func(planID uint64, planName string) {
				defer wg.Done()
				defer func() { <-sem }()

				refreshCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
				defer cancel()

				done := make(chan struct{})
				go func(innerCtx context.Context) {
					defer close(done)
					select {
					case <-innerCtx.Done():
						return
					default:
						logger.Info("刷新计划", zap.Uint64("plan_id", planID), zap.String("plan_name", planName))
						if _, err := plannerService.RefreshPlan(planID); err != nil {
							logger.Error("刷新计划失败", zap.Uint64("plan_id", planID), zap.Error(err))
							atomic.AddInt64(&failCount, 1)
						} else {
							atomic.AddInt64(&successCount, 1)
						}
					}
				}(refreshCtx)

				select {
				case <-done:
				case <-refreshCtx.Done():
					logger.Error("刷新计划超时", zap.Uint64("plan_id", planID))
					atomic.AddInt64(&failCount, 1)
				}
			}(plan.ID, plan.PlanName)
		}

		wg.Wait()
		logger.Info("每日计划刷新完成",
			zap.Int64("success", atomic.LoadInt64(&successCount)),
			zap.Int64("fail", atomic.LoadInt64(&failCount)),
			zap.Int("total", len(plans)),
		)
	}
}

func initDB(cfg *config.Config) (*gorm.DB, error) {
	logLevel := gormlogger.Info
	if cfg.App.Env == "production" {
		logLevel = gormlogger.Warn
	}

	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)

	return db, nil
}
