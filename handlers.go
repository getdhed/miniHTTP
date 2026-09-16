package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (s *Server) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err := writeError(w, http.StatusBadRequest, "bad request", "bad request"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}
	if !confirmEmail(req.Email) || !confirmPassword(req.Password) {
		if err := writeError(w, http.StatusBadRequest, "bad request", "bad request"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}
	_, err := s.users.findByEmail(r.Context(), req.Email)
	if err == nil {
		if err := writeError(w, http.StatusConflict, "already exists", "already exists"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}
	user := User{
		Email:        req.Email,
		PasswordHash: string(passwordHash),
	}
	var pgErr *pgconn.PgError
	user, err = s.users.AddUser(r.Context(), user)
	if err != nil {
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				if err := writeError(w, http.StatusConflict, "already exists", "already exists"); err != nil {
					fmt.Println("Произошла ошибка", err)
				}
				return
			}
		} else {
			if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
				fmt.Println("Произошла ошибка", err)
			}
		}
		return
	}

	token, err := s.generateAccessToken(user)
	if err != nil {
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}
	loginResponse := loginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}
	if err := writeJSON(w, 201, loginResponse); err != nil {
		fmt.Println("Произошла ошибка", loginResponse)
	}

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
	user, err := s.users.findByEmail(r.Context(), req.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := writeError(w, http.StatusUnauthorized, "Ivalid email or password", "Ivalid email or password"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}
	if err != nil {
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
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
	token, err := s.generateAccessToken(user)
	if err != nil {
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}
	loginResponse := loginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}
	if err := writeJSON(w, 200, loginResponse); err != nil {
		fmt.Println("Произошла ошибка", loginResponse)
	}
}

func (s *Server) meHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
			fmt.Println("Ошибка:", err)
		}
		return
	}
	fmt.Println("user ID:", userID)
	user := User{
		ID: userID,
	}
	if err := writeJSON(w, http.StatusOK, user); err != nil {
		fmt.Println("Ошибка:", err)
	}
}
