package handlers

import (
	"errors"
	"net/http"
	"sync"

	"aquasense-backend/internal/logger"
	"aquasense-backend/internal/models"
	"aquasense-backend/internal/repository"
	"aquasense-backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: data})
}

func created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: data})
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, models.ErrorResponse{Success: false, Error: msg})
}

func unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, models.ErrorResponse{Success: false, Error: msg})
}

func notFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, models.ErrorResponse{Success: false, Error: msg})
}

func serverError(c *gin.Context, err error) {
	logger.Get().Error("internal server error",
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.Error(err),
	)
	c.JSON(http.StatusInternalServerError, models.ErrorResponse{Success: false, Error: "เกิดข้อผิดพลาดภายในระบบ กรุณาลองใหม่อีกครั้ง"})
}

// ─── AuthHandler ─────────────────────────────────────────────────────────────

// AuthHandler holds all authentication-related HTTP handlers.
type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// @Summary     Login with email and password
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body body models.LoginRequest true "Login credentials"
// @Success     200 {object} models.APIResponse{data=models.AuthResponse}
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Router      /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	resp, err := h.svc.Login(req)
	if err != nil {
		unauthorized(c, err.Error())
		return
	}
	ok(c, resp)
}

// @Summary     Register a new user
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body body models.RegisterRequest true "Registration data"
// @Success     201 {object} models.APIResponse{data=models.AuthResponse}
// @Failure     400 {object} models.ErrorResponse
// @Failure     409 {object} models.ErrorResponse
// @Router      /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	resp, err := h.svc.Register(req)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	created(c, resp)
}

// @Summary     Social login with Google or Apple
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body body models.SocialLoginRequest true "Provider and id_token"
// @Success     200 {object} models.APIResponse{data=models.AuthResponse}
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Router      /auth/social [post]
func (h *AuthHandler) SocialLogin(c *gin.Context) {
	var req models.SocialLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	resp, err := h.svc.SocialLogin(req)
	if err != nil {
		// Social token verification failure is an auth error (401), not a server error (500)
		unauthorized(c, err.Error())
		return
	}
	ok(c, resp)
}

// @Summary     Request password reset email
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body body models.ForgotPasswordRequest true "Email address"
// @Success     200 {object} models.APIResponse
// @Failure     400 {object} models.ErrorResponse
// @Router      /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.svc.ForgotPassword(req.Email); err != nil {
		serverError(c, err)
		return
	}
	ok(c, gin.H{"message": "หากอีเมลนี้มีในระบบ เราได้ส่งลิงก์รีเซ็ตรหัสผ่านให้แล้ว"})
}

// ─── FarmHandler ─────────────────────────────────────────────────────────────

// FarmHandler holds all farm-related HTTP handlers.
type FarmHandler struct {
	repo *repository.FarmRepository
}

func NewFarmHandler(repo *repository.FarmRepository) *FarmHandler {
	return &FarmHandler{repo: repo}
}

// @Summary     Create or update user's farm (upsert)
// @Tags        Farm
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.CreateFarmRequest true "Farm data"
// @Success     201 {object} models.APIResponse{data=models.FarmJSON}
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Router      /farms [post]
func (h *FarmHandler) CreateFarm(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.CreateFarmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	farm, err := h.repo.CreateFarm(userID, req)
	if err != nil {
		serverError(c, err)
		return
	}
	created(c, farm)
}

// @Summary     Get user's farm
// @Tags        Farm
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.APIResponse{data=models.FarmJSON}
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /farms [get]
func (h *FarmHandler) GetFarm(c *gin.Context) {
	userID := c.GetString("userID")

	farm, err := h.repo.GetFarmByUserID(userID)
	if err != nil {
		serverError(c, err)
		return
	}
	if farm == nil {
		notFound(c, "ยังไม่มีข้อมูลแปลงเกษตร")
		return
	}
	ok(c, farm)
}

// @Summary     Update farm GPS location
// @Tags        Farm
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Farm ID"
// @Param       body body models.UpdateLocationRequest true "GPS coordinates"
// @Success     200 {object} models.APIResponse{data=models.FarmJSON}
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /farms/{id}/location [put]
func (h *FarmHandler) UpdateLocation(c *gin.Context) {
	userID := c.GetString("userID") // [Security #1] must verify ownership
	farmID := c.Param("id")

	var req models.UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.repo.UpdateLocation(userID, farmID, req.Latitude, req.Longitude); err != nil {
		if errors.Is(err, repository.ErrFarmNotFound) { // [Fix K] sentinel error check
			notFound(c, "ไม่พบฟาร์มหรือไม่มีสิทธิ์แก้ไข")
			return
		}
		serverError(c, err)
		return
	}

	farm, err := h.repo.GetFarmByID(farmID)
	if err != nil || farm == nil {
		ok(c, gin.H{"message": "อัปเดตที่ตั้งสำเร็จ"})
		return
	}
	ok(c, farm)
}

