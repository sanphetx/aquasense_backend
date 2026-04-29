package router

import (
	"time"

	"aquasense-backend/internal/config"
	"aquasense-backend/internal/handlers"
	"aquasense-backend/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Setup configures and returns a new gin.Engine with all routes registered.
func Setup(
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	farmHandler *handlers.FarmHandler,
	sensorHandler *handlers.SensorHandler,
	accountHandler *handlers.AccountHandler,
	aiHandler *handlers.AiHandler,
) *gin.Engine {
	r := gin.Default()

	// ── CORS Middleware ──────────────────────────────────────────────────────
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Adjust this in production
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Swagger documentation route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "aquasense-backend"})
	})

	api := r.Group("/api/v1")

	// ── Public routes (no JWT required) ──────────────────────────────────────
	auth := api.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
		auth.POST("/social", authHandler.SocialLogin)
	}

	// ── Protected routes (JWT required) ──────────────────────────────────────
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// Farms
		protected.POST("/farms", farmHandler.CreateFarm)
		protected.GET("/farms", farmHandler.GetFarm)
		protected.PUT("/farms/:id/location", farmHandler.UpdateLocation)
		protected.POST("/farms/:id/sensor", farmHandler.LinkSensor)

		// Sensors
		protected.GET("/sensors/nearby", sensorHandler.GetNearbySensors)
		protected.GET("/sensors/:id/latest", sensorHandler.GetSensorLatest)
		protected.GET("/sensors/:id/history", sensorHandler.GetSensorHistory)
		protected.GET("/sensors/:id/status", sensorHandler.GetSensorStatus)

		// Dashboard
		protected.GET("/dashboard/summary", sensorHandler.GetDashboardSummary)

		// Analytics
		protected.GET("/analytics/soil-moisture", sensorHandler.GetSoilMoistureHistory)

		// AI
		protected.GET("/ai/recommendations", aiHandler.GetRecommendations)
		protected.GET("/ai/recommendations/:id", aiHandler.GetRecommendationDetail)
		protected.GET("/ai/advisory-history", aiHandler.GetAdvisoryHistory)
		protected.GET("/ai/crop-suggestions", aiHandler.GetCropSuggestions)

		// Account
		protected.GET("/users/profile", accountHandler.GetProfile)
		protected.PUT("/users/profile", accountHandler.UpdateProfile)

		// Notification Settings
		protected.GET("/users/notification-settings", accountHandler.GetNotificationSettings)
		protected.PUT("/users/notification-settings", accountHandler.SaveNotificationSettings)

		// Subscriptions
		protected.GET("/subscriptions/plans", accountHandler.GetSubscriptionPlans)
		protected.POST("/subscriptions/subscribe", accountHandler.Subscribe)
	}

	// ── Admin routes (JWT + Admin Role required) ─────────────────────────────
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	admin.Use(middleware.AdminMiddleware())
	{
		admin.GET("/users/:user_id/summary", sensorHandler.GetUserDashboardSummary)
	}

	return r
}
