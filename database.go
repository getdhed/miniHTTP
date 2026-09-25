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

func (r *PGPostsRepository) FindByID(ctx context.Context, postID int64) (Post, error) {
	row := r.db.QueryRow(
		ctx, `
		SELECT id,user_id,title,content,created_at,updated_at
		FROM posts
		WHERE id=$1;`,
		postID)

	var post Post
	if err := row.Scan(
		&post.ID,
		&post.UserID,
		&post.Title,
		&post.Content,
		&post.CreatedAt,
		&post.UpdatedAt); err != nil {
		return Post{}, err
	}

	return post, nil
}

func (r *PGPostsRepository) FindAll(ctx context.Context) ([]Post, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id,user_id,title,content,created_at,updated_at
		FROM posts
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&post.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (r *PGPostsRepository) FindByUserID(ctx context.Context, userID int64) ([]Post, error) {
	rows, err := r.db.Query(ctx, `
	SELECT  id,user_id,title,content,created_at,updated_at
	FROM POSTS
	WHERE user_id=$1
	ORDER BY created_at DESC;
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&post.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (r *PGPostsRepository) CreatePost(ctx context.Context, newPost Post) error {
	_, err := r.db.Exec(ctx, `
	INSERT INTO posts
	(user_id,title,content)
	values($1,$2,$3);
	`, newPost.UserID, newPost.Title, newPost.Content)
	if err != nil {
		return err
	}
	return nil
}
