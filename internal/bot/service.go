package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"

	"github.com/amalrajan30/spacedgram/internal/highlights"
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