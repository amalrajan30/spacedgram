package bot

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"gorm.io/gorm"
)

type BotHandler struct {
	service *BotService

	rwMux    sync.RWMutex
	userData map[string]int
}

func (handler *BotHandler) setUserData(key string, val int) {
	handler.rwMux.Lock()
	defer handler.rwMux.Unlock()

	if handler.userData == nil {
		handler.userData = map[string]int{}
	}

	log.Printf("Setting user Data: %v, %v \n", key, val)

	handler.userData[key] = val
}

func (handler *BotHandler) getUserData(key string) (int, bool) {
	handler.rwMux.RLock()
	defer handler.rwMux.RUnlock()

	val, err := handler.userData[key]

	return val, err
}

func NewBotHandler(service *BotService) *BotHandler {
	return &BotHandler{
		service: service,
	}
}

func (handler *BotHandler) ListTopics(b *gotgbot.Bot, ctx *ext.Context) error {

	fmt.Println("Got list all topic")

	sources := handler.service.repo.GetSources()

	msg := ""

	for i, source := range sources {
		msg = msg + fmt.Sprintf("%v) %v \n", i+1, source)
	}

	_, err := ctx.EffectiveMessage.Reply(b, msg, &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})

	if err != nil {
		return fmt.Errorf("Failed to sent list message: %w", err)
	}

	return nil
}

func (handler *BotHandler) SyncNotes(b *gotgbot.Bot, ctx *ext.Context) error {

	handler.service.SyncHighlights()

	_, err := ctx.EffectiveMessage.Reply(b, "All notes synced", &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})

	if err != nil {
		return fmt.Errorf("Failed to sent list message: %w", err)
	}

	return nil
}

func (handler *BotHandler) StartReviewing(b *gotgbot.Bot, ctx *ext.Context) error {

	sources := handler.service.repo.GetSources()

	var keyboard [][]gotgbot.InlineKeyboardButton
	var currentRow []gotgbot.InlineKeyboardButton

	for i, source := range sources {
		callbackData := strconv.Itoa(source.Id)
		currentRow = append(currentRow, gotgbot.InlineKeyboardButton{
			Text:         strings.Split(source.Name, ":")[0],
			CallbackData: callbackData,
		})

		if (i+1)%3 == 0 || i == len(sources)-1 {
			keyboard = append(keyboard, currentRow)
			currentRow = []gotgbot.InlineKeyboardButton{}
		}
	}

	inlineKeyword := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}

	_, err := ctx.EffectiveMessage.Reply(b, "Please choose a Title:", &gotgbot.SendMessageOpts{
		ReplyMarkup: inlineKeyword,
		ParseMode:   "HTML",
	})

	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil

}

func (handler *BotHandler) HandleCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	bookID := ctx.CallbackQuery.Data
	cb := ctx.Update.CallbackQuery

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Processing...",
	})

	if err != nil {
		return fmt.Errorf("Failed to answer callback query: %v", err)
	}

	if bookID == "" {
		_, _, err = cb.Message.EditText(b, "Got invalid response", nil)
	}

	id, err := strconv.Atoi(bookID)

	if err != nil {
		_, _, err = cb.Message.EditText(b, "Got invalid response", nil)
	}

	source := handler.service.repo.GetSource(id)
	handler.setUserData("source_id", id)

	keyboard := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{
			{
				Text:         "Start Review",
				CallbackData: "start_review",
			},
			{
				Text:         "Reset",
				CallbackData: "reset",
			},
		}},
	}

	_, _, err = cb.Message.EditText(b, fmt.Sprintf(
		"Selected book: %s\nNo of notes: %v",
		source.Title, source.TotalNotes,
	), &gotgbot.EditMessageTextOpts{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})

	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}

