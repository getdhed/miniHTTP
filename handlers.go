package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func myHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"message": "hello bro 4",
		"id":      12,
	}
	if err := writeJSON(w, http.StatusOK, data); err != nil {
		fmt.Println("write response error:", err)
	}
}

func successHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"message": "everything is ok",
		"id":      123,
	}
	if err := writeJSON(w, http.StatusOK, data); err != nil {
		fmt.Println("write response error:", err)
	}

}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	if err := writeError(w, http.StatusNotFound, "user_not_found", "user_not_found"); err != nil {
		fmt.Println("write response error:", err)
	}

}

func confirmEmail(email string) bool {
	return email != ""
}

func confirmPassword(password string) bool {
	return password != ""
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err := writeError(w, http.StatusBadRequest, "bad request", "bad request"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return

	}

	if !confirmPassword(req.Password) || !confirmEmail(req.Email) {
		if err := writeError(w, http.StatusBadRequest, "bad request", "bad request"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}
	user, isExist := s.userStore.FindByEmail(req.Email)
	if !isExist {
		if err := writeError(w, http.StatusUnauthorized, "Unauthorized", "Unauthorized"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		if err := writeError(w, http.StatusUnauthorized, "Unauthorized", "Unauthorized"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}
	if err := writeJSON(w, 200, user); err != nil {
		fmt.Println("Произошла ошибка", err)
	}
}
