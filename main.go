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

func main() {
	// โหลดค่าจาก .env
	err := godotenv.Load()
	utils.InitRedis()
	if err != nil {
		log.Fatal("❌ ไม่พบไฟล์ .env หรือโหลดไม่สำเร็จ")
	}

	// เชื่อม MongoDB
	database.InitMongo()

	// create seed
	utils.SeedAdminUser()
	utils.SeedRoom()

	// สร้าง route
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.Handle("/me", middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.MeHandler)))
	http.Handle("/logout", middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.LogoutHandler)))

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

	port := ":4001"
	fmt.Println("🚀 Auth service running at http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
