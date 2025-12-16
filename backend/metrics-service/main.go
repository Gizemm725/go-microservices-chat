package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/segmentio/kafka-go"
)

type IngestEvent struct {
	Sender  string `json:"sender"`
	Content string `json:"content"`
	Topic   string `json:"topic"`
}

func envOrDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func mustClickHouse(ctx context.Context, addr, database string) driver.Conn {
	user := envOrDefault("CLICKHOUSE_USER", "default")
	pass := envOrDefault("CLICKHOUSE_PASSWORD", "")

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: pass,
		},
		DialTimeout: 5 * time.Second,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{Method: clickhouse.CompressionLZ4},
	})
	if err != nil {
		log.Fatalf("clickhouse open error: %v", err)
	}

	deadline := time.Now().Add(60 * time.Second)
	for {
		if err := conn.Ping(ctx); err != nil {
			var e *clickhouse.Exception
			if errors.As(err, &e) {
				log.Fatalf("clickhouse ping error [%d] %s\n%s", e.Code, e.Message, e.StackTrace)
			}
			if time.Now().After(deadline) {
				log.Fatalf("clickhouse ping error: %v", err)
			}
			log.Printf("clickhouse not ready yet (%v), retrying...", err)
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	return conn
}

func ensureSchema(ctx context.Context, conn driver.Conn, database, table string) {
	err := conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", database))
	if err != nil {
		log.Fatalf("create database error: %v", err)
	}

	ddl := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.%s (
	inserted_at DateTime DEFAULT now(),
	event_time DateTime,
	topic String,
	sender String,
	content String,
	raw String
) ENGINE = MergeTree
ORDER BY (event_time, topic, sender)
`, database, table)

	if err := conn.Exec(ctx, ddl); err != nil {
		log.Fatalf("create table error: %v", err)
	}
}

func insertWithRetry(ctx context.Context, conn driver.Conn, database, table string, eventTime time.Time, topic, sender, content, raw string) error {
	query := fmt.Sprintf("INSERT INTO %s.%s (event_time, topic, sender, content, raw) VALUES (?, ?, ?, ?, ?)", database, table)
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		lastErr = conn.Exec(ctx, query, eventTime, topic, sender, content, raw)
		if lastErr == nil {
			return nil
		}
		time.Sleep(time.Duration(attempt) * 400 * time.Millisecond)
	}
	return lastErr
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	kafkaBroker := envOrDefault("KAFKA_BROKER", "localhost:9092")
	kafkaTopic := envOrDefault("KAFKA_TOPIC", "message-events")
	kafkaGroup := envOrDefault("KAFKA_GROUP_ID", "metrics-service")

	clickhouseAddr := envOrDefault("CLICKHOUSE_ADDR", "localhost:9000")
	clickhouseDB := envOrDefault("CLICKHOUSE_DB", "default")
	clickhouseTable := envOrDefault("CLICKHOUSE_TABLE", "message_events")

	ch := mustClickHouse(ctx, clickhouseAddr, clickhouseDB)
	ensureSchema(ctx, ch, clickhouseDB, clickhouseTable)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{kafkaBroker},
		Topic:       kafkaTopic,
		GroupID:     kafkaGroup,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     1 * time.Second,
		StartOffset: kafka.LastOffset,
	})
	defer func() {
		_ = reader.Close()
	}()

	log.Printf("metrics-service started | kafka=%s topic=%s group=%s | clickhouse=%s db=%s table=%s", kafkaBroker, kafkaTopic, kafkaGroup, clickhouseAddr, clickhouseDB, clickhouseTable)

	for {
		select {
		case <-ctx.Done():
			log.Printf("shutdown requested")
			return
		default:
		}

		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("kafka read error: %v", err)
			time.Sleep(750 * time.Millisecond)
			continue
		}

		raw := string(msg.Value)
		eventTime := msg.Time
		if eventTime.IsZero() {
			eventTime = time.Now()
		}

		var e IngestEvent
		if err := json.Unmarshal(msg.Value, &e); err != nil {
			e = IngestEvent{}
		}

		topic := e.Topic
		if topic == "" {
			topic = kafkaTopic
		}

		sender := e.Sender
		if sender == "" {
			sender = string(msg.Key)
		}

		content := e.Content

		insCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = insertWithRetry(insCtx, ch, clickhouseDB, clickhouseTable, eventTime, topic, sender, content, raw)
		cancel()
		if err != nil {
			if errors.Is(err, sql.ErrConnDone) {
				log.Printf("clickhouse connection closed: %v", err)
			} else {
				log.Printf("clickhouse insert error: %v", err)
			}
			continue
		}
	}
}