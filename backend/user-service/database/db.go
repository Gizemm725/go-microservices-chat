package database

import (
	"database/sql"
	"fmt"
	"os"
	_ "github.com/lib/pq" // Sürücü burada gerekli
)

// DB değişkenini büyük harfle başlattık ki diğer dosyalardan erişebilelim (Exported)
var DB *sql.DB
func Connect() {
	// Ortam değişkenlerini oku, yoksa varsayılanı kullan
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost" // Bilgisayarında çalıştırırken burası
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5433" // Bilgisayarında çalıştırırken burası
	}

	// Docker içindeyken bu değişkenleri docker-compose'dan vereceğiz!
	connStr := fmt.Sprintf("host=%s port=%s user=twinup_user password=twinup_password dbname=twinup_db sslmode=disable", host, port)
	
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	err = DB.Ping()
	if err != nil {
		panic(err)
	}
   CreateTables()
   fmt.Println("tablo oluşturldu")
	fmt.Println("Veritabanı bağlantısı sağlandı! 🔌 (" + host + ":" + port + ")")
}

func CreateTables() {
    // Eski tablo silme kodunu kapattık (Verilerimiz artık silinmeyecek)
    // _, err := DB.Exec(`DROP TABLE IF EXISTS users`)
    // if err != nil {
    //     panic(err)
    // }

    // DÜZELTME BURADA YAPILDI:
    // err değişkenini ilk kez burada kullandığımız için ':=' kullandık.
    _, err := DB.Exec(`CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        username TEXT NOT NULL,
        email TEXT NOT NULL,
        password TEXT NOT NULL
    )`)
    
    if err != nil {
        panic(err)
    }
    fmt.Println("Tablolar kontrol edildi (Veriler korundu). 📋")
}