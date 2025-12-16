package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	
	"twinup/user-service/database"
	"twinup/user-service/handlers"
	"twinup/user-service/middleware" // <-- Bunu eklemeyi unutma!
)

func main() {
	database.Connect()
	// database.CreateTables() // Tablolar zaten var, her seferinde çalıştırmana gerek yok artık.

	app := fiber.New()
	app.Use(logger.New())

	// 1. HERKESE AÇIK ROTALAR (Public)
	app.Post("/register", handlers.RegisterUser)
	app.Post("/login", handlers.Login)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Twinup Backend Çalışıyor! 🚀")
	})

	// 2. KORUMALI ROTALAR (Private / VIP) 🔒
	// '/api' grubunu oluşturuyoruz ve kapısına 'Protected' bekçisini dikiyoruz.
	api := app.Group("/api", middleware.Protected())

	// Artık kullanıcı listesi '/api/user' adresinde ve KİLİTLİ!
	api.Get("/user", handlers.GetUsers) 
	api.Get("/welcome", handlers.Welcome)

	fmt.Println("Fiber Sunucu 8080 portunda çalışıyor... ⚡")
	app.Listen(":8080")
}