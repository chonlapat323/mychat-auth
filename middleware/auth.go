package middleware

import (
	"context"
	"log"
	"mychat-auth/shared/contextkey"
	"mychat-auth/utils"
	"net/http"
)

type contextKey string

func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("🔐 JWTAuthMiddleware called")

		cookie, err := r.Cookie("token")
		if err != nil || cookie.Value == "" {
			log.Println("❌ Token not found in cookie:", err)
			http.Error(w, "Missing or invalid token", http.StatusUnauthorized)
			return
		}

		tokenString := cookie.Value
		log.Println("🧪 Token received:", tokenString[:10], "...")

		// isBlacklisted, err := utils.IsTokenBlacklisted(tokenString)
		// if err != nil {
		// 	log.Println("❌ Redis check failed:", err)
		// 	http.Error(w, "Server error", http.StatusInternalServerError)
		// 	return
		// }
		// if isBlacklisted {
		// 	log.Println("🚫 Token is blacklisted")
		// 	http.Error(w, "Token revoked", http.StatusUnauthorized)
		// 	return
		// }

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			log.Println("❌ Token validation failed:", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		log.Println("✅ Token valid. User ID:", claims.UserID)

		ctx := context.WithValue(r.Context(), contextkey.UserID, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
