package highlights

import "time"

type Highlight struct {
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Location  string    `json:"location"`
	DateAdded time.Time `json:"dateAdded"`
	Content   string    `json:"content"`
}