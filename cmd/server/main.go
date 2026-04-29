package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aquasense-backend/internal/config"
	"aquasense-backend/internal/database"
	"aquasense-backend/internal/handlers"
	"aquasense-backend/internal/repository"
	"aquasense-backend/internal/router"
	"aquasense-backend/internal/service"

	"github.com/gin-gonic/gin"

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
	// ── Config ────────────────────────────────────────────────────────────────
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := database.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("[main] failed to connect to database: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	// ── Repositories ─────────────────────────────────────────────────────────
	authRepo := repository.NewAuthRepository(db)
	farmRepo := repository.NewFarmRepository(db)
	sensorRepo := repository.NewSensorRepository(db)
	notifRepo := repository.NewNotificationRepository(db)

	// ── Services ─────────────────────────────────────────────────────────────
	authSvc := service.NewAuthService(authRepo, cfg.JWTSecret, cfg.JWTExpireHours)
	aiSvc := service.NewAiService()
	subSvc := service.NewSubscriptionService()

	// ── Handlers ─────────────────────────────────────────────────────────────
	authHandler := handlers.NewAuthHandler(authSvc)
	farmHandler := handlers.NewFarmHandler(farmRepo)
	sensorHandler := handlers.NewSensorHandler(sensorRepo, farmRepo, aiSvc)
	accountHandler := handlers.NewAccountHandler(authRepo, notifRepo, subSvc)
	aiHandler := handlers.NewAiHandler(aiSvc)

	// ── Router ────────────────────────────────────────────────────────────────
	r := router.Setup(cfg, authHandler, farmHandler, sensorHandler, accountHandler, aiHandler)

	// ── Graceful Shutdown ─────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Run the server in a goroutine so it doesn't block
	go func() {
		log.Printf("[main] AquaSense backend listening on %s (mode: %s)\n", srv.Addr, cfg.GinMode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server gracefully...")

	// The context is used to inform the server it has 5 seconds to finish existing requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
