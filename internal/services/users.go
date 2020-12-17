package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/maknahar/alpha-flow/internal/models"
	"github.com/maknahar/alpha-flow/internal/utils"
)

var (
	ErrInvalidEmail       = errors.New("validation error: email")
	ErrInvalidPassword    = errors.New("validation error: password")
	ErrAccountExists      = errors.New("validation error: account with given email already exists")
	ErrInvalidToken       = errors.New("token invalid")
	ErrAccessDenied       = errors.New("access denied")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrExpiredToken       = errors.New("token expired")
	ErrInvalidPair        = errors.New("invalid pair")
)

var (
	ShareShiftHostURL = "https://shapeshift.io"
)

type UserServicer interface {
	SignUp(ctx context.Context, dto *SignUpRequestDTO) (*SignUpResponseDTO, error)
	Login(ctx context.Context, dto *LoginRequestDTO) (*LoginResponseDTO, error)
	GetSecret(ctx context.Context, token string) (*GetSecretResponseDTO, error)
	Update(ctx context.Context, dto *UpdateCredentialsRequestDTO) (*SignUpResponseDTO, error)
	GetAllValidPairs(ctx context.Context) ([]string, error)
	CreateSubscription(ctx context.Context, userId int64, pair string) error
}

type user struct {
	model models.UserModel
}

func NewUserService(db *sql.DB) UserServicer {
	return &user{model: models.NewUser(db)}
}

type SignUpRequestDTO struct {
	LoginRequestDTO
}

type SignUpResponseDTO struct {
	ID        int64      `json:"id"`
	Email     string     `json:"email"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

func (u user) SignUp(ctx context.Context, dto *SignUpRequestDTO) (*SignUpResponseDTO, error) {
	if err := dto.Validate(); err != nil {
		return nil, err
	}

	_, err := u.model.ByEmail(ctx, dto.Email)
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAccountExists
	}

	userDetails, err := u.model.Create(ctx, dto.Email, dto.Password)
	if err != nil {
		return nil, err
	}

	response := &SignUpResponseDTO{
		ID:        userDetails.ID,
		Email:     userDetails.Email,
		CreatedAt: userDetails.CreatedAt,
	}

	if userDetails.UpdatedAt.Valid {
		response.UpdatedAt = &userDetails.UpdatedAt.Time
	}

	return response, nil
}

type LoginRequestDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *LoginRequestDTO) Validate() error {
	if !utils.IsEmailValid(s.Email) {
		return ErrInvalidEmail
	}

	if len(s.Password) < 8 {
		return ErrInvalidPassword
	}

	return nil
}

type LoginResponseDTO struct {
	Token string `json:"token"`
}

func (u user) Login(ctx context.Context, dto *LoginRequestDTO) (*LoginResponseDTO, error) {
	if err := dto.Validate(); err != nil {
		return nil, err
	}

	userDetails, err := u.model.Login(ctx, dto.Email, dto.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	return &LoginResponseDTO{
		Token: userDetails.Token.String,
	}, nil
}

type GetSecretResponseDTO struct {
	ID     int64  `json:"user_id"`
	Secret string `json:"secret"`
}

func (u user) GetSecret(ctx context.Context, token string) (*GetSecretResponseDTO, error) {
	userDetails, err := u.model.GetDetails(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidToken
		}

		return nil, err
	}

	if userDetails.TokenCreationTime.Before(time.Now().Add(-time.Minute * 60)) {
		return nil, ErrExpiredToken
	}

	return &GetSecretResponseDTO{
		ID:     userDetails.ID,
		Secret: userDetails.Secret.String,
	}, nil
}

type UpdateCredentialsRequestDTO struct {
	LoginRequestDTO
	Token string
	ID    int64
}

func (u *UpdateCredentialsRequestDTO) Validate() error {
	if u.Email != "" && !utils.IsEmailValid(u.Email) {
		return ErrInvalidEmail
	}

	if u.Password != "" && len(u.Password) < 8 {
		return ErrInvalidPassword
	}

	return nil
}

func (u user) Update(ctx context.Context, dto *UpdateCredentialsRequestDTO) (*SignUpResponseDTO, error) {
	if err := dto.Validate(); err != nil {
		return nil, err
	}

	_, err := u.GetSecret(ctx, dto.Token)
	if err != nil {
		return nil, err
	}

	userDetails, err := u.model.ChangeCredentials(ctx, dto.Email, dto.Password, dto.Token, dto.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccessDenied
		}

		return nil, err
	}

	response := &SignUpResponseDTO{
		ID:        userDetails.ID,
		Email:     userDetails.Email,
		CreatedAt: userDetails.CreatedAt,
		UpdatedAt: &userDetails.UpdatedAt.Time,
	}

	return response, nil
}

func (u user) GetAllValidPairs(ctx context.Context) (validPairs []string, err error) {
	res, err := http.Get(ShareShiftHostURL + "/validpairs")
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &validPairs)
	if err != nil {
		return nil, err
	}

	return validPairs, nil
}

type CreateSubscriptionDTO struct {
	Pair string `json:"pair"`
}

func (u user) CreateSubscription(ctx context.Context, userID int64, pair string) error {
	pairs, err := u.GetAllValidPairs(ctx)
	if err != nil {
		return err
	}

	validPair := false

	for _, p := range pairs {
		if pair == p {
			validPair = true
			break
		}
	}

	if !validPair {
		return ErrInvalidPair
	}

	return u.model.CreateSubscription(ctx, userID, pair)
}
