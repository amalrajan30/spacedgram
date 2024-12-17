package spaced

import (
	"math/rand"
	"time"

	"github.com/amalrajan30/spacedgram/internal/storage"
)

type Note struct {
	nextDueDate    time.Time
	lastReviewed   time.Time
	easinessFactor float64
	interval       int
	reviewCount    int
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

func GetNextDueDate(note storage.Note, reviewQuality int) (nextDate time.Time, interval int, easinessFactor float64) {
	if note.ReviewCount == 0 {
		return time.Now(), 0, 0
	}

	if note.ReviewCount == 1 {
		return time.Now().AddDate(0, 0, 1), 1, 0
	}

	if note.ReviewCount == 2 {
		return time.Now().AddDate(0, 0, 6), 6, 0
	}

	easinessFactor = calculateEasinessFactor(reviewQuality)

	interval = addRandomVariance(int(float64(note.Interval) * easinessFactor))

	nextDueDate := note.LastReviewed.AddDate(0, 0, interval)

	return nextDueDate, interval, easinessFactor

}
