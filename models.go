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
}

type UserRepository struct {
	db *pgxpool.Pool
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
		users:         users,
		sessions:      sessions,
		jwtConfig:     jwtConfig,
		LoginAttempts: loginAttempts,
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

func (r *RedisLoginAttemptLimiter) IsBlocked(
	ctx context.Context,
	key string,
	limit int64,
) (bool, time.Duration, error) {
	if limit <= 0 {
		return false, 0, errInvalidData
	}

	count, err := r.client.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	if count < limit {
		return false, 0, nil
	}

	ttl, err := r.client.PTTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	return true, ttl, nil
}

func (r *RedisLoginAttemptLimiter) RegisterFailure(
	ctx context.Context,
	key string,
	limit int64,
	window time.Duration,
) (bool, time.Duration, error) {
	windowSeconds := int64(window / time.Second)
	script := redis.NewScript(`
	local count = redis.call("INCR", KEYS[1])

	if count == 1 then
		redis.call("EXPIRE", KEYS[1], ARGV[1])
	end

	local ttl = redis.call("PTTL", KEYS[1])

	return {count, ttl}
	`)

	values, err := script.Run(ctx, r.client, []string{key}, windowSeconds).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	if len(values) != 2 {
		return false, 0, errUnexpectedResult
	}

	count := values[0]
	ttl := time.Duration(values[1]) * time.Millisecond

	if count >= limit {
		return true, ttl, nil
	}

	return false, ttl, nil
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