// @Summary     Link a sensor to user's farm
// @Tags        Farm
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Farm ID"
// @Param       body body models.LinkSensorRequest true "Sensor ID"
// @Success     200 {object} models.APIResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /farms/{id}/sensor [post]
func (h *FarmHandler) LinkSensor(c *gin.Context) {
	userID := c.GetString("userID") // [Security #1] must verify ownership
	farmID := c.Param("id")

	var req models.LinkSensorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.repo.LinkSensor(userID, farmID, req.SensorID); err != nil {
		switch {
		case errors.Is(err, repository.ErrFarmNotFound):
			notFound(c, "ไม่พบฟาร์มหรือไม่มีสิทธิ์แก้ไข")
		case errors.Is(err, repository.ErrSensorNotFound):
			notFound(c, "ไม่พบ Sensor ID ที่ระบุ")
		default:
			serverError(c, err)
		}
		return
	}
	ok(c, gin.H{"message": "เชื่อมต่อ Sensor สำเร็จ"})
}

// ─── SensorHandler ───────────────────────────────────────────────────────────

// SensorHandler holds all sensor-related HTTP handlers.
type SensorHandler struct {
	sensorRepo *repository.SensorRepository
	farmRepo   *repository.FarmRepository
	aiSvc      *service.AiService
}

func NewSensorHandler(
	sensorRepo *repository.SensorRepository,
	farmRepo *repository.FarmRepository,
	aiSvc *service.AiService,
) *SensorHandler {
	return &SensorHandler{sensorRepo: sensorRepo, farmRepo: farmRepo, aiSvc: aiSvc}
}

// @Summary     Get nearby sensors sorted by distance
// @Tags        Sensor
// @Produce     json
// @Security    BearerAuth
// @Param       lat query number true "Latitude"
// @Param       lng query number true "Longitude"
// @Success     200 {object} models.APIResponse{data=[]models.SensorJSON}
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Router      /sensors/nearby [get]
func (h *SensorHandler) GetNearbySensors(c *gin.Context) {
	type LatLng struct {
		Lat float64 `form:"lat" binding:"required"`
		Lng float64 `form:"lng" binding:"required"`
	}
	var q LatLng
	if err := c.ShouldBindQuery(&q); err != nil {
		badRequest(c, "lat and lng query params are required and must be valid numbers")
		return
	}

	sensors, err := h.sensorRepo.GetNearbySensors(q.Lat, q.Lng)
	if err != nil {
		serverError(c, err)
		return
	}
	ok(c, sensors)
}

// @Summary     Get latest sensor reading
// @Tags        Sensor
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Sensor ID"
// @Success     200 {object} models.APIResponse{data=models.SensorJSON}
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /sensors/{id}/latest [get]
func (h *SensorHandler) GetSensorLatest(c *gin.Context) {
	sensorID := c.Param("id")

	sensor, err := h.sensorRepo.GetSensorLatest(sensorID)
	if err != nil {
		serverError(c, err)
		return
	}
	if sensor == nil {
		notFound(c, "ไม่พบ Sensor ที่ระบุ")
		return
	}
	ok(c, sensor)
}

// @Summary     Get sensor status (safe/warning/danger)
// @Tags        Sensor
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Sensor ID"
// @Success     200 {object} models.APIResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /sensors/{id}/status [get]
func (h *SensorHandler) GetSensorStatus(c *gin.Context) {
	sensorID := c.Param("id")

	sensor, err := h.sensorRepo.GetSensorLatest(sensorID)
	if err != nil {
		serverError(c, err)
		return
	}
	if sensor == nil {
		notFound(c, "ไม่พบ Sensor ที่ระบุ")
		return
	}
	ok(c, gin.H{"status": sensor.Status, "tds_value": sensor.TDSValue})
}

// @Summary     Get sensor water quality history
// @Tags        Sensor
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Sensor ID"
// @Param       period query string false "Period" Enums(7d,30d) default(7d)
// @Success     200 {object} models.APIResponse{data=[]models.WaterRecordJSON}
// @Failure     401 {object} models.ErrorResponse
// @Router      /sensors/{id}/history [get]
func (h *SensorHandler) GetSensorHistory(c *gin.Context) {
	sensorID := c.Param("id")
	period := c.DefaultQuery("period", "7d")

	records, err := h.sensorRepo.GetSensorHistory(sensorID, period)
	if err != nil {
		serverError(c, err)
		return
	}
	ok(c, records)
}

