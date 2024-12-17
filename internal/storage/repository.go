package storage

import (
	"errors"
	"fmt"
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

	log.Printf("%v new highlights found \n", len(highlightsToInsert))

	if len(highlightsToInsert) > 0 {
		repo.insertHighlight(highlightsToInsert)
	}
}

type title struct {
	Id   int
	Name string
}

func (repo Repository) GetSources() []title {

	titles := []title{}

	var sources []Source

	repo.db.Find(&sources)

	for _, source := range sources {
		titles = append(titles, title{
			Id:   int(source.ID),
			Name: source.Title,
		})
	}

	return titles
}

func (repo Repository) GetSource(id int) Source {
	var source Source

	repo.db.Where("id = ?", id).First(&source)

	return source
}

func (repo Repository) GetNotes(source_id int) []Note {
	var notes []Note

	repo.db.Joins("JOIN sources ON notes.source_id = sources.id").Where(
		"sources.id = ?", source_id).Find(&notes)

	return notes
}

func (repo Repository) GetNextNote(source_id int, skip int) (Note, error) {

	var notes []Note

	result := repo.db.
		Where("source_id = ?", source_id).
		Preload("Source").
		Order("id ASC").
		Offset(skip).
		Limit(1).
		Find(&notes)

	if result.Error != nil || result.RowsAffected == 0 {
		return Note{}, result.Error
	}

	return notes[0], nil
}

func (repo Repository) GetNote(id int) (*Note, error) {

	var note Note

	result := repo.db.Limit(1).
		Preload("Source").
		Where("id = ?", id).
		Find(&note)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get note: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("note with id %d not found", id)
	}

	return &note, nil
}

func (repo Repository) UpdateNote(id int, update Note) {

	var note Note

	repo.db.Debug().Model(&note).Where("id = ?", id).Updates(update)
}
