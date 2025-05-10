package middleware

import (
	"context"
	"net/http"
	"strings"

	"mychat-auth/utils" // แก้ตาม module ของคุณ
)

type contextKey string

const UserIDKey contextKey = "user_id"

func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing or invalid token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// ✅ ตรวจว่าถูก blacklist หรือไม่
		isBlacklisted, err := utils.IsTokenBlacklisted(tokenString)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		if isBlacklisted {
			http.Error(w, "Token revoked", http.StatusUnauthorized)
			return
		}

		// ✅ ตรวจ token ตามปกติ
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
