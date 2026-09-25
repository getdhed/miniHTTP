package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
)

type contextKey string

const userIDKey contextKey = "userID"

const readHeaderTimeout = 3 * time.Second

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", myHandler)
	s.mux.HandleFunc("GET /success", successHandler)
	s.mux.HandleFunc("GET /error", errorHandler)
	s.mux.HandleFunc("/auth/login", s.loginHandler)
	s.mux.HandleFunc("POST /register", s.registerHandler)
	s.mux.HandleFunc("/auth/refresh", s.refreshHandler)
	s.mux.HandleFunc("/auth/logout", s.logoutHandler)
	s.mux.HandleFunc("POST /posts",s.postsHandler)
	s.mux.Handle(
		"GET /me",
		s.authMiddleware(http.HandlerFunc(s.meHandler)),
	)

}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using environment variables")
	}
	jwtSecret, ok := os.LookupEnv("JWT_SECRET")
	if !ok || jwtSecret == "" {
		log.Println("JWT_SECRET is not set")
	}
	databaseUrl, ok := os.LookupEnv("DATABASE_URL")
	if !ok || databaseUrl == "" {
		log.Println("DATABASE_URL is not set")
	}
	ctx := context.Background()

	pool, err := connectDB(ctx, databaseUrl)
	if err != nil {
		fmt.Println("База данных не подключена...")
		return
	}
	defer pool.Close()

	redisAddr, ok := os.LookupEnv("REDIS_ADDR")
	if !ok || redisAddr == "" {
		fmt.Println("редис не подключен...")
		return
	}
	redisClient, err := connectRedis(ctx, redisAddr)
	if err != nil {
		fmt.Println("Redis connection failed:", err)
		return
	}
	defer redisClient.Close()
	jwtConfig := JWTConfig{
		Secret: []byte(jwtSecret),
		TTL:    3600,
	}
	sessions := NewSessionStore(redisClient)

	userRepo := NewUserRepository(pool)
	authServise := NewAuthService(userRepo, sessions, jwtConfig)

	server := NewServer(":3030", []byte(jwtSecret), userRepo, authServise)
	server.routes()
	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	go func() {
		if err := server.Start(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				fmt.Println("Сервер закрылся!")
			} else {
				fmt.Println("Неожиданная ошибка", err)
			}

		}
	}()

	<-shutdownCtx.Done()
	fmt.Println("Ctrl+C received")
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(timeoutCtx); err != nil {
		fmt.Println(err)
	}
}
