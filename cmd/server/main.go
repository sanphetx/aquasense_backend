package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aquasense-backend/internal/config"
	"aquasense-backend/internal/database"
	"aquasense-backend/internal/handlers"
	"aquasense-backend/internal/logger"
	"aquasense-backend/internal/repository"
	"aquasense-backend/internal/router"
	"aquasense-backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	_ "aquasense-backend/docs" // Import generated swagger docs
)

// @title           AquaSense Backend API
// @version         1.0
// @description     This is the API server for AquaSense TDS application.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@aquasense.local

// @host      localhost:8080
// @BasePath  /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// ── Logger ───────────────────────────────────────────────────────────────
	logger.InitLogger()
	log := logger.Get()
	defer log.Sync() //nolint:errcheck

	// ── Config ────────────────────────────────────────────────────────────────
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	// ── Database ──────────────────────────────────────────────────────────────
	// [Security #9] AutoMigrate only in dev; [Security #8] admin password from config
	db, err := database.Connect(cfg.DSN(), cfg.GinMode != "release", cfg.AdminDefaultPassword)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	// ── Repositories ─────────────────────────────────────────────────────────
	authRepo := repository.NewAuthRepository(db)
	farmRepo := repository.NewFarmRepository(db)
	sensorRepo := repository.NewSensorRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	nodeRepo := repository.NewNodeRepository(db)
	aiRepo := repository.NewAiRepository(db)
	subPlanRepo := repository.NewSubscriptionPlanRepository(db)

	// ── Services ─────────────────────────────────────────────────────────────
	authSvc := service.NewAuthService(authRepo, cfg.JWTSecret, cfg.JWTExpireHours, cfg.GoogleClientID, cfg.AppleClientID)
	aiSvc := service.NewAiService(aiRepo)
	subSvc := service.NewSubscriptionService(subPlanRepo)

	// ── Handlers ─────────────────────────────────────────────────────────────
	authHandler := handlers.NewAuthHandler(authSvc)
	farmHandler := handlers.NewFarmHandler(farmRepo)
	sensorHandler := handlers.NewSensorHandler(sensorRepo, farmRepo, aiSvc)
	accountHandler := handlers.NewAccountHandler(authRepo, notifRepo, subSvc)
	aiHandler := handlers.NewAiHandler(aiSvc)
	nodeHandler := handlers.NewNodeHandler(nodeRepo)

	// ── Router ────────────────────────────────────────────────────────────────
	r := router.Setup(cfg, authHandler, farmHandler, sensorHandler, accountHandler, aiHandler, nodeHandler)

	// ── Graceful Shutdown ─────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Run the server in a goroutine so it doesn't block
	go func() {
		log.Info("AquaSense backend started",
			zap.String("addr", srv.Addr),
			zap.String("mode", cfg.GinMode),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server listen failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server gracefully...")

	// The context is used to inform the server it has 5 seconds to finish existing requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exited")
}
