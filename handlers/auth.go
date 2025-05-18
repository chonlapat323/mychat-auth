package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"mychat-auth/database"
	"mychat-auth/models"
	"mychat-auth/shared/contextkey"
	"mychat-auth/types"
	"mychat-auth/utils"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var validate = validator.New()

// RegisterHandler รับ POST /register
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("📥 RegisterHandler called")

	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("❌ Failed to decode body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// ✅ Validate input
	if err := validate.Struct(req); err != nil {
		log.Println("❌ Validation error:", err)
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("🔍 Checking duplicate email:", req.Email)
	count, err := database.UserCollection.CountDocuments(context.TODO(), bson.M{"email": req.Email})
	if err != nil {
		log.Println("❌ DB count error:", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		log.Println("⚠️ Email already exists")
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	// ✅ Hash password
	log.Println("🔐 Hashing password...")
	hashedPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Println("❌ Password hash error:", err)
		http.Error(w, "Hash error", http.StatusInternalServerError)
		return
	}
	req.Password = hashedPwd
	req.CreatedAt = time.Now()
	req.Role = "member"

	// ✅ Insert user
	log.Println("📝 Inserting new user into DB:", req.Email)
	res, err := database.UserCollection.InsertOne(context.TODO(), req)
	if err != nil {
		log.Println("❌ Insert DB error:", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	userID := res.InsertedID.(primitive.ObjectID).Hex()
	log.Println("✅ User created with ID:", userID)

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"userID": userID,
	})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	var user models.User
	err := database.UserCollection.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil || !utils.CheckPassword(req.Password, user.Password) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid email or password",
		})
		return
	}

	accessToken, refreshToken, err := utils.GenerateTokens(user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to generate token",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    accessToken,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(1 * time.Minute),
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkey.UserID).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user models.User
	err := database.UserCollection.FindOne(context.TODO(), bson.M{"_id": models.StringToObjectID(userID)}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user.Password = "" // ซ่อน password

	safeUser := types.SafeUser{
		ID:        user.ID,
		Email:     user.Email,
		ImageURL:  user.ImageURL,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}

	json.NewEncoder(w).Encode(safeUser)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "No token to logout", http.StatusBadRequest)
		return
	}

	// แบล็คลิสต์ token ตามเดิม (optional)
	claims, err := utils.ValidateToken(cookie.Value)
	if err == nil {
		_ = utils.BlacklistToken(cookie.Value, claims.ExpiresAt.Time) // ไม่ต้อง panic ถ้า error
	}

	// ✅ ลบ cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

func RefreshHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Missing refresh token", http.StatusUnauthorized)
		return
	}

	claims, err := utils.ValidateToken(cookie.Value)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}
	// Generate access token ใหม่
	accessToken, _, err := utils.GenerateTokens(claims.UserID, claims.Email, claims.Role)
	if err != nil {
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    accessToken,
		HttpOnly: true,
		Path:     "/",
		Expires:  claims.ExpiresAt.Time,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusOK)
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		http.Error(w, "Missing ids parameter", http.StatusBadRequest)
		return
	}

	idList := strings.Split(idsParam, ",")

	// สร้าง filter: {"id": {"$in": [id1, id2, id3]}}
	filter := bson.M{
		"id": bson.M{
			"$in": idList,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := database.UserCollection.Find(ctx, filter)
	if err != nil {
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var users []bson.M
	if err := cursor.All(ctx, &users); err != nil {
		http.Error(w, "Error decoding users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
