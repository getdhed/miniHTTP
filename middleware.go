package main

import (
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

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		fmt.Println("before")

		next.ServeHTTP(w, r)

		fmt.Println("after")
	})
}
