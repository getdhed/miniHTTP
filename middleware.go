package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

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

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				fmt.Println("panic:", v)
				wr, ok := w.(*responseWriter)
				if ok && wr.wroteHeader {
					fmt.Println("Произошла паника но ответ уже улетел к клиенту => не меняем статус код")
					return
				}
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)

			}
		}()
		next.ServeHTTP(w, r)

	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := extractBearerToken(r)
		if err != nil {
			fmt.Println("ошибка", err)
			w.Header().Set("WWW-Authenticate", "Bearer")
			if err := writeJSON(w, http.StatusUnauthorized, "invalid data"); err != nil {
				fmt.Println("ошибка", err)
			}
			return
		}
		userID, err := s.parseAccessToken(tokenString)
		if err != nil {
			fmt.Println("ошибка", err)
			w.Header().Set("WWW-Authenticate", "Bearer")
			if err := writeJSON(w, http.StatusUnauthorized, "invalid data"); err != nil {
				fmt.Println("ошибка", err)
			}
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.RemoteAddr)

		next.ServeHTTP(w, r)
	})
}
