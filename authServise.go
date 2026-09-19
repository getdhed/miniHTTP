package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionStore struct {
	client *redis.Client
}

func NewSessionStore(client *redis.Client) *SessionStore {
	return &SessionStore{
		client: client,
	}
}

func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(bytes), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *SessionStore) Create(ctx context.Context, refreshHash string, session Session, ttl time.Duration) error {
	key := sessionKey(refreshHash)
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return err
	}
	return nil

}

func (s *SessionStore) Rotate(ctx context.Context, oldRefreshHash string, newRefreshHash string, session Session) error {
	var rotateSessionScript = redis.NewScript(`
		local oldKey = KEYS[1]
		local newKey = KEYS[2]
		local newValue = ARGV[1]

		if redis.call("EXISTS", oldKey) == 0 then
			return 0
		end

		local ttl = redis.call("PTTL", oldKey)

		if ttl <= 0 then
			return 0
		end

		redis.call("SET", newKey, newValue, "PX", ttl)
		redis.call("DEL", oldKey)

		return 1
	`)
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	oldKey := sessionKey(oldRefreshHash)
	newKey := sessionKey(newRefreshHash)

	result, err := rotateSessionScript.Run(ctx, s.client, []string{oldKey, newKey}, data).Int()
	if err != nil {
		return fmt.Errorf("rotate session script: %w", err)
	}

	if result == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func sessionKey(refreshHash string) string {
	return "session:" + refreshHash
}
func (s *SessionStore) Get(ctx context.Context, refreshHash string) (Session, error) {
	key := sessionKey(refreshHash)

	data, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, ErrSessionNotFound) {
		return Session{}, ErrSessionNotFound
	}

	if err != nil {
		return Session{}, err
	}
	var session Session

	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *SessionStore) Delete(ctx context.Context, refreshHash string) error {
	key := sessionKey(refreshHash)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}

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

func (a *AuthService) refresh(ctx context.Context, oldRefreshToken string) (RefreshResponse, error) {

	oldRefreshHash := hashRefreshToken(oldRefreshToken)

	session, err := a.sessions.Get(ctx, oldRefreshHash)
	if errors.Is(err, ErrSessionNotFound) {
		return RefreshResponse{}, ErrSessionNotFound
	}
	if err != nil {
		return RefreshResponse{}, fmt.Errorf("get session: %w", err)
	}

	user := User{
		ID: session.UserID,
	}

	accessToken, err := a.generateAccessToken(user)
	if err != nil {
		return RefreshResponse{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		return RefreshResponse{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	newRefreshHash := hashRefreshToken(newRefreshToken)

	newSession := Session{
		UserID: session.UserID,
	}

	if err := a.sessions.Rotate(ctx, oldRefreshHash, newRefreshHash, newSession); err != nil {
		return RefreshResponse{}, err
	}
	response := RefreshResponse{
		RefreshToken: newRefreshToken,
		AccessToken:  accessToken,
	}
	return response, nil

}

func (a *AuthService) logout(ctx context.Context, refreshToken string) error {
	hashRefresh := hashRefreshToken(refreshToken)
	if err := a.sessions.Delete(ctx, hashRefresh); err != nil {
		return fmt.Errorf("err with deleting refresh hash %w", err)
	}
	return nil
}
