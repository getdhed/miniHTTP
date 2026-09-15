package main

import (
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	httpServer *http.Server
	mux        *http.ServeMux
	userStore  *UserStore
	jwtConfig  JWTConfig
}
type JWTConfig struct {
	Secret []byte
	TTL    time.Duration
}
type User struct {
	ID           int
	Email        string
	PasswordHash string `json:"-"`
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
type UserStore struct {
	users map[string]User
}

func NewUserStore() (*UserStore, error) {
	us := UserStore{
		users: make(map[string]User),
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("qwerty123"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	testUser := User{
		ID:           1,
		Email:        "testmail",
		PasswordHash: string(hash),
	}

	us.users["testmail"] = testUser
	return &us, nil
}

func (us *UserStore) FindByEmail(email string) (User, bool) {
	user, ok := us.users[email]
	return user, ok
}

func NewServer(addr string, jwtSecret []byte) *Server {
	mux := http.NewServeMux()
	TTL := time.Hour
	jwtConfig := JWTConfig{
		Secret: jwtSecret,
		TTL:    TTL,
	}
	userStore, err := NewUserStore()
	if err != nil {
		return nil
	}

	s := &Server{
		mux:       mux,
		userStore: userStore,
		jwtConfig: jwtConfig,
	}
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           logMiddleware(recoverMiddleware(mux)),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	return s
}
