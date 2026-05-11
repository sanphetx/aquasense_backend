package service

import (
	"fmt"
	"strings"
	"time"

	"aquasense-backend/internal/models"
	"aquasense-backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ─── AuthService ─────────────────────────────────────────────────────────────

// AuthService encapsulates auth business logic.
type AuthService struct {
	repo           *repository.AuthRepository
	jwtSecret      string
	jwtExpire      int // hours
	googleClientID string
	appleClientID  string
}

func NewAuthService(repo *repository.AuthRepository, secret string, expireHours int, googleClientID, appleClientID string) *AuthService {
	return &AuthService{
		repo:           repo,
		jwtSecret:      secret,
		jwtExpire:      expireHours,
		googleClientID: googleClientID,
		appleClientID:  appleClientID,
	}
}

// Login verifies credentials and returns an AuthResponse.
func (s *AuthService) Login(req models.LoginRequest) (*models.AuthResponse, error) {
	email := strings.ToLower(req.Email)

	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil || !s.repo.CheckPassword(user.PasswordHash, req.Password) {
		return nil, fmt.Errorf("อีเมลหรือรหัสผ่านไม่ถูกต้อง")
	}

	token, err := s.generateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:        token,
		User:         user.ToJSON(),
		IsFirstLogin: !s.repo.HasFarm(user.ID),
	}, nil
}

// Register creates a new user and returns an AuthResponse.
func (s *AuthService) Register(req models.RegisterRequest) (*models.AuthResponse, error) {
	req.Email = strings.ToLower(req.Email)

	// Handle display_name → split into first/last if first_name not provided
	if req.FirstName == "" && req.DisplayName != "" {
		parts := strings.SplitN(strings.TrimSpace(req.DisplayName), " ", 2)
		req.FirstName = parts[0]
		if len(parts) > 1 {
			req.LastName = parts[1]
		}
	}
	if req.FirstName == "" {
		return nil, fmt.Errorf("first_name or display_name is required")
	}

	existing, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("อีเมลนี้ถูกใช้งานแล้ว")
	}

	user, err := s.repo.CreateUser(req)
	if err != nil {
		return nil, err
	}

	token, err := s.generateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:        token,
		User:         user.ToJSON(),
		IsFirstLogin: true, // newly registered user always needs setup
	}, nil
}

// ForgotPassword initiates a password reset flow.
// In production, this would send an email with a reset link/OTP.
func (s *AuthService) ForgotPassword(email string) error {
	email = strings.ToLower(email)
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return err
	}
	if user == nil {
		// Return nil even if not found to prevent email enumeration
		return nil
	}
	// TODO: Send reset email via SMTP or third-party service
	return nil
}

// SocialLogin verifies a provider id_token and returns an AuthResponse.
// Supports: google, apple
func (s *AuthService) SocialLogin(req models.SocialLoginRequest) (*models.AuthResponse, error) {
	var info *models.SocialUserInfo
	var err error

	switch strings.ToLower(req.Provider) {
	case "google":
		info, err = VerifyGoogleToken(req.IDToken, s.googleClientID)
		if err != nil {
			return nil, fmt.Errorf("Google login failed: %w", err)
		}
	case "apple":
		info, err = VerifyAppleToken(req.IDToken, s.appleClientID)
		if err != nil {
			return nil, fmt.Errorf("Apple login failed: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %q (supported: google, apple)", req.Provider)
	}

	// Find existing user or create a new one
	user, err := s.repo.FindByEmail(info.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		firstName := info.FirstName
		if firstName == "" {
			firstName = req.Provider // fallback: "google" or "apple"
		}
		fakeReq := models.RegisterRequest{
			FirstName: firstName,
			LastName:  info.LastName,
			Email:     info.Email,
			BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			Password:  uuid.NewString(), // random, unusable password
		}
		user, err = s.repo.CreateUser(fakeReq)
		if err != nil {
			return nil, err
		}
	}

	token, err := s.generateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}
	return &models.AuthResponse{
		Token:        token,
		User:         user.ToJSON(),
		IsFirstLogin: !s.repo.HasFarm(user.ID),
	}, nil
}

