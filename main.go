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
	s.mux.Handle(
		"GET /me",
		s.authMiddleware(http.HandlerFunc(s.meHandler)),
	)
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("failed to load .env")
	}
	jwtSecret, ok := os.LookupEnv("JWT_SECRET")
	if !ok || jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}
	ctx := context.Background()
	server := NewServer(":3030", []byte(jwtSecret))
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
