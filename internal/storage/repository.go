package storage

import (
	"errors"
	"log"

	"github.com/amalrajan30/spacedgram/internal/highlights"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	db.AutoMigrate(&Note{}, &Source{})

	return &Repository{
		db: db,
	}
}

func insertToSource(db *gorm.DB, data *Source) (sourceId int, err error) {
	result := db.Create(data)

	if result.Error == nil {
		log.Printf("New Source inserted: %v \n", result.RowsAffected)
		return int(data.ID), nil
	}

	return 0, result.Error
}

func (repo Repository) ensureSourceExists(itm highlights.Highlight) (id int, err error) {
	var source Source

	result := repo.db.Where("title = ?", itm.Title).First(&source)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("No record for %v found, inserting.. \n", itm.Title)
			newSource, err := insertToSource(repo.db, &Source{
				Title:      itm.Title,
				Origin:     "kindle",
				TotalNotes: 0,
			})

			return newSource, err
		} else {
			log.Printf("Failed to query Source for: %v with err: %v", itm.Title, result.Error)
			return 0, result.Error
		}
	}

	return int(source.ID), nil

}

func (repo Repository) insertHighlight(items []highlights.Highlight) {
	var notes []Note

	for _, itm := range items {
		sourceId, err := repo.ensureSourceExists(itm)
		if err != nil {
			log.Printf("Failed to get source for: %v from %v, skipping insert...\n", itm.Location, itm.Title)

			continue
		}
		notes = append(notes, Note{
			Content:  itm.Content,
			Location: itm.Location,
			SourceID: sourceId,
		})
	}

	result := repo.db.Create(&notes)

	if result.Error != nil {
		log.Printf("Failed to insert highlights: %v \n", result.Error)
	}
}

func (repo Repository) BulkInsertHighlights(toInsert []highlights.Highlight) {
	// var source Source

	highlightsToInsert := make([]highlights.Highlight, 0)

	for _, highlight := range toInsert {
		var notes Note
		location := highlight.Location
		title := highlight.Title
		result := repo.db.Joins("JOIN sources ON notes.source_id = sources.id").Where("notes.location = ? AND sources.title = ?", location, title).First(&notes)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				log.Printf("No entry found for highlight: %v from %v", location, title)
				highlightsToInsert = append(highlightsToInsert, highlight)
			} else {
				log.Printf("Error while querying: %v", result.Error)
			}
		}
	}

	if len(highlightsToInsert) > 0 {
		repo.insertHighlight(highlightsToInsert)
	}
}

func (repo Repository) GetSources() {

}
