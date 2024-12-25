package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amalrajan30/spacedgram/internal/highlights"
	"github.com/amalrajan30/spacedgram/internal/llm"
	"github.com/amalrajan30/spacedgram/internal/spaced"
	"github.com/amalrajan30/spacedgram/internal/storage"
)

type BotService struct {
	repo *storage.Repository
}

func NewBotService(repo *storage.Repository) *BotService {
	return &BotService{
		repo: repo,
	}
}

// GetLatestFile returns the most recent file from a list of files that match the pattern
// upload_YYYYMMDDHHMMSS_highlights.json
func getLatestFile(files []fs.DirEntry) (fs.DirEntry, error) {
	type FileInfo struct {
		file      fs.DirEntry
		Timestamp time.Time
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	var filesInfo []FileInfo

	// Parse each filename to extract the timestamp
	for _, file := range files {
		// Get just the filename without path
		filename := file.Name()

		// Extract timestamp part (assumes format upload_YYYYMMDDHHMMSS_highlights.json)
		parts := strings.Split(filename, "_")
		if len(parts) != 3 {
			continue // Skip files that don't match expected format
		}

		// Parse timestamp
		timestamp, err := time.ParseInLocation("20060102150405", parts[1], time.Local)
		if err != nil {
			continue // Skip files with invalid timestamps
		}

		filesInfo = append(filesInfo, FileInfo{
			file:      file,
			Timestamp: timestamp,
		})
	}

	if len(filesInfo) == 0 {
		return nil, fmt.Errorf("no valid files found matching the expected pattern")
	}

	// Find the latest file
	latest := filesInfo[0]
	for _, file := range filesInfo[1:] {
		if file.Timestamp.After(latest.Timestamp) {
			latest = file
		}
	}

	return latest.file, nil
}

func getUploadFile() []highlights.Highlight {
	highlightsFile, err := os.ReadDir("uploads")

	if err != nil {
		log.Println("Failed to read uploads folder")
		return nil
	}

	latestHighlight, err := getLatestFile(highlightsFile)

	if err != nil {
		log.Printf("Failed to find latest upload: %v \n", err)
		return nil
	}

	jsonFile, err := os.Open(fmt.Sprintf("uploads/%v", latestHighlight.Name()))

	defer jsonFile.Close()

	byteVal, _ := io.ReadAll(jsonFile)

	if err != nil {
		log.Printf("Failed to open the file '%v': %v \n", latestHighlight.Name(), err)
		return nil
	}

	var highlights []highlights.Highlight

	json.Unmarshal(byteVal, &highlights)

	return highlights

}

func (service BotService) SyncHighlights() {
	items := getUploadFile()

	if items == nil {
		log.Println("No highlights found to sync")
		return
	}

	service.repo.BulkInsertHighlights(items)

}

func (s BotService) SelectSource(callbackData string) (storage.Source, error) {
	if callbackData == "" {
		return storage.Source{}, fmt.Errorf("No callback data found")
	}

	id, err := strconv.Atoi(callbackData)

	if err != nil {
		return storage.Source{}, fmt.Errorf("failed to convert callback id to int")
	}

	source, err := s.repo.GetSource(id)

	if err != nil {
		return storage.Source{}, err
	}

	return source, nil
}

type ReviewSession struct {
	Source  storage.Source
	Count   int
	NoteIDs []int
}

func (s BotService) StartSourceReview(sourceID int) (*ReviewSession, error) {
	source, err := s.repo.GetSource(sourceID)
	noteIDs := []int{}

	if err != nil {
		return nil, fmt.Errorf("getting source: %w", err)
	}

	notes, err := s.repo.GetNotes(int(source.ID))

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve notes for source ID %v", sourceID)
	}

	for _, note := range notes {
		noteIDs = append(noteIDs, int(note.ID))
	}

	return &ReviewSession{
		Source:  source,
		Count:   len(notes),
		NoteIDs: noteIDs,
	}, nil

}

type ReviewState struct {
	NoteToReview *storage.Note
	IsComplete   bool
	CurrentCount int
	TotalCount   int
}

func (s BotService) ProcessReview(notes []int, skip int, previousResponse string, clazeQuestion bool) (*ReviewState, error) {

	fmt.Printf("Process review data: %v", previousResponse)

	if !strings.HasPrefix(previousResponse, "cloze") && previousResponse != "start_review_schedule" {
		if err := s.HandleReviewResponse(previousResponse); err != nil {
			return nil, fmt.Errorf("handling review response: %w", err)
		}
	}

	if skip >= len(notes) {
		return &ReviewState{
			IsComplete:   true,
			CurrentCount: skip,
			TotalCount:   skip,
		}, nil
	}

	noteID := notes[skip]

	var note *storage.Note

	nextNote, err := s.repo.GetNextNote(noteID)
	note = &nextNote
	if err != nil {
		return nil, fmt.Errorf("getting next note: %w", err)
	}

	if clazeQuestion {
		clazeNote, err := s.GetClozeQuestion(note)

		if err != nil {
			return nil, fmt.Errorf("getting claze question: %w", err)
		}

		note = clazeNote
	}

	return &ReviewState{
		NoteToReview: note,
		IsComplete:   false,
		CurrentCount: skip,
		TotalCount:   skip + 1,
	}, nil
}

func (service BotService) HandleReviewResponse(callbackData string) error {
	noteId, err := strconv.Atoi(strings.Split(callbackData, "_")[1])
	rating, err := strconv.Atoi(strings.Split(callbackData, "_")[2])

	if err != nil {
		fmt.Errorf("Failed to parse data while handling review response: %w", err)
	}

	log.Printf("Got note: %v from review response with rating: %v", noteId, rating)

	note, err := service.repo.GetNote(noteId)

	if err != nil {
		return fmt.Errorf("Failed to get note: %w", err)

	}

	nextDue, interval, easiness := spaced.GetNextDueDate(*note, rating)

	now := time.Now()

	service.repo.UpdateNote(noteId, storage.Note{
		NextDueDate:    &nextDue,
		Interval:       interval,
		EasinessFactor: &easiness,
		LastReviewed:   &now,
		ReviewCount:    note.ReviewCount + 1,
	})

	log.Println("Note updated")

	return nil
}

func (service BotService) HandleReset(source int) {

	service.repo.ResetSource(source)
}

type ScheduledReviews struct {
	Count   int
	NoteIDs []int
}

func (s BotService) ScheduledReview() (*ScheduledReviews, error) {
	notes, err := s.repo.GetPendingReviewNotes()
	noteIDs := []int{}

	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)

	}

	for _, note := range notes {
		noteIDs = append(noteIDs, int(note.ID))
	}

	return &ScheduledReviews{
		Count:   len(notes),
		NoteIDs: noteIDs,
	}, nil
}

func (s BotService) ClozeStatus(sourceID int) (bool, error) {

	source, err := s.repo.GetSource(sourceID)

	if err != nil {
		return false, fmt.Errorf("failed to get source: %w", err)
	}

	return source.ClozeQuestion, nil

}

func (s BotService) GetClozeQuestion(note *storage.Note) (*storage.Note, error) {

	if note.Question != "" {
		return note, nil
	}

	clazeQuestion, err := llm.GenerateClaseQuestionAnswer(note.Content)

	if err != nil {
		return nil, err
	}

	updatedNote, err := s.repo.UpdateNote(int(note.ID), storage.Note{
		Question: clazeQuestion.Question,
		Answer:   clazeQuestion.Answer,
	})

	if err != nil {
		return nil, err
	}

	return updatedNote, nil

}
