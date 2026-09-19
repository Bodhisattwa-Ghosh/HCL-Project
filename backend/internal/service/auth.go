package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/domain"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/repository"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/validation"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("an account with this email already exists")
)

type AuthService struct {
	store  *repository.Store
	secret []byte
	ttl    time.Duration
}

func NewAuth(store *repository.Store, secret []byte, ttl time.Duration) *AuthService {
	return &AuthService{store: store, secret: secret, ttl: ttl}
}

func (s *AuthService) SignUp(ctx context.Context, name, email, password string) (domain.User, string, error) {
	name = validation.CleanText(name, 60)
	if len([]rune(name)) < 2 {
		return domain.User{}, "", errors.New("name must be 2 to 60 characters")
	}
	var err error
	if email, err = validation.Email(email); err != nil {
		return domain.User{}, "", err
	}
	if err := validation.Password(password); err != nil {
		return domain.User{}, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, "", fmt.Errorf("hash password: %w", err)
	}
	user, err := s.store.CreateUser(ctx, domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().UTC(),
	})
	if errors.Is(err, repository.ErrDuplicate) {
		return domain.User{}, "", ErrEmailTaken
	}
	if err != nil {
		return domain.User{}, "", err
	}
	token, err := s.issue(user.ID)
	return user, token, err
}

func (s *AuthService) Login(ctx context.Context, email, password string) (domain.User, string, error) {
	email, err := validation.Email(email)
	if err != nil {
		return domain.User{}, "", ErrInvalidCredentials
	}
	user, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.User{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return domain.User{}, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return domain.User{}, "", ErrInvalidCredentials
	}
	token, err := s.issue(user.ID)
	return user, token, err
}

func (s *AuthService) UserFromToken(ctx context.Context, rawToken string) (domain.User, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return domain.User{}, ErrInvalidCredentials
	}
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected token signing method")
		}
		return s.secret, nil
	}, jwt.WithIssuer("pulsepoll"), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return domain.User{}, ErrInvalidCredentials
	}
	userID, err := primitive.ObjectIDFromHex(claims.Subject)
	if err != nil {
		return domain.User{}, ErrInvalidCredentials
	}
	user, err := s.store.UserByID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.User{}, ErrInvalidCredentials
	}
	return user, err
}

func (s *AuthService) issue(userID primitive.ObjectID) (string, error) {
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    "pulsepoll",
		Subject:   userID.Hex(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}
