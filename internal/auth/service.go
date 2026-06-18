package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterInput struct {
	FullName          string
	PhoneNumber       string
	District          string
	PreferredLanguage string
	Password          string
}

type LoginInput struct {
	PhoneNumber string
	Password    string
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         UserView `json:"user"`
}

type UserView struct {
	ID                uuid.UUID `json:"id"`
	FullName          string    `json:"full_name"`
	PhoneNumber       string    `json:"phone_number"`
	District          string    `json:"district"`
	PreferredLanguage string    `json:"preferred_language"`
	Role              string    `json:"role"`
}

type UpdatePreferencesInput struct {
	UserID            uuid.UUID
	FullName          *string
	District          *string
	PreferredLanguage *string
}

type Service interface {
	Register(ctx context.Context, input RegisterInput) (*TokenPair, error)
	Login(ctx context.Context, input LoginInput) (*TokenPair, error)
	RefreshToken(ctx context.Context, refreshTokenStr string) (*TokenPair, error)
	Logout(ctx context.Context, userID, refreshTokenID uuid.UUID) error
	GetUser(ctx context.Context, userID uuid.UUID) (*UserView, error)
	UpdatePreferences(ctx context.Context, input UpdatePreferencesInput) (*UserView, error)
	TransferAnonymousData(ctx context.Context, anonymousID, userID uuid.UUID) error
	NormalizePhone(phone string) string
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type service struct {
	db               *gorm.DB
	accessSecret     string
	refreshSecret    string
	accessDuration   time.Duration
	refreshDuration  time.Duration
}

func NewService(db *gorm.DB, accessSecret, refreshSecret string, accessDuration, refreshDuration time.Duration) Service {
	return &service{
		db:              db,
		accessSecret:    accessSecret,
		refreshSecret:   refreshSecret,
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
	}
}

func (s *service) NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	if strings.HasPrefix(phone, "00") {
		phone = "+" + phone[2:]
	}
	if strings.HasPrefix(phone, "0") && !strings.HasPrefix(phone, "+") {
		phone = "+232" + phone[1:]
	}
	if !strings.HasPrefix(phone, "+") {
		phone = "+232" + phone
	}
	return phone
}

func (s *service) Register(ctx context.Context, input RegisterInput) (*TokenPair, error) {
	if input.FullName == "" {
		return nil, errors.New("full name is required")
	}
	if input.Password == "" {
		return nil, errors.New("password is required")
	}
	if len(input.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}

	phone := s.NormalizePhone(input.PhoneNumber)
	if phone == "" {
		return nil, errors.New("phone number is required")
	}

	lang := input.PreferredLanguage
	if lang == "" {
		lang = "english"
	}

	var existing User
	if err := s.db.WithContext(ctx).Where("phone_number = ?", phone).First(&existing).Error; err == nil {
		return nil, errors.New("phone number already registered")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("database error: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &User{
		ID:                uuid.New(),
		FullName:          input.FullName,
		PhoneNumber:       phone,
		PasswordHash:      string(hash),
		District:          input.District,
		PreferredLanguage: lang,
		Role:              "farmer",
		IsActive:          true,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	tokens, err := s.generateTokenPair(user)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *service) Login(ctx context.Context, input LoginInput) (*TokenPair, error) {
	phone := s.NormalizePhone(input.PhoneNumber)

	var user User
	if err := s.db.WithContext(ctx).Where("phone_number = ?", phone).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid phone number or password")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, errors.New("invalid phone number or password")
	}

	tokens, err := s.generateTokenPair(&user)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *service) RefreshToken(ctx context.Context, refreshTokenStr string) (*TokenPair, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(refreshTokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.refreshSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	tokenHash := hashToken(refreshTokenStr)

	var storedToken RefreshToken
	if err := s.db.WithContext(ctx).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).First(&storedToken).Error; err != nil {
		return nil, errors.New("refresh token revoked or not found")
	}

	if time.Now().UTC().After(storedToken.ExpiresAt) {
		return nil, errors.New("refresh token expired")
	}

	now := time.Now().UTC()
	storedToken.RevokedAt = &now
	s.db.WithContext(ctx).Save(&storedToken)

	var user User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", claims.UserID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	return s.generateTokenPair(&user)
}

func (s *service) Logout(ctx context.Context, userID, refreshTokenID uuid.UUID) error {
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", refreshTokenID, userID).
		Update("revoked_at", now)
	return result.Error
}

func (s *service) GetUser(ctx context.Context, userID uuid.UUID) (*UserView, error) {
	var user User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &UserView{
		ID:                user.ID,
		FullName:          user.FullName,
		PhoneNumber:       user.PhoneNumber,
		District:          user.District,
		PreferredLanguage: user.PreferredLanguage,
		Role:              user.Role,
	}, nil
}

func (s *service) UpdatePreferences(ctx context.Context, input UpdatePreferencesInput) (*UserView, error) {
	var user User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", input.UserID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if input.FullName != nil {
		user.FullName = *input.FullName
	}
	if input.District != nil {
		user.District = *input.District
	}
	if input.PreferredLanguage != nil {
		user.PreferredLanguage = *input.PreferredLanguage
	}
	user.UpdatedAt = time.Now().UTC()

	if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return &UserView{
		ID:                user.ID,
		FullName:          user.FullName,
		PhoneNumber:       user.PhoneNumber,
		District:          user.District,
		PreferredLanguage: user.PreferredLanguage,
		Role:              user.Role,
	}, nil
}

func (s *service) TransferAnonymousData(ctx context.Context, anonymousID, userID uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("UPDATE ai_conversations SET user_id = ? WHERE user_id = ?", userID, anonymousID).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE crop_diagnoses SET user_id = ? WHERE user_id = ?", userID, anonymousID).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE transcription_feedback SET user_id = ? WHERE user_id = ?", userID, anonymousID).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *service) generateTokenPair(user *User) (*TokenPair, error) {
	now := time.Now().UTC()

	accessClaims := &Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString([]byte(s.accessSecret))
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	refreshID := uuid.New()
	refreshClaims := &Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        refreshID.String(),
			Subject:   user.ID.String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString([]byte(s.refreshSecret))
	if err != nil {
		return nil, fmt.Errorf("signing refresh token: %w", err)
	}

	rtHash := hashToken(refreshTokenStr)
	storedRefresh := &RefreshToken{
		ID:        refreshID,
		UserID:    user.ID,
		TokenHash: rtHash,
		ExpiresAt: now.Add(s.refreshDuration),
		CreatedAt: now,
	}

	if err := s.db.Create(storedRefresh).Error; err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		User: UserView{
			ID:                user.ID,
			FullName:          user.FullName,
			PhoneNumber:       user.PhoneNumber,
			District:          user.District,
			PreferredLanguage: user.PreferredLanguage,
			Role:              user.Role,
		},
	}, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func ValidateToken(tokenStr string, secret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func GenerateRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
