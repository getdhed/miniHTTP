package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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

func (s *Server) testHandler(w http.ResponseWriter, r *http.Request) {

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

func (s *Server) createPostHandler(w http.ResponseWriter, r *http.Request) {

	userID, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		fmt.Println("произошла ошибка: ", ok)
		if err := writeError(w, http.StatusBadRequest, "bad request", "bad request"); err != nil {
			fmt.Println("произошла ошибка: ", err)
		}
		return
	}

	post := Post{
		UserID: int64(userID),
	}

	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		fmt.Println("произошла ошибка: ", err)
		if err := writeError(w, http.StatusBadRequest, "bad request", "bad request"); err != nil {
			fmt.Println("произошла ошибка: ", err)
		}
		return
	}

	err := s.posts.CreatePost(r.Context(), post)
	if err != nil {
		fmt.Println("произошла ошибка: ", err)
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
			fmt.Println("произошла ошибка: ", err)
		}
		return
	}

	if err := writeJSON(w, http.StatusCreated, nil); err != nil {
		fmt.Println("произошла ошибка: ", err)
	}

}

func (s *Server) getAllPostsHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := s.posts.FindAll(r.Context())
	if err != nil {
		fmt.Println("проищошла ошибка", err)
		writeError(w, http.StatusInternalServerError, "interanl error", "interanl error")
		return
	}
	if err := writeJSON(w, http.StatusOK, posts); err != nil {
		fmt.Println("проищошла ошибка", err)
	}
}

func (s *Server) getUsersPostsHandler(w http.ResponseWriter, r *http.Request) {
	userIDString := r.PathValue("userID")
	userID, err := strconv.ParseInt(userIDString, 10, 64)
	if err != nil {
		if err := writeError(w, http.StatusBadRequest, "bad request", "bad request"); err != nil {
			fmt.Println("произошла ошибка:", err)
		}
		return
	}
	posts, err := s.posts.FindByUserID(r.Context(), userID)
	if err != nil {
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
			fmt.Println("произошла ошибка:", err)
		}
		return
	}
	if err := writeJSON(w, http.StatusOK, posts); err != nil {
		fmt.Println("произошла ошибка:", err)
	}
}

func (s *Server) getPostHandler(w http.ResponseWriter, r *http.Request) {
	postIDString := r.PathValue("postID")
	postID, err := strconv.ParseInt(postIDString, 10, 64)
	if err != nil {
		if err := writeError(w, http.StatusBadRequest, "bad request", "bad request"); err != nil {
			fmt.Println("произошла ошибка:", err)
		}
		return
	}
	if postID <= 0 {
		if err := writeError(w, http.StatusBadRequest, "bad request", "bad request"); err != nil {
			fmt.Println("произошла ошибка:", err)
		}
		return
	}
	post, err := s.posts.FindByID(r.Context(), postID)
	if err != nil {
		if err := writeError(w, http.StatusNotFound, "not found", "not found"); err != nil {
			fmt.Println("произошла ошибка:", err)
		}
		return
	}
	if err := writeJSON(w, http.StatusOK, post); err != nil {
		fmt.Println("произошла ошибка: ", err)
	}
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

	token, err := s.auth.generateAccessToken(user)
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

	LoginResult, err := s.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err := writeError(w, http.StatusBadRequest, "bad request", "bad request"); err != nil {
			fmt.Println("Произошла ошибка", err)
		}
		return
	}

	response := loginResponse{
		AccessToken: LoginResult.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   LoginResult.ExpiresIn,
	}

	setRefreshCookie(w, LoginResult.RefreshToken)

	if err := writeJSON(w, http.StatusOK, response); err != nil {
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
	fmt.Println("=== ENTER REFRESH HANDLER ===")
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
	refreshResponse, err := s.auth.refresh(r.Context(), cookie.Value)
	if err != nil {
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
			fmt.Println(err)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshResponse.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(refreshesTTL.Seconds()),
		Secure:   false,
	})

	response := loginResponse{
		AccessToken: refreshResponse.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		fmt.Println(err)
	}
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
			fmt.Println(err)
		}
		return
	}
	if err := s.auth.logout(r.Context(), cookie.Value); err != nil {
		if err := writeError(w, http.StatusInternalServerError, "internal error", "internal error"); err != nil {
			fmt.Println(err)
		}
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Secure:   false,
	})
	w.WriteHeader(http.StatusNoContent)
}
