package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	ur := UserRepository{
		db: pool,
	}
	return &ur
}

func (r *UserRepository) findByEmail(ctx context.Context, email string) (User, error) {
	row := r.db.QueryRow(
		ctx,
		`SELECT id,name,password_hash
		FROM users
		WHERE email=$1;`,
		email)
	user := User{Email: email}
	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.PasswordHash,
	); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *UserRepository) AddUser(ctx context.Context, user User) (User, error) {
	row := r.db.QueryRow(
		ctx,
		`INSERT INTO users (name,email,password_hash)
		VALUES($1,$2,$3)
		RETURNING id,name,email,password_hash,created_at;`,
		user.Name,
		user.Email,
		user.PasswordHash)

	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	); err != nil {
		return User{}, err
	}

	return user, nil
}

func (r *UserRepository) findByID(ctx context.Context, userID int64) (User, error) {
	row := r.db.QueryRow(ctx, `
	SELECT id,name,email,password_hash
	FROM users
	WHERE id=$1
	`, userID)
	var user User
	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *UserRepository) editUser(ctx context.Context, user ProfileResp) (bool, error) {
	var err error
	var tag pgconn.CommandTag

	switch {
	case user.Email == "":

		tag, err = r.db.Exec(ctx, `
	UPDATE users
	SET name=$1,updated_at = CURRENT_TIMESTAMP
	WHERE id=$2;
	`, user.Name, user.ID)

	case user.Name == "":
		tag, err = r.db.Exec(ctx, `
	UPDATE users
	SET email=$1,updated_at = CURRENT_TIMESTAMP
	WHERE id=$2;
	`, user.Email, user.ID)

	default:
		tag, err = r.db.Exec(ctx, `
	UPDATE users
	SET name=$1,email=$2,updated_at = CURRENT_TIMESTAMP
	WHERE id=$3;
	`, user.Name, user.Email, user.ID)
	}

	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil

}

func (r *UserRepository) editPassword(
	ctx context.Context,
	passwordHash string,
	userID int64,
) (bool, error) {
	tag, err := r.db.Exec(ctx, `
	UPDATE users
	SET password_hash=$1,updated_at = CURRENT_TIMESTAMP
	WHERE id=$2
	`, passwordHash, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
