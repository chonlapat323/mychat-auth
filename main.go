package main

import (
	"fmt"
	"log"
	"mychat-auth/database"
	"mychat-auth/handlers"
	"mychat-auth/middleware"
	"mychat-auth/utils"
	"net/http"
	"strings"

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

		// สำหรับ preflight request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// โหลดค่าจาก .env
	err := godotenv.Load()
	utils.InitRedis()
	if err != nil {
		log.Fatal("ไม่พบไฟล์ .env หรือโหลดไม่สำเร็จ")
	}

	// เชื่อม MongoDB
	database.InitMongo()

	// create seed
	utils.SeedAdminUser()
	utils.SeedRoom()

	// สร้าง route
	http.Handle("/register", corsMiddleware(http.HandlerFunc(handlers.RegisterHandler)))
	http.Handle("/login", corsMiddleware(http.HandlerFunc(handlers.LoginHandler)))
	http.Handle("/me", corsMiddleware(middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.MeHandler))))
	http.Handle("/logout", corsMiddleware(middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.LogoutHandler))))
	http.Handle("/auth/refresh", corsMiddleware(http.HandlerFunc(handlers.RefreshHandler)))
	http.Handle("/api/users", corsMiddleware(middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.UsersHandler))))
	http.Handle("/rooms", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetRoomsHandler(w, r)
		case http.MethodPost:
			// ใช้ middleware ตรวจ role admin
			middleware.RequireAdmin(http.HandlerFunc(handlers.CreateRoomHandler)).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	http.Handle("/rooms/", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		log.Println("📡 Routed:", path)

		if strings.HasSuffix(path, "/messages") && r.Method == http.MethodGet {
			middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.GetRoomMessagesHandler)).ServeHTTP(w, r)
			return
		}

		if strings.HasSuffix(path, "/join") && r.Method == http.MethodPost {
			middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.JoinRoomHandler)).ServeHTTP(w, r)
			return
		}

		http.Error(w, "Not Found", http.StatusNotFound)
	})))

	http.Handle("/ws", corsMiddleware(http.HandlerFunc(handlers.WebSocketHandler)))

	port := ":4001"
	fmt.Println("Auth service running at http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
