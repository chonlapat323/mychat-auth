package middleware

import (
	"context"
	"mychat-auth/shared/contextkey"
	"mychat-auth/utils"
	"net/http"
	"strings"
)

// RequireAdmin เป็น middleware ที่ตรวจว่า token มี role เป็น admin หรือไม่
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if claims.Role != "admin" {
			http.Error(w, "Forbidden: admin only", http.StatusForbidden)
			return
		}

		// ส่ง user_id และ role เข้า context เพื่อให้ handler ปลายทางใช้ได้
		ctx := context.WithValue(r.Context(), contextkey.UserID, claims.UserID)
		ctx = context.WithValue(ctx, contextkey.UserID, claims.Role)
		next(w, r.WithContext(ctx))
	}
}
