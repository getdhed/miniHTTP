package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
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
