package main

import (
	"errors"
	"net/http"
	"time"
)

var errInvalidData = errors.New("invalid data")

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
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
