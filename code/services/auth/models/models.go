package models

type AuthUser struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	Email    string `gorm:"uniqueIndex"`
	Password string
}