func (handler *BotHandler) HandleStartReview(b *gotgbot.Bot, ctx *ext.Context) error {
	cb := ctx.Update.CallbackQuery

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Starting Review....",
	})

	if err != nil {
		return fmt.Errorf("Failed to answer callback query: %v", err)
	}

	id, not_found := handler.getUserData("source_id")

	if not_found {
		_, _, err = cb.Message.EditText(b, "Got invalid response", nil)
	}

	log.Printf("Got book to start review: %v\n", id)

	source := handler.service.repo.GetSource(id)

	_, _, err = cb.Message.EditText(b, fmt.Sprintf(
		"Starting review for book: %s\nNo of notes: %v",
		source.Title, source.TotalNotes,
	), &gotgbot.EditMessageTextOpts{
		ParseMode: "HTML",
	})

	notesToReview := handler.service.repo.GetNotes(int(source.ID))

	handler.setUserData("notes_count", len(notesToReview))
	handler.setUserData("skip", 0)

	handler.HandleReviews(b, ctx)

	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}

func (handler *BotHandler) HandleReviews(b *gotgbot.Bot, ctx *ext.Context) error {
	cb := ctx.Update.CallbackQuery
	data := ctx.CallbackQuery.Data

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Getting Note",
	})

	if err != nil {
		return fmt.Errorf("Failed to answer callback query: %v", err)
	}

	if data != "start_review" {
		handler.service.HandleReviewResponse(data)
	}

	source_id, not_found := handler.getUserData("source_id")
	skip, not_found := handler.getUserData("skip")
	notes_count, not_found := handler.getUserData("notes_count")

	if not_found {
		_, _, err := cb.Message.EditText(b, "Got invalid response", nil)

		if err != nil {
			return fmt.Errorf("failed to answer callback query: %v", err)
		}
	}

	if skip >= notes_count {
		log.Printf("Review completed for: %v", source_id)
		_, _, msgErr := cb.Message.EditText(b, "Review complete",
			&gotgbot.EditMessageTextOpts{
				ParseMode: "HTML",
			})
		if msgErr != nil {
			return fmt.Errorf("failed to edit message: %w", err)
		}
		return nil
	}

	noteToReview, err := handler.service.repo.GetNextNote(source_id, skip)

	if err != nil {
		msg := "Something went wrong"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("no more notes found for source %d", source_id)
		} else {
			log.Printf("Something went wrong while getting next note for review: %v", err)
		}
		_, _, msgErr := cb.Message.EditText(b, msg,
			&gotgbot.EditMessageTextOpts{
				ParseMode: "HTML",
			})
		if msgErr != nil {
			return fmt.Errorf("failed to edit message: %w", err)
		}
		return nil
	}

	log.Printf("Got note to review: %v", noteToReview)

	keyboard := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{
				Text:         "Prefect",
				CallbackData: fmt.Sprintf("review_%v_%v", noteToReview.ID, 5),
			}},
			{{
				Text:         "Some Hesitation",
				CallbackData: fmt.Sprintf("review_%v_%v", noteToReview.ID, 4),
			}},
			{{
				Text:         "With Difficulty",
				CallbackData: fmt.Sprintf("review_%v_%v", noteToReview.ID, 3),
			}},
			{{
				Text:         "Wrong, Recalled",
				CallbackData: fmt.Sprintf("review_%v_%v", noteToReview.ID, 2),
			}},
			{{
				Text:         "Wrong, Remembered when shown",
				CallbackData: fmt.Sprintf("review_%v_%v", noteToReview.ID, 1),
			}},
			{{
				Text:         "Complete Blackout",
				CallbackData: fmt.Sprintf("review_%v_%v", noteToReview.ID, 0),
			}},
		},
	}

	_, _, err = cb.Message.EditText(b, fmt.Sprintf(
		"<b>%s</b> \n\n<i>%v</i>\n",
		noteToReview.Content, noteToReview.Source.Title,
	), &gotgbot.EditMessageTextOpts{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})

	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	handler.setUserData("skip", skip+1)

	return nil
}
