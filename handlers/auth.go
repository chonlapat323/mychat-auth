package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"mychat-auth/middleware"
	"mychat-auth/models"
	"mychat-auth/types"
	"mychat-auth/utils"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var userCollection *mongo.Collection
var validate = validator.New()

func InitMongo() {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(os.Getenv("MONGO_URL")))
	if err != nil {
		panic(err)
	}
	userCollection = client.Database("mychat").Collection("users")
}

// RegisterHandler รับ POST /register
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// validate email/password
	if err := validate.Struct(req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// check duplicate email
	count, _ := userCollection.CountDocuments(context.TODO(), bson.M{"email": req.Email})
	if count > 0 {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	// hash password
	hashedPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Hash error", http.StatusInternalServerError)
		return
	}
	req.Password = hashedPwd
	req.CreatedAt = time.Now()

	// insert user
	res, err := userCollection.InsertOne(context.TODO(), req)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	// ดึง ObjectID กลับมา แล้วแปลงเป็น string
	userID := res.InsertedID.(primitive.ObjectID).Hex()

	// generate token
	token, err := utils.GenerateToken(userID, req.Email)

	if err != nil {
		http.Error(w, "Token error", http.StatusInternalServerError)
		return
	}

	// return token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// ค้นหาผู้ใช้จาก email
	var user models.User
	err := userCollection.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// เช็ค password
	if !utils.CheckPassword(req.Password, user.Password) {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// สร้าง JWT
	token, err := utils.GenerateToken(user.ID.Hex(), user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user models.User
	err := userCollection.FindOne(context.TODO(), bson.M{"_id": models.StringToObjectID(userID)}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user.Password = "" // ซ่อน password

	safeUser := types.SafeUser{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	json.NewEncoder(w).Encode(safeUser)
}
