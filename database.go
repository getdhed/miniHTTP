package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func connectDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db does not ping: %w", err)
	}
	return pool, nil
}

func (r *UserRepository) findByEmail(ctx context.Context, email string) (User, error) {
	row := r.db.QueryRow(
		ctx,
		`SELECT id,password_hash
		FROM users
		WHERE email=$1;`,
		email)
	user := User{Email: email}
	if err := row.Scan(
		&user.ID,
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
		VALUES($1,$2)
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
