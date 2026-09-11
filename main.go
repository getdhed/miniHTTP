package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"
)

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
		Addr:    addr,
		Handler: mux,
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

func (s *Server) routes() {
	s.mux.HandleFunc("/health", myHandler)
}
func main() {
	ctx := context.Background()
	server := NewServer(":3030")
	server.routes()
	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	go func() {
		if err := server.Start(); err != nil {
			fmt.Println(err)
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
