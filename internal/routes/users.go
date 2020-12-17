package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"

	"github.com/sirupsen/logrus"

	"github.com/maknahar/alpha-flow/internal/services"
)

type UserHandler struct {
	service services.UserServicer
}

func NewUsersHandler(db *sql.DB) *UserHandler {
	return &UserHandler{service: services.NewUserService(db)}
}

func (u *UserHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, func() (interface{}, int, error) {
		ctx := r.Context()
		body := &services.SignUpRequestDTO{}

		err := json.NewDecoder(r.Body).Decode(body)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		dto, err := u.service.SignUp(ctx, body)
		if err != nil {
			logrus.WithError(err).Error("Error in user signup")
			return nil, http.StatusInternalServerError, err
		}

		return dto, http.StatusOK, nil
	})
}

func (u *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, func() (interface{}, int, error) {
		ctx := r.Context()
		body := &services.LoginRequestDTO{}

		err := json.NewDecoder(r.Body).Decode(body)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		dto, err := u.service.Login(ctx, body)
		if err != nil {
			if errors.Is(err, services.ErrInvalidCredentials) {
				return nil, http.StatusUnauthorized, err
			}

			logrus.WithError(err).Error("Error in user login")
			return nil, http.StatusInternalServerError, err
		}

		return dto, http.StatusOK, nil
	})
}

func (u *UserHandler) GetSecret(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, func() (interface{}, int, error) {
		ctx := r.Context()
		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer ")
		if len(splitToken) < 2 {
			return nil, http.StatusUnauthorized, services.ErrInvalidToken
		}
		reqToken = splitToken[1]

		dto, err := u.service.GetSecret(ctx, reqToken)
		if err != nil {
			if errors.Is(err, services.ErrInvalidToken) || errors.Is(err, services.ErrExpiredToken) {
				return nil, http.StatusForbidden, err
			}

			logrus.WithError(err).Error("Error in user login")
			return nil, http.StatusInternalServerError, err
		}

		return dto, http.StatusOK, nil
	})
}

func (u *UserHandler) GetValidPairs(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, func() (interface{}, int, error) {
		ctx := r.Context()
		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer ")
		if len(splitToken) < 2 {
			return nil, http.StatusUnauthorized, services.ErrInvalidToken
		}
		reqToken = splitToken[1]

		_, err := u.service.GetSecret(ctx, reqToken)
		if err != nil {
			if errors.Is(err, services.ErrInvalidToken) || errors.Is(err, services.ErrExpiredToken) {
				return nil, http.StatusForbidden, err
			}

			logrus.WithError(err).Error("Error in user authentication")
			return nil, http.StatusInternalServerError, err
		}

		validPairs, err := u.service.GetAllValidPairs(ctx)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		return validPairs, http.StatusOK, nil
	})
}

func (u *UserHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, func() (interface{}, int, error) {
		ctx := r.Context()
		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer ")
		if len(splitToken) < 2 {
			return nil, http.StatusUnauthorized, services.ErrInvalidToken
		}
		reqToken = splitToken[1]

		userInfo, err := u.service.GetSecret(ctx, reqToken)
		if err != nil {
			if errors.Is(err, services.ErrInvalidToken) || errors.Is(err, services.ErrExpiredToken) {
				return nil, http.StatusForbidden, err
			}

			logrus.WithError(err).Error("Error in user authentication")
			return nil, http.StatusInternalServerError, err
		}

		body := &services.CreateSubscriptionDTO{}

		err = json.NewDecoder(r.Body).Decode(body)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		err = u.service.CreateSubscription(ctx, userInfo.ID, body.Pair)
		if err != nil {
			if err != services.ErrInvalidPair {
				return nil, http.StatusUnprocessableEntity, err
			}
			return nil, http.StatusInternalServerError, err
		}

		return nil, http.StatusOK, nil
	})
}

func (u *UserHandler) UpdateCredentials(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, func() (interface{}, int, error) {
		ctx := r.Context()
		body := &services.UpdateCredentialsRequestDTO{}
		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer ")
		if len(splitToken) < 2 {
			return nil, http.StatusForbidden, services.ErrInvalidToken
		}
		reqToken = splitToken[1]

		userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			return nil, http.StatusBadRequest, errors.New("invalid user id")
		}

		err = json.NewDecoder(r.Body).Decode(body)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		body.Token = reqToken
		body.ID = userID

		dto, err := u.service.Update(ctx, body)
		if err != nil {
			if errors.Is(err, services.ErrAccessDenied) || errors.Is(err, services.ErrInvalidCredentials) {
				return nil, http.StatusForbidden, err
			}

			logrus.WithError(err).Error("Error in user login")
			return nil, http.StatusInternalServerError, err
		}

		return dto, http.StatusOK, nil
	})
}
