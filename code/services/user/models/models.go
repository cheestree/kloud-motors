package models

type User struct {
	ID       string `gorm:"primaryKey;type:uuid"`
	Name     string
	Email    string `gorm:"uniqueIndex"`
	Password string
}
