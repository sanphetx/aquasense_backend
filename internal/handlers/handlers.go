package handlers

import (
	"net/http"
	"sync"

	"aquasense-backend/internal/models"
	"aquasense-backend/internal/repository"
	"aquasense-backend/internal/service"

	"github.com/gin-gonic/gin"
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
	c.JSON(http.StatusInternalServerError, models.ErrorResponse{Success: false, Error: err.Error()})
}

// ─── AuthHandler ─────────────────────────────────────────────────────────────

// AuthHandler holds all authentication-related HTTP handlers.
type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Login handles POST /auth/login
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

// Register handles POST /auth/register
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

// SocialLogin handles POST /auth/social
func (h *AuthHandler) SocialLogin(c *gin.Context) {
	var req models.SocialLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	resp, err := h.svc.SocialLogin(req)
	if err != nil {
		serverError(c, err)
		return
	}
	ok(c, resp)
}

// ─── FarmHandler ─────────────────────────────────────────────────────────────

// FarmHandler holds all farm-related HTTP handlers.
type FarmHandler struct {
	repo *repository.FarmRepository
}

func NewFarmHandler(repo *repository.FarmRepository) *FarmHandler {
	return &FarmHandler{repo: repo}
}

// CreateFarm handles POST /farms
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

// GetFarm handles GET /farms (returns the user's primary farm)
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

