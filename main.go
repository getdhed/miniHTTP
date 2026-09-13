package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"
)

const readHeaderTimeout = 3 * time.Second

type Server struct {
	httpServer *http.Server
	mux        *http.ServeMux
}

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	s := &Server{
		mux: mux,
	}
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           logMiddleware(recoverMiddleware(mux)),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	return s
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := &responseWriter{
			ResponseWriter: w,
		}
		start := time.Now()
		defer func() {
			duration := time.Since(start)
			fmt.Println(r.Method, r.URL.Path, wr.status, wr.bytes, duration)

		}()

		next.ServeHTTP(wr, r)

	})
}

func testMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		fmt.Println("before")

		next.ServeHTTP(w, r)

		fmt.Println("after")
	})
}
func (s *Server) routes() {
	s.mux.HandleFunc("/health", myHandler)
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				fmt.Println("panic:", v)
				http.Error(w, "Internal server error", 500)
			}
		}()
		next.ServeHTTP(w, r)

	})
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

	<-shutdownCtx.Done()
	fmt.Println("Ctrl+C received")
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(timeoutCtx); err != nil {
		fmt.Println(err)
	}
}
