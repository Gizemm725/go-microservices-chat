package models

// User struct'ı artık models paketi altında
// Baş harfi büyük (User) olduğu için diğer dosyalardan erişilebilir.
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}