// UpdateLocation handles PUT /farms/:id/location
func (h *FarmHandler) UpdateLocation(c *gin.Context) {
	farmID := c.Param("id")

	var req models.UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.repo.UpdateLocation(farmID, req.Latitude, req.Longitude); err != nil {
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

// LinkSensor handles POST /farms/:id/sensor
func (h *FarmHandler) LinkSensor(c *gin.Context) {
	farmID := c.Param("id")

	var req models.LinkSensorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.repo.LinkSensor(farmID, req.SensorID); err != nil {
		serverError(c, err)
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

// GetNearbySensors handles GET /sensors/nearby?lat=&lng=
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

// GetSensorLatest handles GET /sensors/:id/latest
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

// GetSensorStatus handles GET /sensors/:id/status
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

// GetSensorHistory handles GET /sensors/:id/history?period=7d|30d
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

// GetDashboardSummary handles GET /dashboard/summary
func (h *SensorHandler) GetDashboardSummary(c *gin.Context) {
	userID := c.GetString("userID")

	farm, err := h.farmRepo.GetFarmByUserID(userID)
	if err != nil {
		serverError(c, err)
		return
	}

	var activeSensorID string
	if farm != nil && farm.ActiveSensorID != nil {
		activeSensorID = *farm.ActiveSensorID
	} else {
		activeSensorID = "s001" // default seed sensor
	}

	var sensor *models.SensorJSON
	var history []models.WaterRecordJSON
	var sensorErr, histErr error

	var wg sync.WaitGroup
	wg.Add(2)

	// 1. Fetch sensor data concurrently
	go func() {
		defer wg.Done()
		sensor, sensorErr = h.sensorRepo.GetSensorLatest(activeSensorID)
		if sensorErr == nil && sensor == nil && activeSensorID != "s001" {
			// Fallback: try the first seed sensor
			sensor, sensorErr = h.sensorRepo.GetSensorLatest("s001")
		}
	}()

	// 2. Fetch history concurrently
	go func() {
		defer wg.Done()
		history, histErr = h.sensorRepo.GetSensorHistory(activeSensorID, "7d")
	}()

	wg.Wait() // Wait for both queries to finish

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
		ActiveSensor:    *sensor,
		Recommendations: h.aiSvc.GetRecommendations(),
		CropSuggestions: h.aiSvc.GetCropSuggestions(tds),
		TrendHistory:    history,
	}
	ok(c, summary)
}

// GetUserDashboardSummary handles GET /admin/users/:user_id/summary (Admin only)
func (h *SensorHandler) GetUserDashboardSummary(c *gin.Context) {
	targetUserID := c.Param("user_id")

	farm, err := h.farmRepo.GetFarmByUserID(targetUserID)
	if err != nil {
		serverError(c, err)
		return
	}

	var activeSensorID string
	if farm != nil && farm.ActiveSensorID != nil {
		activeSensorID = *farm.ActiveSensorID
	} else {
		activeSensorID = "s001" // default seed sensor
	}

	var sensor *models.SensorJSON
	var history []models.WaterRecordJSON
	var sensorErr, histErr error

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		sensor, sensorErr = h.sensorRepo.GetSensorLatest(activeSensorID)
		if sensorErr == nil && sensor == nil && activeSensorID != "s001" {
			sensor, sensorErr = h.sensorRepo.GetSensorLatest("s001")
		}
	}()

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
		ActiveSensor:    *sensor,
		Recommendations: h.aiSvc.GetRecommendations(),
		CropSuggestions: h.aiSvc.GetCropSuggestions(tds),
		TrendHistory:    history,
	}
	ok(c, summary)
}

// GetSoilMoistureHistory handles GET /analytics/soil-moisture?sensor_id=&period=
// sensor_id is optional — falls back to the user's active farm sensor, then seed sensor s001.
func (h *SensorHandler) GetSoilMoistureHistory(c *gin.Context) {
	userID := c.GetString("userID")
	period := c.DefaultQuery("period", "7d")

	// Resolve sensor_id
	sensorID := c.Query("sensor_id")
	if sensorID == "" {
		if farm, err := h.farmRepo.GetFarmByUserID(userID); err == nil && farm != nil && farm.ActiveSensorID != nil {
			sensorID = *farm.ActiveSensorID
		} else {
			sensorID = "s001"
		}
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

// GetProfile handles GET /users/profile
func (h *AccountHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("userID")

	user, err := h.authRepo.FindByID(userID)
	if err != nil || user == nil {
		notFound(c, "ไม่พบผู้ใช้งาน")
		return
	}
	ok(c, user.ToJSON())
}

// UpdateProfile handles PUT /users/profile
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

// GetNotificationSettings handles GET /users/notification-settings
func (h *AccountHandler) GetNotificationSettings(c *gin.Context) {
	userID := c.GetString("userID")

	settings, err := h.notifRepo.GetSettings(userID)
	if err != nil {
		serverError(c, err)
		return
	}
	ok(c, settings)
}

// SaveNotificationSettings handles PUT /users/notification-settings
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

// GetSubscriptionPlans handles GET /subscriptions/plans
func (h *AccountHandler) GetSubscriptionPlans(c *gin.Context) {
	ok(c, h.subSvc.GetPlans())
}

// Subscribe handles POST /subscriptions/subscribe
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

// GetRecommendations handles GET /ai/recommendations
func (h *AiHandler) GetRecommendations(c *gin.Context) {
	ok(c, h.svc.GetRecommendations())
}

// GetRecommendationDetail handles GET /ai/recommendations/:id
func (h *AiHandler) GetRecommendationDetail(c *gin.Context) {
	id := c.Param("id")
	rec := h.svc.GetRecommendationByID(id)
	if rec == nil {
		notFound(c, "ไม่พบคำแนะนำที่ระบุ")
		return
	}
	ok(c, rec)
}

// GetAdvisoryHistory handles GET /ai/advisory-history
func (h *AiHandler) GetAdvisoryHistory(c *gin.Context) {
	ok(c, h.svc.GetAdvisoryHistory())
}

// GetCropSuggestions handles GET /ai/crop-suggestions?tds=&soil_ph=
func (h *AiHandler) GetCropSuggestions(c *gin.Context) {
	type Query struct {
		TDS    float64  `form:"tds"`
		SoilPH *float64 `form:"soil_ph"`
	}
	var q Query
	_ = c.ShouldBindQuery(&q)

	ok(c, h.svc.GetCropSuggestions(q.TDS))
}
