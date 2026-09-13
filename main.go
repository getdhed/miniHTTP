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

	"golang.org/x/crypto/bcrypt"
)

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
	s.mux.HandleFunc("/auth/login", loginHandler)
}

func main() {
	ctx := context.Background()
	server := NewServer(":3030")
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

	password := "qwerty123"

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	hash2, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(hash))
	err = bcrypt.CompareHashAndPassword(hash, []byte("qwerty123"))
	fmt.Println(err)

	err = bcrypt.CompareHashAndPassword(hash, []byte("qwerty124"))
	fmt.Println(err)
	fmt.Println(string(hash2) == string(hash))

	<-shutdownCtx.Done()
	fmt.Println("Ctrl+C received")
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(timeoutCtx); err != nil {
		fmt.Println(err)
	}
}
