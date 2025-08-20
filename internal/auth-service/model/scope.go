package model

type Scope struct {
	ID          string `gorm:"default:(-)"`
	Name        string
	Description string
}
