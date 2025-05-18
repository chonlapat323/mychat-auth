package main

import (
	"fmt"
	"log"
	"mychat-auth/database"
	"mychat-auth/handlers"
	"mychat-auth/middleware"
	"mychat-auth/utils"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		log.Println("🔥 CORS Middleware:", r.URL.Path)
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// โหลดค่าจาก .env
	if os.Getenv("APP_ENV") != "production" {
		err := godotenv.Load()
		utils.InitRedis()
		if err != nil {
			log.Fatal("ไม่พบไฟล์ .env หรือโหลดไม่สำเร็จ")
		}
	}

	// เชื่อม MongoDB
	database.InitMongo()

	// สร้าง route เฉพาะที่เกี่ยวกับ Auth และ User Management
	http.Handle("/register", corsMiddleware(http.HandlerFunc(handlers.RegisterHandler)))
	http.Handle("/login", corsMiddleware(http.HandlerFunc(handlers.LoginHandler)))
	http.Handle("/me", corsMiddleware(middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.MeHandler))))
	http.Handle("/logout", corsMiddleware(middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.LogoutHandler))))
	http.Handle("/auth/refresh", corsMiddleware(http.HandlerFunc(handlers.RefreshHandler)))
	http.Handle("/api/users", corsMiddleware(middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.UsersHandler))))

	port := ":4001"
	fmt.Println("Auth service running at http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
