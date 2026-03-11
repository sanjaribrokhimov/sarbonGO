package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"sarbonNew/internal/timeutil"
)

type Tokens struct {
	AccessToken       string `json:"access_token"`
	RefreshToken      string `json:"refresh_token"`
	ExpiresIn         int64  `json:"expires_in"`          // время жизни access токена (секунды)
	ExpiresAt         int64  `json:"expires_at"`           // Unix timestamp в миллисекундах (ms) — когда истекает access токен; для JS: new Date(expires_at)
	RefreshExpiresIn  int64  `json:"refresh_expires_in"`  // время жизни refresh токена (секунды)
	RefreshExpiresAt  int64  `json:"refresh_expires_at"`   // Unix timestamp в миллисекундах (ms) — когда истекает refresh токен; для JS: new Date(refresh_expires_at)
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
	Role      string `json:"role"`       // driver | dispatcher | admin | user
	UserID    string `json:"user_id"`    // UUID
	CompanyID string `json:"company_id"` // UUID, optional (for app user after switch-company)
	SID       string `json:"sid"`        // session id (= refresh JTI); для инвалидации после refresh
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Role   string `json:"role"`
	UserID string `json:"user_id"`
	JTI    string `json:"jti"`
}

func (m *JWTManager) Issue(role string, userID uuid.UUID) (Tokens, RefreshClaims, error) {
	return m.IssueWithCompany(role, userID, uuid.Nil)
}

// IssueWithCompany issues tokens with optional company_id in access claims (for app user switch-company).
// Текущее время берётся по Ташкенту (Asia/Tashkent). В access кладётся sid = JTI refresh — для инвалидации после refresh.
func (m *JWTManager) IssueWithCompany(role string, userID uuid.UUID, companyID uuid.UUID) (Tokens, RefreshClaims, error) {
	now := timeutil.NowTashkent()
	jti := uuid.NewString()
	accessClaims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
		Role:   role,
		UserID: userID.String(),
		SID:    jti,
	}
	if companyID != uuid.Nil {
		accessClaims.CompanyID = companyID.String()
	}
	access := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := access.SignedString(m.signingKey)
	if err != nil {
		return Tokens{}, RefreshClaims{}, err
	}

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

	accessExpiresAt := now.Add(m.accessTTL)
	refreshExpiresAt := now.Add(m.refreshTTL)
	return Tokens{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        int64(m.accessTTL.Seconds()),
		ExpiresAt:        accessExpiresAt.UnixMilli(),
		RefreshExpiresIn: int64(m.refreshTTL.Seconds()),
		RefreshExpiresAt: refreshExpiresAt.UnixMilli(),
	}, refreshClaims, nil
}

// ParseAccess returns userID, role (driver|dispatcher|admin|user).
func (m *JWTManager) ParseAccess(tokenStr string) (userID uuid.UUID, role string, err error) {
	uid, r, _, _, err := m.ParseAccessWithSID(tokenStr)
	return uid, r, err
}

// ParseAccessWithCompany returns userID, role, companyID (may be Nil), error.
func (m *JWTManager) ParseAccessWithCompany(tokenStr string) (userID uuid.UUID, role string, companyID uuid.UUID, err error) {
	uid, r, cid, _, err := m.ParseAccessWithSID(tokenStr)
	return uid, r, cid, err
}

// ParseAccessWithSID returns userID, role, companyID, sid (session id for revocation check), error.
func (m *JWTManager) ParseAccessWithSID(tokenStr string) (userID uuid.UUID, role string, companyID uuid.UUID, sid string, err error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.signingKey, nil
	})
	if err != nil {
		return uuid.Nil, "", uuid.Nil, "", err
	}
	claims, ok := tok.Claims.(*AccessClaims)
	if !ok || !tok.Valid {
		return uuid.Nil, "", uuid.Nil, "", fmt.Errorf("invalid token")
	}
	idStr := claims.UserID
	if idStr == "" {
		idStr = claims.Subject
	}
	uid, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, "", uuid.Nil, "", err
	}
	r := claims.Role
	if r == "" {
		r = "driver"
	}
	var cid uuid.UUID
	if claims.CompanyID != "" {
		cid, _ = uuid.Parse(claims.CompanyID)
	}
	return uid, r, cid, claims.SID, nil
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
