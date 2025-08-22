package model

import "time"

type Role struct {
	ID          string `gorm:"default:(-)"`
	Name        string
	Description string
	Scopes      []Scope `gorm:"many2many:role_scopes;"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