// buildDashboardSummary is a shared helper that builds a DashboardSummaryJSON
// for any given userID. Used by both user and admin dashboard endpoints.
func (h *SensorHandler) buildDashboardSummary(c *gin.Context, userID string) {
	farm, err := h.farmRepo.GetFarmByUserID(userID)
	if err != nil {
		serverError(c, err)
		return
	}

	// [Fix C] Remove hardcoded "s001" fallback — return 404 if no farm/sensor configured.
	// User must create a farm and link a sensor before using the dashboard.
	if farm == nil {
		notFound(c, "ยังไม่มีแปลงเกษตร กรุณาสร้างฟาร์มก่อนใช้งาน Dashboard")
		return
	}
	if farm.ActiveSensorID == nil {
		notFound(c, "ยังไม่มี Sensor ที่เชื่อมต่อ กรุณาเพิ่ม Node และเชื่อมต่อ Sensor ก่อน")
		return
	}
	activeSensorID := *farm.ActiveSensorID

	var sensor *models.SensorJSON
	var history []models.WaterRecordJSON
	var sensorErr, histErr error

	var wg sync.WaitGroup
	wg.Add(2)

	// 1. Fetch sensor data concurrently
	go func() {
		defer wg.Done()
		sensor, sensorErr = h.sensorRepo.GetSensorLatest(activeSensorID)
	}()

	// 2. Fetch history concurrently
	go func() {
		defer wg.Done()
		history, histErr = h.sensorRepo.GetSensorHistory(activeSensorID, "7d")
	}()

	wg.Wait()

	if sensorErr != nil {
		serverError(c, sensorErr)
		return
	}
	if histErr != nil {
		serverError(c, histErr)
		return
	}
	if sensor == nil {
		notFound(c, "ไม่พบข้อมูล Sensor สำหรับ Dashboard")
		return
	}

	tds := sensor.TDSValue
	summary := models.DashboardSummaryJSON{
		FarmID:          farm.ID,   // Blocker #3: Flutter _LocationCard needs farm info
		FarmName:        farm.Name, // Blocker #3
		ActiveSensor:    *sensor,
		Recommendations: h.aiSvc.GetRecommendations(),
		CropSuggestions: h.aiSvc.GetCropSuggestions(tds),
		TrendHistory:    history,
	}
	ok(c, summary)
}

// @Summary     Get full dashboard summary
// @Tags        Dashboard
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.APIResponse{data=models.DashboardSummaryJSON}
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /dashboard/summary [get]
func (h *SensorHandler) GetDashboardSummary(c *gin.Context) {
	userID := c.GetString("userID")
	h.buildDashboardSummary(c, userID)
}

// @Summary     Get dashboard summary for a specific user (admin only)
// @Tags        Admin
// @Produce     json
// @Security    BearerAuth
// @Param       user_id path string true "Target User ID"
// @Success     200 {object} models.APIResponse{data=models.DashboardSummaryJSON}
// @Failure     401 {object} models.ErrorResponse
// @Failure     403 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /admin/users/{user_id}/summary [get]
func (h *SensorHandler) GetUserDashboardSummary(c *gin.Context) {
	targetUserID := c.Param("user_id")
	h.buildDashboardSummary(c, targetUserID)
}

// @Summary     Get soil moisture history for analytics
// @Tags        Dashboard
// @Produce     json
// @Security    BearerAuth
// @Param       sensor_id query string false "Sensor ID (defaults to farm's active sensor)"
// @Param       period query string false "Period" Enums(7d,30d) default(7d)
// @Success     200 {object} models.APIResponse{data=[]models.WaterRecordJSON}
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /analytics/soil-moisture [get]
// Resolves sensor from user's active farm — returns 404 if no sensor linked.
func (h *SensorHandler) GetSoilMoistureHistory(c *gin.Context) {
	userID := c.GetString("userID")
	period := c.DefaultQuery("period", "7d")

	// Resolve sensor_id: explicit query param → farm's active sensor → 404
	sensorID := c.Query("sensor_id")
	if sensorID == "" {
		farm, err := h.farmRepo.GetFarmByUserID(userID)
		if err != nil {
			serverError(c, err)
			return
		}
		if farm == nil || farm.ActiveSensorID == nil {
			notFound(c, "ยังไม่มี Sensor ที่เชื่อมต่อ กรุณาเพิ่ม Node ก่อน")
			return
		}
		sensorID = *farm.ActiveSensorID
	}

	records, err := h.sensorRepo.GetSensorHistory(sensorID, period)
	if err != nil {
		serverError(c, err)
		return
	}
	ok(c, records)
}

