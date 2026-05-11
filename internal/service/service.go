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

// AiService reads AI recommendations and crop suggestions from the database.
type AiService struct {
	repo *repository.AiRepository
}

func NewAiService(repo *repository.AiRepository) *AiService {
	return &AiService{repo: repo}
}

// GetRecommendations returns all AI recommendations from DB.
func (s *AiService) GetRecommendations() []models.AiRecommendationJSON {
	recs, err := s.repo.GetRecommendations()
	if err != nil || len(recs) == 0 {
		return []models.AiRecommendationJSON{}
	}
	return recs
}

// GetRecommendationByID returns a single recommendation or nil.
func (s *AiService) GetRecommendationByID(id string) *models.AiRecommendationJSON {
	rec, err := s.repo.GetRecommendationByID(id)
	if err != nil {
		return nil
	}
	return rec
}

// GetAdvisoryHistory returns older recommendations from DB.
func (s *AiService) GetAdvisoryHistory() []models.AiRecommendationJSON {
	recs, err := s.repo.GetAdvisoryHistory()
	if err != nil || len(recs) == 0 {
		return []models.AiRecommendationJSON{}
	}
	return recs
}

// GetCropSuggestions returns filtered crop suggestions based on TDS from DB.
func (s *AiService) GetCropSuggestions(tds float64) []models.CropSuggestionJSON {
	crops, err := s.repo.GetCropSuggestions(tds)
	if err != nil || len(crops) == 0 {
		return []models.CropSuggestionJSON{}
	}
	return crops
}

// ─── SubscriptionService ─────────────────────────────────────────────────────

// SubscriptionService reads subscription plans from the database.
type SubscriptionService struct {
	repo *repository.SubscriptionPlanRepository
}

func NewSubscriptionService(repo *repository.SubscriptionPlanRepository) *SubscriptionService {
	return &SubscriptionService{repo: repo}
}

// GetPlans returns all subscription plans from DB.
func (s *SubscriptionService) GetPlans() []models.SubscriptionPlanJSON {
	plans, err := s.repo.GetAll()
	if err != nil || len(plans) == 0 {
		return []models.SubscriptionPlanJSON{}
	}
	return plans
}

// ValidatePlan checks if a plan ID exists in DB.
func (s *SubscriptionService) ValidatePlan(planID string) bool {
	return s.repo.FindByID(planID)
}
