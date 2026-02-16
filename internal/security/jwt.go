package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // access token seconds
}

type JWTManager struct {
	signingKey []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJWTManager(signingKey string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		signingKey: []byte(signingKey),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

type AccessClaims struct {
	jwt.RegisteredClaims
	Role   string `json:"role"`    // driver | dispatcher
	UserID string `json:"user_id"` // UUID
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Role   string `json:"role"`
	UserID string `json:"user_id"`
	JTI    string `json:"jti"`
}

func (m *JWTManager) Issue(role string, userID uuid.UUID) (Tokens, RefreshClaims, error) {
	now := time.Now()

	accessClaims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
		Role:   role,
		UserID: userID.String(),
	}
	access := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := access.SignedString(m.signingKey)
	if err != nil {
		return Tokens{}, RefreshClaims{}, err
	}

	jti := uuid.NewString()
	refreshClaims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		},
		Role:   role,
		UserID: userID.String(),
		JTI:    jti,
	}
	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := refresh.SignedString(m.signingKey)
	if err != nil {
		return Tokens{}, RefreshClaims{}, err
	}

	return Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(m.accessTTL.Seconds()),
	}, refreshClaims, nil
}

func (m *JWTManager) ParseAccess(tokenStr string) (userID uuid.UUID, role string, err error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.signingKey, nil
	})
	if err != nil {
		return uuid.Nil, "", err
	}
	claims, ok := tok.Claims.(*AccessClaims)
	if !ok || !tok.Valid {
		return uuid.Nil, "", fmt.Errorf("invalid token")
	}
	idStr := claims.UserID
	if idStr == "" {
		idStr = claims.Subject
	}
	uid, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, "", err
	}
	r := claims.Role
	if r == "" {
		r = "driver"
	}
	return uid, r, nil
}

func (m *JWTManager) ParseRefresh(tokenStr string) (RefreshClaims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &RefreshClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.signingKey, nil
	})
	if err != nil {
		return RefreshClaims{}, err
	}
	claims, ok := tok.Claims.(*RefreshClaims)
	if !ok || !tok.Valid {
		return RefreshClaims{}, fmt.Errorf("invalid token")
	}
	if claims.UserID == "" {
		claims.UserID = claims.Subject
	}
	if claims.Role == "" {
		claims.Role = "driver"
	}
	return *claims, nil
}