// ─── AccountHandler ──────────────────────────────────────────────────────────

// AccountHandler holds user account and subscription HTTP handlers.
type AccountHandler struct {
	authRepo  *repository.AuthRepository
	notifRepo *repository.NotificationRepository
	subSvc    *service.SubscriptionService
}

func NewAccountHandler(
	authRepo *repository.AuthRepository,
	notifRepo *repository.NotificationRepository,
	subSvc *service.SubscriptionService,
) *AccountHandler {
	return &AccountHandler{authRepo: authRepo, notifRepo: notifRepo, subSvc: subSvc}
}

// @Summary     Get current user profile
// @Tags        Account
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.APIResponse{data=models.UserJSON}
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /users/profile [get]
func (h *AccountHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("userID")

	user, err := h.authRepo.FindByID(userID)
	if err != nil || user == nil {
		notFound(c, "ไม่พบผู้ใช้งาน")
		return
	}
	ok(c, user.ToJSON())
}

// @Summary     Update current user profile
// @Tags        Account
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.UpdateProfileRequest true "Profile data"
// @Success     200 {object} models.APIResponse{data=models.UserJSON}
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Router      /users/profile [put]
func (h *AccountHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.authRepo.UpdateProfile(userID, req.FirstName, req.LastName, req.Phone); err != nil {
		serverError(c, err)
		return
	}

	user, err := h.authRepo.FindByID(userID)
	if err != nil || user == nil {
		ok(c, gin.H{"message": "อัปเดตโปรไฟล์สำเร็จ"})
		return
	}
	ok(c, user.ToJSON())
}

// @Summary     Get notification settings
// @Tags        Account
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.APIResponse{data=models.NotificationSettingsJSON}
// @Failure     401 {object} models.ErrorResponse
// @Router      /users/notification-settings [get]
func (h *AccountHandler) GetNotificationSettings(c *gin.Context) {
	userID := c.GetString("userID")

	settings, err := h.notifRepo.GetSettings(userID)
	if err != nil {
		serverError(c, err)
		return
	}
	ok(c, settings)
}

// @Summary     Save notification settings
// @Tags        Account
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.NotificationSettingsJSON true "Notification settings"
// @Success     200 {object} models.APIResponse{data=models.NotificationSettingsJSON}
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Router      /users/notification-settings [put]
func (h *AccountHandler) SaveNotificationSettings(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.NotificationSettingsJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.notifRepo.SaveSettings(userID, req); err != nil {
		serverError(c, err)
		return
	}
	ok(c, req)
}

// @Summary     Get all subscription plans
// @Tags        Account
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.APIResponse{data=[]models.SubscriptionPlanJSON}
// @Failure     401 {object} models.ErrorResponse
// @Router      /subscriptions/plans [get]
func (h *AccountHandler) GetSubscriptionPlans(c *gin.Context) {
	ok(c, h.subSvc.GetPlans())
}

// @Summary     Subscribe to a plan
// @Tags        Account
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.SubscribeRequest true "Plan ID"
// @Success     200 {object} models.APIResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /subscriptions/subscribe [post]
func (h *AccountHandler) Subscribe(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	if !h.subSvc.ValidatePlan(req.PlanID) {
		notFound(c, "ไม่พบแผนที่ระบุ: "+req.PlanID)
		return
	}

	if err := h.authRepo.UpdateSubscription(userID, req.PlanID); err != nil {
		serverError(c, err)
		return
	}
	ok(c, gin.H{"message": "สมัครแผน " + req.PlanID + " สำเร็จ"})
}

// ─── AiHandler ───────────────────────────────────────────────────────────────

// AiHandler holds all AI recommendation HTTP handlers.
type AiHandler struct {
	svc *service.AiService
}

func NewAiHandler(svc *service.AiService) *AiHandler {
	return &AiHandler{svc: svc}
}

// @Summary     Get all AI recommendations
// @Tags        AI
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.APIResponse{data=[]models.AiRecommendationJSON}
// @Failure     401 {object} models.ErrorResponse
// @Router      /ai/recommendations [get]
func (h *AiHandler) GetRecommendations(c *gin.Context) {
	ok(c, h.svc.GetRecommendations())
}

