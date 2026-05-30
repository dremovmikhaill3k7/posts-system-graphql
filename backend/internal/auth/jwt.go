package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const CookieName = "auth_token"

var (
	ErrInvalidToken = errors.New("invalid token")
)

type JWT struct {
	secret     []byte
	expiration time.Duration
	secure     bool
	domain     string
}

type Claims struct {
	UserID int `json:"uid"`
	jwt.RegisteredClaims
}

func NewJWT(secret string, expiration time.Duration, secure bool, domain string) (*JWT, error) {
	if secret == "" {
		return nil, fmt.Errorf("JWT secret must not be empty")
	}
	if expiration <= 0 {
		expiration = 7 * 24 * time.Hour
	}
	return &JWT{
		secret:     []byte(secret),
		expiration: expiration,
		secure:     secure,
		domain:     domain,
	}, nil
}

func (j *JWT) Issue(userID int) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.expiration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWT) Parse(tokenString string) (int, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return j.secret, nil
	})
	if err != nil {
		return 0, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.UserID <= 0 {
		return 0, ErrInvalidToken
	}
	return claims.UserID, nil
}

func (j *JWT) SetAuthCookie(w http.ResponseWriter, userID int) error {
	token, err := j.Issue(userID)
	if err != nil {
		return err
	}
	http.SetCookie(w, j.newCookie(token, int(j.expiration.Seconds())))
	return nil
}

func (j *JWT) ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, j.newCookie("", -1))
}

func (j *JWT) newCookie(value string, maxAge int) *http.Cookie {
	c := &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   j.secure,
		SameSite: http.SameSiteLaxMode,
	}
	if j.domain != "" {
		c.Domain = j.domain
	}
	return c
}

func (j *JWT) UserIDFromRequest(r *http.Request) (int, bool) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return 0, false
	}
	id, err := j.Parse(cookie.Value)
	if err != nil {
		return 0, false
	}
	return id, true
}
