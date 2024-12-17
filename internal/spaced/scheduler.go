package spaced

import (
	"fmt"
	"time"
)

// Scheduler is responsible for scheduling reviews for notes. It will run every day and review the notes that are due.
type Scheduler struct {
	notes []*Note
}

func (s *Scheduler) Run() {
	for _, note := range s.notes {
		if time.Now().After(note.nextDueDate) {
			// s.reviewNote(note)
		}
	}
}

// func (s *Scheduler) reviewNote(note *Note) {
// 	reviewQuality := s.getReviewQuality()
// 	nextDueDate := GetNextDueDate(note, reviewQuality)

// 	note.nextDueDate = nextDueDate
// 	note.lastReviewed = time.Now()
// }

func (s *Scheduler) getReviewQuality() int {
	fmt.Println("Getting review quality")
	return 5
}

func NewScheduler(notes []*Note) *Scheduler {
	return &Scheduler{notes: notes}
}

