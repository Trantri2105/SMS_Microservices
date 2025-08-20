package model

import "time"

type User struct {
	ID        string `gorm:"default:(-)"`
	Email     string
	Password  string
	FirstName string
	LastName  string
	Role      []Role `gorm:"many2many:role_users;"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
