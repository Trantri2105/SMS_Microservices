package model

import "time"

type User struct {
	ID        string `gorm:"default:(-)"`
	Email     string
	Password  string
	FirstName string
	LastName  string
	Roles     []Role `gorm:"many2many:roles_users;"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
