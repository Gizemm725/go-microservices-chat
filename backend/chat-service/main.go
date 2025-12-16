package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go" // <-- YENİ: Kafka Kütüphanesi
)

var mqttClient mqtt.Client
var db *sql.DB
var kafkaWriter *kafka.Writer // <-- YENİ: Kafka Yazarı (Producer)

type Message struct {
	ID        int       `json:"id"`
	Topic     string    `json:"topic"`
	Content   string    `json:"content"`
	Sender    string    `json:"sender"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	// --- 1. KAFKA BAĞLANTISI (YENİ) ---
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" { kafkaBroker = "localhost:9092" } // Local test

	fmt.Printf("Kafka Kamyonu Hazırlanıyor... Hedef: %s 🚛\n", kafkaBroker)
	
	// Kafka'ya yazacak olan "Writer"ı ayarlıyoruz
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    "message-events", // Mesajları bu kanala dökeceğiz
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close() // Program kapanırken kamyonu garaja çek

	// --- 2. VERİTABANI BAĞLANTISI ---
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" { dbHost = "localhost" }
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" { dbPort = "5433" }
	
	connStr := fmt.Sprintf("host=%s port=%s user=twinup_user password=twinup_password dbname=twinup_db sslmode=disable", dbHost, dbPort)
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil { log.Fatal("DB Hatası:", err) }

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id SERIAL PRIMARY KEY,
		topic TEXT NOT NULL,
		sender TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil { log.Fatal("Tablo Hatası:", err) }

	// --- 3. MQTT BAĞLANTISI ---
	brokerAddr := os.Getenv("MQTT_BROKER")
	if brokerAddr == "" { brokerAddr = "tcp://localhost:1883" }
	
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerAddr)
	opts.SetClientID("go_chat_api")
	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("MQTT Bağlantı Hatası:", token.Error())
	} else {
		fmt.Println("✅ MQTT Bağlantısı Başarılı!")
	}

	// --- 4. FIBER API ---
	app := fiber.New()
	app.Use(cors.New())

	app.Post("/send", func(c *fiber.Ctx) error {
		var msg Message
		if err := c.BodyParser(&msg); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Veri okunamadı"})
		}

		// A) PostgreSQL'e Kaydet
		sqlStatement := `INSERT INTO messages (topic, sender, content) VALUES ($1, $2, $3) RETURNING id, created_at`
		err := db.QueryRow(sqlStatement, msg.Topic, msg.Sender, msg.Content).Scan(&msg.ID, &msg.CreatedAt)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "DB Hatası: " + err.Error()})
		}

		// B) MQTT'ye Yayınla (Anlık Sohbet)
		mqttMsg := fmt.Sprintf("%s: %s", msg.Sender, msg.Content)
		token := mqttClient.Publish(msg.Topic, 1, false, mqttMsg)
		token.Wait()

		// C) KAFKA'YA GÖNDER (YENİ! - Analiz İçin) 🚛
		// Bu işlem arka planda (goroutine ile) yapılabilir ama şimdilik düz yapalım.
		go func() {
			err := kafkaWriter.WriteMessages(context.Background(),
				kafka.Message{
					Key:   []byte(msg.Sender), // Anahtar: Gönderen kişi
					Value: []byte(fmt.Sprintf(`{"sender":"%s", "content":"%s", "topic":"%s"}`, msg.Sender, msg.Content, msg.Topic)),
				},
			)
			if err != nil {
				fmt.Println("Kafka'ya yazılamadı:", err)
			} else {
				fmt.Println("Kafka'ya veri atıldı! 📊")
			}
		}()

		return c.JSON(msg)
	})

	// History API (Aynı kalsın)
	app.Get("/history", func(c *fiber.Ctx) error {
		topic := c.Query("topic")
		rows, err := db.Query("SELECT id, sender, content, created_at FROM messages WHERE topic=$1 ORDER BY created_at ASC", topic)
		if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
		defer rows.Close()
		messages := []Message{}
		for rows.Next() {
			var m Message
			m.Topic = topic
			if err := rows.Scan(&m.ID, &m.Sender, &m.Content, &m.CreatedAt); err != nil { continue }
			messages = append(messages, m)
		}
		return c.JSON(messages)
	})

	log.Fatal(app.Listen(":8081"))
}