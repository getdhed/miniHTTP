package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const refreshesTTL = 7 * 24 * time.Hour

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
		Name:         req.Name,
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
	refreshToken, err := generateRefreshToken()
	if err != nil {
		if err := writeError(
			w,
			http.StatusInternalServerError,
			"internal error",
			"failed to generate refresh token",
		); err != nil {
			fmt.Println(err)
		}
		return
	}
	refreshHash := hashRefreshToken(refreshToken)
	session := Session{
		UserID: user.ID,
	}

	if err := s.sessions.Create(
		r.Context(),
		refreshHash,
		session,
		refreshesTTL); err != nil {
		if err := writeError(
			w,
			http.StatusInternalServerError,
			"internal error",
			"failed to create session",
		); err != nil {
			fmt.Println(err)
		}
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(refreshesTTL.Seconds()),
		Secure:   false,
	})
	if err := writeJSON(w, 200, loginResponse); err != nil {
		fmt.Println("Произошла ошибка", err)
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

func (s *Server) refreshHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if errors.Is(err, http.ErrNoCookie) {
		_ = writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"refresh token not found",
		)
		return
	}

	if err != nil {
		_ = writeError(
			w,
			http.StatusBadRequest,
			"bad request",
			"invalid cookie",
		)
		return
	}

	oldRefreshToken := cookie.Value
	oldRefreshHash := hashRefreshToken(oldRefreshToken)

	session, err := s.sessions.Get(r.Context(), oldRefreshHash)
	if errors.Is(err, ErrSessionNotFound) {
		_ = writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"invalid refresh token",
		)
		return
	}

	if err != nil {
		_ = writeError(
			w,
			http.StatusInternalServerError,
			"internal error",
			"failed to get session",
		)
		return
	}

	user := User{
		ID: session.UserID,
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		_ = writeError(
			w,
			http.StatusInternalServerError,
			"internal error",
			"failed to generate access token",
		)
		return
	}

	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		_ = writeError(
			w,
			http.StatusInternalServerError,
			"internal error",
			"failed to generate refresh token",
		)
		return
	}

	newRefreshHash := hashRefreshToken(newRefreshToken)

	newSession := Session{
		UserID: session.UserID,
	}

	if err := s.sessions.Create(
		r.Context(),
		newRefreshHash,
		newSession,
		refreshesTTL,
	); err != nil {
		_ = writeError(
			w,
			http.StatusInternalServerError,
			"internal error",
			"failed to create new session",
		)
		return
	}

	if err := s.sessions.Delete(
		r.Context(),
		oldRefreshHash,
	); err != nil {
		_ = writeError(
			w,
			http.StatusInternalServerError,
			"internal error",
			"failed to rotate session",
		)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(refreshesTTL.Seconds()),
		Secure:   false,
	})

	response := loginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		fmt.Println(err)
	}
}
