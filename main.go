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

		if origin != "" {
			if origin == "http://localhost:3000" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
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

	http.HandleFunc("/rooms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/join/") {
			middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.JoinRoomHandler)).ServeHTTP(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handlers.GetRoomsHandler(w, r)
		case http.MethodPost:
			middleware.RequireAdmin(handlers.CreateRoomHandler)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/ws", handlers.WebSocketHandler)

	port := ":4001"
	fmt.Println("Auth service running at http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
