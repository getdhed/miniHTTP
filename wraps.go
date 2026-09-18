package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func writeError(w http.ResponseWriter, status int, code string, message string) error {

	resp := errorResponse{
		Error:   code,
		Message: message,
	}

	return writeJSON(w, status, resp)
}
func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(rw.status)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	n, err := rw.ResponseWriter.Write(data)

	rw.bytes += n

	return n, err
}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	resp, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(resp); err != nil {
		return err
	}

	return nil
}
func (a *AuthService) generateAccessToken(user User) (string, error) {
	now := time.Now()

	claims := jwt.RegisteredClaims{
		Subject:   strconv.Itoa(user.ID),
		Issuer:    "miniHTTP",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.jwtConfig.Secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil

}

func (s *Server) parseAccessToken(tokenString string) (int, error) {
	claims := &jwt.RegisteredClaims{}
	if _, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			return s.jwtConfig.Secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer("miniHTTP"),
		jwt.WithExpirationRequired()); err != nil {
		return 0, err
	}
	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	parts := strings.Fields(authHeader)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid authorization header")
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("invalid authorization header")
	}
	return parts[1], nil
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func (a *AuthService) Login(ctx context.Context, email string, password string) (LoginResult, error) {

	if !confirmPassword(password) || !confirmEmail(email) {
		return LoginResult{}, ErrInvalidCredentials
	}

	user, err := a.users.findByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("find user by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	accessToken, err := a.generateAccessToken(user)
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshHash := hashRefreshToken(refreshToken)

	session := Session{
		UserID: user.ID,
	}

	if err := a.sessions.Create(
		ctx,
		refreshHash,
		session,
		refreshesTTL); err != nil {
		return LoginResult{}, fmt.Errorf("create session: %w", err)
	}
	return LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
	}, nil
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(refreshesTTL.Seconds()),
		Secure:   false,
	})
}
