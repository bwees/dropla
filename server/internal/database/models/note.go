package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Note struct {
	ID    string `gorm:"primaryKey" json:"id" validate:"required"`
	Title string `gorm:"not null" json:"title" validate:"required"`
	Body  string `gorm:"not null" json:"body" validate:"required"`
}

func (n *Note) BeforeCreate(*gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	return nil
}

func (Note) TableName() string {
	return "note"
}
