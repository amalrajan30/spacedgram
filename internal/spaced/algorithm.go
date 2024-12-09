package spaced

import (
	"math/rand"
	"time"
)

type Note struct {
	nextDueDate time.Time
	lastReviewed time.Time
	easinessFactor float64
	interval int
	reviewCount int
}

func calculateEasinessFactor(reviewQuality int) float64 {
	 // Formula: EF = 0.1 - (5-q)(0.08 + (5-q)(0.02))
    
    // Calculate (5-q) once since it's used twice
    fiveMinusQ := 5 - reviewQuality
    
    // Calculate the inner parentheses: (5-q)(0.02)
    innerTerm := float64(fiveMinusQ) * 0.02
    
    // Add 0.08 to get: (0.08 + (5-q)(0.02))
    middleTerm := 0.08 + innerTerm
    
    // Multiply by (5-q) again
    outerTerm := float64(fiveMinusQ) * middleTerm
    
    // Final calculation: 0.1 - (previous result)
    ef := 0.1 - outerTerm

	return ef
}

func addRandomVariance(interval int) int {
	return interval + rand.Intn(3) - 1
}


func GetNextDueDate(note *Note, reviewQuality int) time.Time {
	if note.reviewCount == 0 {
		note.nextDueDate = time.Now()
		note.interval = 0
		return note.nextDueDate
	}

	if note.reviewCount == 1 {
		note.nextDueDate = time.Now().AddDate(0, 0, 1)
		note.interval = 1
		return note.nextDueDate
	}

	if note.reviewCount == 2 {
		note.nextDueDate = time.Now().AddDate(0, 0, 6)
		note.interval = 6
		return note.nextDueDate
	}

	easinessFactor := calculateEasinessFactor(reviewQuality)

	note.easinessFactor = easinessFactor

	note.interval = addRandomVariance(int(float64(note.interval) * easinessFactor))

	note.nextDueDate = note.lastReviewed.AddDate(0, 0, note.interval)

	return note.nextDueDate

}