// @Summary     Get single AI recommendation detail
// @Tags        AI
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Recommendation ID"
// @Success     200 {object} models.APIResponse{data=models.AiRecommendationJSON}
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /ai/recommendations/{id} [get]
func (h *AiHandler) GetRecommendationDetail(c *gin.Context) {
	id := c.Param("id")
	rec := h.svc.GetRecommendationByID(id)
	if rec == nil {
		notFound(c, "ไม่พบคำแนะนำที่ระบุ")
		return
	}
	ok(c, rec)
}

// @Summary     Get AI advisory history
// @Tags        AI
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.APIResponse{data=[]models.AiRecommendationJSON}
// @Failure     401 {object} models.ErrorResponse
// @Router      /ai/advisory-history [get]
func (h *AiHandler) GetAdvisoryHistory(c *gin.Context) {
	ok(c, h.svc.GetAdvisoryHistory())
}

// @Summary     Get crop suggestions based on TDS value
// @Tags        AI
// @Produce     json
// @Security    BearerAuth
// @Param       tds query number false "Current TDS value (ppm)"
// @Success     200 {object} models.APIResponse{data=[]models.CropSuggestionJSON}
// @Failure     401 {object} models.ErrorResponse
// @Router      /ai/crop-suggestions [get]
func (h *AiHandler) GetCropSuggestions(c *gin.Context) {
	type Query struct {
		TDS float64 `form:"tds"`
	}
	var q Query
	_ = c.ShouldBindQuery(&q)

	ok(c, h.svc.GetCropSuggestions(q.TDS))
}

// ─── NodeHandler ───────────────────────────────────────────────────────────

// NodeHandler holds all sensor node management HTTP handlers.
type NodeHandler struct {
	repo *repository.NodeRepository
}

func NewNodeHandler(repo *repository.NodeRepository) *NodeHandler {
	return &NodeHandler{repo: repo}
}

// @Summary     Get all sensor nodes for current user
// @Tags        Node
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.APIResponse{data=[]models.NodeJSON}
// @Failure     401 {object} models.ErrorResponse
// @Router      /nodes [get]
func (h *NodeHandler) GetNodes(c *gin.Context) {
	userID := c.GetString("userID")
	nodes, err := h.repo.GetUserNodes(userID)
	if err != nil {
		serverError(c, err)
		return
	}
	if nodes == nil {
		nodes = []models.NodeJSON{}
	}
	ok(c, nodes)
}

// @Summary     Add a new sensor node (max 5 per user)
// @Tags        Node
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.LinkSensorRequest true "Sensor ID to add"
// @Success     201 {object} models.APIResponse{data=models.NodeJSON}
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Failure     409 {object} models.ErrorResponse
// @Router      /nodes [post]
// [Fix K] Use errors.Is to distinguish user errors from server errors
func (h *NodeHandler) AddNode(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.LinkSensorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	node, err := h.repo.AddNode(userID, req.SensorID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrSensorNotFound):
			notFound(c, "ไม่พบ Sensor ID ที่ระบุ")
		case errors.Is(err, repository.ErrNodeDuplicate):
			badRequest(c, "เชื่อมต่อ Sensor นี้ไว้อยู่แล้ว")
		case errors.Is(err, repository.ErrNodeCapacity):
			badRequest(c, "เชื่อมต่อได้สูงสุด 5 nodes")
		default:
			serverError(c, err)
		}
		return
	}
	created(c, node)
}

// @Summary     Set a node as active (used in dashboard)
// @Tags        Node
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Node ID"
// @Success     200 {object} models.APIResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /nodes/{id}/active [put]
func (h *NodeHandler) SetActiveNode(c *gin.Context) {
	userID := c.GetString("userID")
	nodeID := c.Param("id")

	if err := h.repo.SetActiveNode(userID, nodeID); err != nil {
		if errors.Is(err, repository.ErrNodeNotFound) {
			notFound(c, "ไม่พบ Node ที่ระบุ")
			return
		}
		serverError(c, err)
		return
	}
	ok(c, gin.H{"message": "ตั้งเป็น Active Node สำเร็จ"})
}

// @Summary     Remove a sensor node
// @Tags        Node
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Node ID"
// @Success     200 {object} models.APIResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /nodes/{id} [delete]
func (h *NodeHandler) RemoveNode(c *gin.Context) {
	userID := c.GetString("userID")
	nodeID := c.Param("id")

	if err := h.repo.RemoveNode(userID, nodeID); err != nil {
		if errors.Is(err, repository.ErrNodeNotFound) {
			notFound(c, "ไม่พบ Node ที่ระบุ")
			return
		}
		serverError(c, err)
		return
	}
	ok(c, gin.H{"message": "ยกเลิกการเชื่อมต่อสำเร็จ"})
}
