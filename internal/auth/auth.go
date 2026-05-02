package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/luanlucolli/uy3-leads-api/internal/models"
)

type Service struct {
	db        *sql.DB
	jwtSecret []byte
}

type Claims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func NewService(db *sql.DB, jwtSecret string) (*Service, error) {
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET nao configurado")
	}
	return &Service{db: db, jwtSecret: []byte(jwtSecret)}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, models.User, error) {
	var user models.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash
		FROM users
		WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", models.User{}, fmt.Errorf("credenciais invalidas")
	}
	if err != nil {
		return "", models.User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", models.User{}, fmt.Errorf("credenciais invalidas")
	}

	now := time.Now()
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", models.User{}, err
	}

	return signedToken, user, nil
}

func (s *Service) ParseToken(rawToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(rawToken, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metodo de assinatura inesperado")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("token invalido")
	}

	return claims, nil
}
