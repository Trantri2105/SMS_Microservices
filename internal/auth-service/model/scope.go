package model

import "time"

type Scope struct {
	ID          string `gorm:"default:(-)"`
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
