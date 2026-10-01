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

	s.mux.Handle("POST /auth/login",
		s.rateLimitMiddleware("login", 3, time.Hour, http.HandlerFunc(s.loginHandler)))

	s.mux.Handle("POST /auth/register",
		s.rateLimitMiddleware("register", 3, time.Hour, http.HandlerFunc(s.registerHandler)))

	s.mux.HandleFunc("GET /posts", s.getAllPostsHandler)
	s.mux.HandleFunc("GET /user/{userID}/posts", s.getUsersPostsHandler)
	s.mux.HandleFunc("GET /posts/{postID}", s.getPostHandler)
	s.mux.Handle("DELETE /posts/{postID}", s.authMiddleware(http.HandlerFunc(s.deletePostHnadler)))
	s.mux.Handle("POST /posts", s.authMiddleware(http.HandlerFunc(s.createPostHandler)))

	s.mux.HandleFunc("/auth/refresh", s.refreshHandler)
	s.mux.HandleFunc("/auth/logout", s.logoutHandler)
	s.mux.Handle("GET /me", s.authMiddleware(http.HandlerFunc(s.meHandler)))
	s.mux.Handle("PATCH /me/", s.authMiddleware(http.HandlerFunc(s.patchProfileHandler)))
	s.mux.Handle("PATCH /me/password", s.authMiddleware(http.HandlerFunc(s.changePasswordHandler)))
	// s.mux.Handle("GET /test", s.rateLimitMiddleware(http.HandlerFunc(s.testHandler)))
	//s.mux.HandleFunc("POST /posts", s.getAllPostsHandler)

}

func main() {
	ctx := context.Background()
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatal("error:", err)
	}

	app, err := buildApp(ctx, cfg)
	if err != nil {
		log.Fatal("error:", err)
	}
	defer app.Close()

	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	go func() {
		if err := app.server.Start(); err != nil {
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
	if err := app.server.Shutdown(timeoutCtx); err != nil {
		fmt.Println(err)
	}
}