func (s *AuthService) generateToken(userID, email, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(time.Duration(s.jwtExpire) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// ─── AiService ───────────────────────────────────────────────────────────────

// AiService returns hard-coded AI recommendations (same data as MockAiRepository).
// Replace with a real ML backend call in production.
type AiService struct{}

func NewAiService() *AiService { return &AiService{} }

// staticRecommendations mirrors MockAiRepository._recommendations.
var staticRecommendations = []models.AiRecommendationJSON{
	{
		ID:    "ai001",
		Title: "ค่า TDS มีแนวโน้มเพิ่มขึ้น",
		Body:  "พยากรณ์อากาศ 5 วันข้างหน้าไม่มีฝนในพื้นที่ของคุณ ค่า TDS อาจเพิ่มขึ้นถึง 450–500 ppm แนะนำให้เปิดประตูน้ำเพื่อเจือจางก่อนปลูก",
		Type:  "tds_alert",
		ReasonChips: []models.ReasonChip{
			{Label: "TDS 420 ppm", Category: "tds"},
			{Label: "เพิ่ม 3 วันติดกัน", Category: "trend"},
			{Label: "ไม่มีฝน 5 วัน", Category: "weather"},
		},
		ConfidenceScore: 0.87,
	},
	{
		ID:    "ai002",
		Title: "แนะนำพืชทางเลือก",
		Body:  "ราคาข้าวในตลาดมีแนวโน้มลดลง 8% ในฤดูกาลนี้ การปลูกถั่วเขียวหรือข้าวโพดหวานอาจให้ผลตอบแทนสูงกว่า",
		Type:  "crop_suggestion",
		ReasonChips: []models.ReasonChip{
			{Label: "ราคาข้าว ↓ 8%", Category: "market"},
			{Label: "TDS 350 ppm", Category: "tds"},
		},
		ConfidenceScore: 0.76,
	},
	{
		ID:    "ai003",
		Title: "ระดับ TDS วิกฤต — Sensor 03",
		Body:  "Sensor 03 (คลองชลประทาน) วัดค่า TDS ที่ 720 ppm เกินระดับวิกฤต 500 ppm แนะนำให้ปิดการรับน้ำจากแหล่งนี้ทันทีและเปิดน้ำจาก Sensor 01 แทน",
		Type:  "tds_danger",
		ReasonChips: []models.ReasonChip{
			{Label: "TDS 720 ppm", Category: "tds"},
			{Label: "เกินเกณฑ์วิกฤต", Category: "trend"},
			{Label: "ไม่มีฝน 5 วัน", Category: "weather"},
		},
		ConfidenceScore: 0.96,
	},
	{
		ID:    "ai004",
		Title: "ช่วงเวลาใส่ปุ๋ยที่เหมาะสม",
		Body:  "ค่า TDS อยู่ในระดับเหมาะสม (350 ppm) แนะนำให้ใส่ปุ๋ยสูตร 16-20-0 ในช่วงเช้า 06:00–08:00 น. ใน 3 วันข้างหน้า เนื่องจากมีฝนพยากรณ์ที่จะช่วยพาปุ๋ยลงดิน",
		Type:  "fertilizer",
		ReasonChips: []models.ReasonChip{
			{Label: "TDS 350 ppm", Category: "tds"},
			{Label: "มีฝน 3 วันข้างหน้า", Category: "weather"},
		},
		ConfidenceScore: 0.82,
	},
}

// staticCropSuggestions mirrors MockAiRepository._cropSuggestions.
var staticCropSuggestions = []models.CropSuggestionJSON{
	{Name: "Mung Bean", NameTH: "ถั่วเขียว", EstimatedPricePerKg: 28.0, Reason: "ทนน้ำเค็มปานกลาง เหมาะกับ TDS 300–500 ppm", Icon: "🫘"},
	{Name: "Sweet Corn", NameTH: "ข้าวโพดหวาน", EstimatedPricePerKg: 12.5, Reason: "ราคาตลาดสูง ใช้น้ำน้อยกว่าข้าว 40%", Icon: "🌽"},
	{Name: "Rice", NameTH: "ข้าว", EstimatedPricePerKg: 9.5, Reason: "พืชหลักปัจจุบัน", Icon: "🌾"},
}

// GetRecommendations returns all AI recommendations with timestamps filled in.
func (s *AiService) GetRecommendations() []models.AiRecommendationJSON {
	now := time.Now()
	result := make([]models.AiRecommendationJSON, len(staticRecommendations))
	offsets := []time.Duration{2 * time.Hour, 5 * time.Hour, 30 * time.Minute, 1 * time.Hour}
	for i, r := range staticRecommendations {
		r.CreatedAt = now.Add(-offsets[i]).Format(time.RFC3339)
		result[i] = r
	}
	return result
}

// GetRecommendationByID returns a single recommendation or nil.
func (s *AiService) GetRecommendationByID(id string) *models.AiRecommendationJSON {
	for _, r := range s.GetRecommendations() {
		if r.ID == id {
			return &r
		}
	}
	return nil
}

// GetAdvisoryHistory returns older copies of recommendations to simulate history.
func (s *AiService) GetAdvisoryHistory() []models.AiRecommendationJSON {
	base := s.GetRecommendations()
	hist := make([]models.AiRecommendationJSON, len(base))
	for i, r := range base {
		t, _ := time.Parse(time.RFC3339, r.CreatedAt)
		r.ID = r.ID + "_hist"
		r.CreatedAt = t.Add(-7 * 24 * time.Hour).Format(time.RFC3339)
		hist[i] = r
	}
	return hist
}

// GetCropSuggestions returns filtered crop suggestions based on TDS.
func (s *AiService) GetCropSuggestions(tds float64) []models.CropSuggestionJSON {
	var filtered []models.CropSuggestionJSON
	for _, c := range staticCropSuggestions {
		switch {
		case tds > 600:
			if c.Name == "Mung Bean" {
				filtered = append(filtered, c)
			}
		case tds > 400:
			if c.Name != "Rice" {
				filtered = append(filtered, c)
			}
		default:
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return staticCropSuggestions
	}
	return filtered
}

// ─── SubscriptionService ─────────────────────────────────────────────────────

// SubscriptionService handles subscription plan logic.
type SubscriptionService struct{}

func NewSubscriptionService() *SubscriptionService { return &SubscriptionService{} }

// staticPlans mirrors MockAccountRepository._plans.
var staticPlans = []models.SubscriptionPlanJSON{
	{
		ID: "free", Name: "Free", Price: "฿0", Period: "",
		Features:    []string{"ดูข้อมูลตัวอย่าง (Demo)", "AI พื้นฐาน", "ไม่มีการแจ้งเตือน"},
		Recommended: false,
	},
	{
		ID: "starter", Name: "Starter", Price: "฿59", Period: "/ฤดูกาล",
		Features:    []string{"เชื่อมต่อ 1 Sensor", "แจ้งเตือนผ่านแอป", "บันทึกสถิติรายสัปดาห์", "AI ระดับพื้นฐาน"},
		Recommended: true,
	},
	{
		ID: "pro", Name: "Pro", Price: "฿199", Period: "/ปี",
		Features:    []string{"เชื่อมต่อ 5 Sensors", "AI Level 3 — วิเคราะห์เชิงลึก", "พยากรณ์ผลผลิตรายเดือน", "Export CSV/PDF", "สรุปรายงานผ่าน LINE"},
		Recommended: false,
	},
}

// GetPlans returns all subscription plans.
func (s *SubscriptionService) GetPlans() []models.SubscriptionPlanJSON {
	return staticPlans
}

// ValidatePlan checks if a plan ID exists.
func (s *SubscriptionService) ValidatePlan(planID string) bool {
	for _, p := range staticPlans {
		if p.ID == planID {
			return true
		}
	}
	return false
}
