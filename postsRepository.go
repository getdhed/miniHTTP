package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PGPostsRepository struct {
	db *pgxpool.Pool
}

func NewPgPostRepository(db *pgxpool.Pool) *PGPostsRepository {
	return &PGPostsRepository{
		db: db,
	}
}

type PostRepository interface {
	FindByID(ctx context.Context, postID int64) (Post, error)
	FindAll(ctx context.Context) ([]Post, error)
	FindByUserID(ctx context.Context, userID int64) ([]Post, error)
	CreatePost(ctx context.Context, post Post) error
	DeletePost(ctx context.Context, postID int64, userID int64) (bool, error)
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

func (r *PGPostsRepository) DeletePost(ctx context.Context, postID int64, userID int64) (bool, error) {
	tag, err := r.db.Exec(ctx, `
	DELETE FROM POSTS
	WHERE id=$1
	AND user_id=$2;
	`, postID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
