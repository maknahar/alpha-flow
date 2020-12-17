package models

import (
	"context"
	"crypto/md5" //nolint:gosec
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserDetails represents all user info. Secret is populated if accessToken is given.
type UserDetails struct {
	ID                int64
	Email             string
	Secret            sql.NullString
	Token             sql.NullString
	TokenCreationTime time.Time
	CreatedAt         *time.Time
	UpdatedAt         sql.NullTime
}

//go:generate mockgen -destination=userStoreMock.go -package=models . UserStore
type UserModel interface {
	Create(ctx context.Context, email, Password string) (*UserDetails, error)

	Login(ctx context.Context, email, Password string) (*UserDetails, error)

	ByEmail(ctx context.Context, email string) (*UserDetails, error)

	GetDetails(ctx context.Context, accessToken string) (*UserDetails, error)

	CreateSubscription(ctx context.Context, userID int64, pair string) error

	ChangeCredentials(ctx context.Context, emailID, password, accessToken string, id int64) (*UserDetails, error)
}

type users struct {
	UserDetails
	db *sql.DB
}

func (u users) load(ctx context.Context, id int64) (*UserDetails, error) {
	user := UserDetails{ID: id}

	query := "SELECT email, secret, token, token_creation_time, created_at, updated_at from users where id=$1"

	err := u.db.QueryRowContext(ctx, query, id).Scan(&user.Email, &user.Secret, &user.Token, &user.TokenCreationTime,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (u users) ByEmail(ctx context.Context, email string) (*UserDetails, error) {
	var id int64

	query := "SELECT id from users where email=$1"

	err := u.db.QueryRowContext(ctx, query, email).Scan(&id)
	if err != nil {
		return nil, err
	}

	return u.load(ctx, id)
}

func (u users) Create(ctx context.Context, email, password string) (*UserDetails, error) {
	var id int64

	query := `INSERT INTO users(email, password) VALUES ($1, crypt($2, gen_salt('bf'))) ON CONFLICT("email") DO UPDATE SET email=EXCLUDED.email RETURNING id`

	err := u.db.QueryRowContext(ctx, query, email, password).Scan(&id)
	if err != nil {
		return nil, err
	}

	return u.load(ctx, id)
}

func (u users) Login(ctx context.Context, email, password string) (*UserDetails, error) {
	var id int64

	query := "SELECT id from users where email=$1 and password=crypt($2, password)"

	err := u.db.QueryRowContext(ctx, query, email, password).Scan(&id)
	if err != nil {
		return nil, err
	}

	accessToken := fmt.Sprintf("%x", md5.Sum([]byte(uuid.New().String()))) //nolint:gosec
	query = "UPDATE users set token=$1, token_creation_time=$2 where id=$3"

	res, err := u.db.ExecContext(ctx, query, accessToken, time.Now().UTC(), id)
	if err != nil {
		fmt.Println(query, accessToken, time.Now().UTC(), id)
		return nil, err
	}

	if n, err := res.RowsAffected(); n != 1 || err != nil {
		return nil, fmt.Errorf("error in generating login credentilas. %w: %d", err, n)
	}

	return u.load(ctx, id)
}

func (u users) GetDetails(ctx context.Context, accessToken string) (*UserDetails, error) {
	var id int64

	query := "SELECT id from users where token=$1"

	err := u.db.QueryRowContext(ctx, query, accessToken).Scan(&id)
	if err != nil {
		return nil, err
	}

	return u.load(ctx, id)
}

func (u users) ChangeCredentials(ctx context.Context, email, password, accessToken string, id int64) (*UserDetails, error) {
	query := "SELECT id from users where token=$1 and id=$2"

	err := u.db.QueryRowContext(ctx, query, accessToken, id).Scan(&id)
	if err != nil {
		return nil, err
	}

	if email == "" && password == "" {
		return u.load(ctx, id)
	}

	var qa queryArgs

	query = "UPDATE users set "

	if email != "" {
		query += " email=" + qa.Append(email)
	}

	if password != "" {
		query += " password= crypt( " + qa.Append(password) + " ,gen_salt('bf'))"
	}

	query += " where id = " + qa.Append(id)

	res, err := u.db.ExecContext(ctx, query, qa...)
	if err != nil {
		return nil, err
	}

	if n, err := res.RowsAffected(); n != 1 || err != nil {
		return nil, fmt.Errorf("error in changing login credentilas. %w: %d", err, n)
	}

	return u.load(ctx, id)
}

func (u users) CreateSubscription(ctx context.Context, userID int64, pair string) error {
	query := `INSERT INTO subscriptions(user_id, pair) VALUES ($1, $2) ON CONFLICT(user_id, pair) DO NOTHING`

	_, err := u.db.ExecContext(ctx, query, userID, pair)
	if err != nil {
		return err
	}

	return nil
}

func NewUser(db *sql.DB) UserModel {
	return &users{db: db}
}
