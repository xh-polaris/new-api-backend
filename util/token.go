package util

import (
	"crypto/rsa"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var private *rsa.PrivateKey
var public *rsa.PublicKey

func InitKey() {
	var err error
	pri, pub := os.Getenv("PRIVATE_KEY"), os.Getenv("PUBLIC_KEY")
	pub = pub[0:26] + "\n\n" + pub[26:len(pub)-24] + "\n\n" + pub[len(pub)-24:]
	pri = pri[0:27] + "\n\n" + pri[27:len(pri)-25] + "\n\n" + pri[len(pri)-25:]
	public, err = jwt.ParseRSAPublicKeyFromPEM([]byte(pub))
	if err != nil {
		panic("failed to parse public key: %v" + err.Error())
	}
	private, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(pri))
	if err != nil {
		panic("failed to parse private key: %v" + err.Error())
	}
}

type Claims struct {
	ID              int    `json:"id,omitempty"`
	Username        string `json:"username,omitempty"`
	Role            int    `json:"role,omitempty"`
	Status          int    `json:"status,omitempty"`
	Group           string `json:"group,omitempty"`
	Turnstile       bool   `json:"turnstile,omitempty"`
	PendingUsername string `json:"pending_username,omitempty"`
	PendingUserID   string `json:"pending_user_id,omitempty"`
	AffCode         string `json:"aff,omitempty"`
	OAuthState      string `json:"oauth_state,omitempty"`
	jwt.RegisteredClaims
}

func GenerateToken(id int, username string, role int, status int, group string) (string, *Claims, error) {
	claims := Claims{
		ID:       id,
		Username: username,
		Role:     role,
		Status:   status,
		Group:    group,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	str, err := token.SignedString(private)
	return str, &claims, err
}

func ParseToken(tokenString string) (claim *Claims, err error) {
	var token *jwt.Token
	if token, err = jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return public, nil
	}); err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
