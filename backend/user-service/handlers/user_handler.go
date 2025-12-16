package handlers

import (
	"time"
	"fmt"

	"twinup/user-service/database"
	"twinup/user-service/models"

	"github.com/gofiber/fiber/v2" // Fiber kütüphanesi
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("benim_cok_gizli_anahtarim")

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 1. GET USERS (Fiber)
// 1. GET USERS (DÜZELTİLMİŞ VERSİYON) ✅
func GetUsers(c *fiber.Ctx) error {
	// Veritabanından 3 sütun istiyoruz
	rows, err := database.DB.Query("SELECT id, username, email FROM users")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	users := make([]models.User, 0)

	for rows.Next() {
		var u models.User
		// DÜZELTME BURADA: Scan içine sadece çektiğimiz 3 veriyi yazdık.
		// Password'ü sildik. Artık 3'e 3 eşleşiyor!
		if err := rows.Scan(&u.ID, &u.Username, &u.Email); err != nil {
			fmt.Println("Okuma hatası:", err) // Hata varsa terminale yazsın
			continue
		}
		users = append(users, u)
	}

	return c.JSON(users)
}

// 2. REGISTER (Fiber)
func RegisterUser(c *fiber.Ctx) error {
	var newUser models.User

	// BodyParser ile gelen JSON'ı değişkene atıyoruz
	if err := c.BodyParser(&newUser); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Veri okunamadı"})
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)

	sqlStatement := `INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`
	_, err := database.DB.Exec(sqlStatement, newUser.Username, newUser.Email, string(hashedPassword))

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Veritabanı hatası: " + err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{"message": "Kayıt Başarılı! (Fiber)", "user": newUser.Username})
}

// 3. LOGIN (Fiber)
func Login(c *fiber.Ctx) error {
	var loginRequest models.User
	if err := c.BodyParser(&loginRequest); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Veri okunamadı"})
	}

	var storedPassword string
	var storedID int

	sqlStatement := `SELECT id, password FROM users WHERE username=$1`
	err := database.DB.QueryRow(sqlStatement, loginRequest.Username).Scan(&storedID, &storedPassword)

	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Kullanıcı bulunamadı"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(loginRequest.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Şifre hatalı"})
	}

	// Token oluşturma (Aynı mantık)
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		Username: loginRequest.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Token hatası"})
	}

	return c.JSON(fiber.Map{"token": tokenString})
}

// 4. WELCOME (VIP) - Fiber
func Welcome(c *fiber.Ctx) error {
	// Header'dan veriyi al: c.Get("HeaderName")
	tokenString := c.Get("Authorization")

	if tokenString == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Bilet yok!"})
	}

	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !tkn.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "Geçersiz Bilet"})
	}

	return c.SendString(fmt.Sprintf("VIP Alanına Hoş Geldin %s! 🥂 (Fiber ile sunuldu)", claims.Username))
}