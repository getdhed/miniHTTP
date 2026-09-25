package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type responseWriter struct {
	http.ResponseWriter

	status      int
	wroteHeader bool
	bytes       int
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type Server struct {
	httpServer  *http.Server
	mux         *http.ServeMux
	jwtConfig   JWTConfig
	users       *UserRepository
	auth        *AuthService
	RateLimiter RateLimiter
	posts       PostRepository
}

type UserRepository struct {
	db *pgxpool.Pool
}

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
}

type JWTConfig struct {
	Secret []byte
	TTL    time.Duration
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type User struct {
	ID           int
	Email        string
	PasswordHash string `json:"-"`
	Name         string
	CreatedAt    time.Time
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
type Session struct {
	ID          string
	UserID      int `json:"user_id"`
	RefreshHash string
}

type AuthService struct {
	users         *UserRepository
	sessions      *SessionStore
	jwtConfig     JWTConfig
	LoginAttempts LoginAttemptsLimiter
}
type Post struct {
	ID        int64         `json:"id"`
	UserID    int64         `json:"user_id"`
	Title     string        `json:"title"`
	Content   string        `json:"content"`
	CreatedAt time.Duration `json:"created_at"`
	UpdatedAt time.Duration `json:"updated_at"`
}

type LoginAttemptsLimiter interface {
	IsBlocked(ctx context.Context, key string, limit int64) (bool, time.Duration, error)

	RegisterFailure(ctx context.Context, key string, limit int64, window time.Duration) (bool, time.Duration, error)

	Reset(ctx context.Context, key string) error
}
type RedisLoginAttemptLimiter struct {
	client *redis.Client
}

func NewRedisLoginAttemptLimiter(client *redis.Client) *RedisLoginAttemptLimiter {
	return &RedisLoginAttemptLimiter{
		client: client,
	}
}

func NewAuthService(users *UserRepository,
	sessions *SessionStore,
	jwtConfig JWTConfig,
	loginAttempts *RedisLoginAttemptLimiter,
) *AuthService {
	return &AuthService{
		users:     users,
		sessions:  sessions,
		jwtConfig: jwtConfig,
		//LoginAttempts: loginAttempts,
	}
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type RefreshResponse struct {
	RefreshToken string
	AccessToken  string
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, time.Duration, error)
}
type RedisRateLimiter struct {
	client *redis.Client
}

func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
	}
}

func (r *RedisLoginAttemptLimiter) Reset(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

var errInvalidData = errors.New("invalid data")

// type UserStore struct {
// 	users map[string]User
// }

// func NewUserStore() (*UserStore, error) {
// 	us := UserStore{
// 		users: make(map[string]User),
// 	}
// 	hash, err := bcrypt.GenerateFromPassword([]byte("qwerty123"), bcrypt.DefaultCost)
// 	if err != nil {
// 		return nil, err
// 	}
// 	testUser := User{
// 		ID:           1,
// 		Email:        "testmail",
// 		PasswordHash: string(hash),
// 	}
// 	us.users["testmail"] = testUser
// 	return &us, nil
// }

// func (us *UserStore) FindByEmail(email string) (User, bool) {
// 	user, ok := us.users[email]
// 	return user, ok
// }

func NewServer(addr string,
	jwtSecret []byte,
	users *UserRepository,
	posts *PGPostsRepository,
	authServise *AuthService,
	rateLimiter *RedisRateLimiter,
) *Server {
	mux := http.NewServeMux()
	TTL := time.Hour
	jwtConfig := JWTConfig{
		Secret: jwtSecret,
		TTL:    TTL,
	}

	s := &Server{
		mux:         mux,
		jwtConfig:   jwtConfig,
		users:       users,
		posts:       posts,
		auth:        authServise,
		RateLimiter: rateLimiter,
	}
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           logMiddleware(recoverMiddleware(mux)),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	return s
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	ur := UserRepository{
		db: pool,
	}
	return &ur
}
