package main

import (
	"fmt"
	"log"
	"mychat-auth/handlers"
	"mychat-auth/middleware"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	// โหลดค่าจาก .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("❌ ไม่พบไฟล์ .env หรือโหลดไม่สำเร็จ")
	}

	// เชื่อม MongoDB
	handlers.InitMongo()

	// สร้าง route
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.Handle("/me", middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.MeHandler)))

	port := ":4001"
	fmt.Println("🚀 Auth service running at http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
