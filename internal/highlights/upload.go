package highlights

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type JSONResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    Error   string `json:"error,omitempty"`
}

func fileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if token := r.Header.Get("Authorization"); token == "" || token != "1qaz!QAZ" {
			sendErrorResponse(w, "Unauthorized", http.StatusUnauthorized)
		}

		file, handler, err := r.FormFile("highlights")

		if err != nil {
			log.Printf("Error while parsing highlights upload: %v", err)
			sendErrorResponse(w, "Error retrieving file", http.StatusBadRequest)
			return
		}

		defer file.Close()

		if filepath.Ext(handler.Filename) != ".json" {
			sendErrorResponse(w, "Invalid file type.", http.StatusBadRequest)
			return
		}

		// Create unique filename using timestamp
    timestamp := time.Now().Format("20060102150405")
    filename := fmt.Sprintf("upload_%s_%s", timestamp, handler.Filename)
    uploadPath := filepath.Join("uploads", filename)

    // Ensure uploads directory exists
    if err := os.MkdirAll("uploads", 0755); err != nil {
		log.Printf("Error creating upload directory: %v", err)
        sendErrorResponse(w, "Something went wrong", http.StatusInternalServerError)
        return
    }

	    // Create the file
    dst, err := os.Create(uploadPath)
    if err != nil {
		log.Printf("Error creating file on server: %v", err)
        sendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    // Verify JSON format
    if !isValidJSON(file) {
		log.Printf("Invalid JSON format: %v", err)
        sendErrorResponse(w, "Invalid file", http.StatusBadRequest)
        return
    }

    // Reset file pointer to beginning after JSON validation
    file.Seek(0, 0)

    // Copy the uploaded file to the created file on disk
    if _, err := io.Copy(dst, file); err != nil {
		log.Printf("Error saving file to disk: %v", err)
        sendErrorResponse(w, "Something went wrong", http.StatusInternalServerError)
        return
    }

    // Send success response
    response := JSONResponse{
        Success: true,
        Message: fmt.Sprintf("File successfully uploaded as %s", filename),
    }
    json.NewEncoder(w).Encode(response)

	})
}

func isValidJSON(file io.Reader) bool {
    decoder := json.NewDecoder(file)
    var js json.RawMessage
    return decoder.Decode(&js) == nil
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
    w.WriteHeader(statusCode)
    response := JSONResponse{
        Success: false,
        Error:   message,
    }
    json.NewEncoder(w).Encode(response)
}

func UploadHandler() {
	mux := http.NewServeMux()
	server := http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	mux.Handle("POST /api/highlights/upload", fileHandler())

	log.Println("Starting upload server")
	server.ListenAndServe()
}