package main

import (
	"net/http"
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
