package service

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"aquasense-backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

// [Security #4] oauthClient has a 10-second timeout to prevent goroutine leaks.
var oauthClient = &http.Client{Timeout: 10 * time.Second}

// ─── Google OAuth ─────────────────────────────────────────────────────────────

// googleTokenInfo is the response from Google's tokeninfo endpoint.
type googleTokenInfo struct {
	Sub           string `json:"sub"`            // Google user ID
	Email         string `json:"email"`           // verified email
	EmailVerified string `json:"email_verified"`  // must be "true"
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Aud           string `json:"aud"`  // must match our client ID
	Exp           string `json:"exp"`  // unix timestamp
	Error         string `json:"error"`
	ErrorDesc     string `json:"error_description"`
}

// VerifyGoogleToken sends the id_token to Google's tokeninfo endpoint and
// returns the verified user info. Validates aud and email_verified.
func VerifyGoogleToken(idToken, clientID string) (*models.SocialUserInfo, error) {
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken

	resp, err := oauthClient.Get(url) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("google: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var info googleTokenInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("google: failed to parse response: %w", err)
	}

	if info.Error != "" {
		return nil, fmt.Errorf("google: token invalid — %s: %s", info.Error, info.ErrorDesc)
	}

	// [Security #10] Validate email_verified
	if info.EmailVerified != "true" {
		return nil, fmt.Errorf("google: email not verified")
	}

	// Validate audience if clientID is configured
	if clientID != "" && info.Aud != clientID {
		return nil, fmt.Errorf("google: token audience mismatch (got %q, want %q)", info.Aud, clientID)
	}

	if info.Email == "" {
		return nil, fmt.Errorf("google: token does not contain email")
	}

	return &models.SocialUserInfo{
		ProviderUserID: info.Sub,
		Email:          strings.ToLower(info.Email),
		FirstName:      info.GivenName,
		LastName:       info.FamilyName,
	}, nil
}

// ─── Apple OAuth ─────────────────────────────────────────────────────────────

// appleJWKS is the response from Apple's public key endpoint.
type appleJWKS struct {
	Keys []appleJWK `json:"keys"`
}

type appleJWK struct {
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"` // RSA modulus (base64url)
	E   string `json:"e"` // RSA exponent (base64url)
}

// appleClaims holds Apple ID token claims.
type appleClaims struct {
	Sub   string `json:"sub"`   // Apple user ID
	Email string `json:"email"` // may be empty if user hides email
	jwt.RegisteredClaims
}

// applePublicKeysURL is Apple's JWKS endpoint.
const applePublicKeysURL = "https://appleid.apple.com/auth/keys"

// [Security #5] Apple JWKS cache — Apple keys rarely rotate (monthly).
// Cache for 24 hours to avoid per-request network calls.
var (
	appleKeysMu       sync.RWMutex
	cachedAppleKeys   []appleJWK
	appleKeysCachedAt time.Time
	appleKeysCacheTTL = 24 * time.Hour
)

// VerifyAppleToken verifies an Apple ID token using Apple's public JWKS.
// clientID (bundle ID) is used to validate the audience claim.
func VerifyAppleToken(idToken, clientID string) (*models.SocialUserInfo, error) {
	// 1. Fetch Apple's public keys (cached)
	keys, err := fetchApplePublicKeys()
	if err != nil {
		return nil, fmt.Errorf("apple: failed to fetch public keys: %w", err)
	}

	// 2. Parse token header to get kid
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("apple: invalid JWT format")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("apple: failed to decode JWT header: %w", err)
	}

	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("apple: failed to parse JWT header: %w", err)
	}

	// 3. Find matching key by kid
	var matchedKey *appleJWK
	for i := range keys {
		if keys[i].Kid == header.Kid {
			matchedKey = &keys[i]
			break
		}
	}
	if matchedKey == nil {
		return nil, fmt.Errorf("apple: no matching public key found for kid=%q", header.Kid)
	}

	// 4. Build RSA public key from JWK
	rsaKey, err := jwkToRSAPublicKey(matchedKey)
	if err != nil {
		return nil, fmt.Errorf("apple: failed to build RSA public key: %w", err)
	}

	// 5. Parse and verify JWT signature + claims
	claims := &appleClaims{}
	token, err := jwt.ParseWithClaims(idToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return rsaKey, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("apple: token verification failed: %w", err)
	}

	// 6. Validate issuer and audience
	if claims.Issuer != "https://appleid.apple.com" {
		return nil, fmt.Errorf("apple: invalid issuer %q", claims.Issuer)
	}
	if clientID != "" {
		validAud := false
		for _, aud := range claims.Audience {
			if aud == clientID {
				validAud = true
				break
			}
		}
		if !validAud {
			return nil, fmt.Errorf("apple: token audience does not match client ID")
		}
	}

	// 7. Check expiry (jwt.ParseWithClaims already does this, extra safety)
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("apple: token is expired")
	}

	// Apple may not return email if user chose to hide it — generate a placeholder
	email := strings.ToLower(claims.Email)
	if email == "" {
		email = fmt.Sprintf("apple.%s@privaterelay.appleid.com", claims.Sub)
	}

	return &models.SocialUserInfo{
		ProviderUserID: claims.Sub,
		Email:          email,
		FirstName:      "",
		LastName:       "",
	}, nil
}

// fetchApplePublicKeys returns cached JWKS keys or fetches fresh ones from Apple.
func fetchApplePublicKeys() ([]appleJWK, error) {
	// Read from cache first
	appleKeysMu.RLock()
	if len(cachedAppleKeys) > 0 && time.Since(appleKeysCachedAt) < appleKeysCacheTTL {
		keys := cachedAppleKeys
		appleKeysMu.RUnlock()
		return keys, nil
	}
	appleKeysMu.RUnlock()

	// Fetch fresh keys
	resp, err := oauthClient.Get(applePublicKeysURL) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jwks appleJWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	// Update cache
	appleKeysMu.Lock()
	cachedAppleKeys = jwks.Keys
	appleKeysCachedAt = time.Now()
	appleKeysMu.Unlock()

	return jwks.Keys, nil
}

// jwkToRSAPublicKey converts an Apple JWK entry to an *rsa.PublicKey.
func jwkToRSAPublicKey(k *appleJWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode N: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode E: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)

	var eInt int
	for _, b := range eBytes {
		eInt = eInt<<8 | int(b)
	}

	return &rsa.PublicKey{N: n, E: eInt}, nil
}
