package storage

import (
	"log"
	"time"

	"gorm.io/gorm"
)

type Note struct {
	gorm.Model
	Content        string
	EasinessFactor *float64
	NextDueDate    *time.Time `gorm:"index"`
	LastReviewed   *time.Time
	Interval       int
	ReviewCount    int
	Location       string `gorm:"index"`
	SourceID       int
	Source         Source
}

type Source struct {
	gorm.Model
	Title      string `gorm:"index"`
	Origin     string
	TotalNotes int
}

func (n *Note) AfterCreate(tx *gorm.DB) (err error) {

	log.Printf("Updating total notes of source %v", n.SourceID)

	tx.Exec("UPDATE sources SET total_notes = total_notes + 1 WHERE id = ?", n.SourceID)

	return
}
