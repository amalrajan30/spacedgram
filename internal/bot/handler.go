package bot

import (
	"errors"
	"fmt"
	"log"
	"os"
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
	notes    []int
}

func checkUser(from int64) bool {
	userId, err := strconv.Atoi(os.Getenv("USER_ID"))

	if err != nil {
		return false
	}

	if from != int64(userId) {
		return false
	}

	return true

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

func (h *BotHandler) editMessage(b *gotgbot.Bot, msg gotgbot.MaybeInaccessibleMessage, text string, opts *gotgbot.EditMessageTextOpts) error {
	if opts == nil {
		opts = &gotgbot.EditMessageTextOpts{
			ParseMode: "HTML",
		}
	}
	_, _, err := msg.EditText(b, text, opts)
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}
	return nil
}

type reviewButton struct {
	Text  string
	Score int
}

var reviewButtons = []reviewButton{
	{Text: "Perfect", Score: 5},
	{Text: "Some Hesitation", Score: 4},
	{Text: "With Difficulty", Score: 3},
	{Text: "Wrong, Recalled", Score: 2},
	{Text: "Wrong, Remembered when shown", Score: 1},
	{Text: "Complete Blackout", Score: 0},
}

func (h *BotHandler) buildReviewKeyboard(noteID int64) gotgbot.InlineKeyboardMarkup {
	var keyboardRows [][]gotgbot.InlineKeyboardButton

	for _, button := range reviewButtons {
		keyboardRows = append(keyboardRows, []gotgbot.InlineKeyboardButton{{
			Text:         button.Text,
			CallbackData: fmt.Sprintf("review_%v_%v", noteID, button.Score),
		}})
	}

	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: keyboardRows,
	}
}

func (handler *BotHandler) ListTopics(b *gotgbot.Bot, ctx *ext.Context) error {

	if !checkUser(ctx.Message.From.Id) {
		return nil
	}

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

	if !checkUser(ctx.Message.From.Id) {
		return nil
	}

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

	if !checkUser(ctx.Message.From.Id) {
		return nil
	}

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

func (h *BotHandler) HandleSelectSourceCallback(b *gotgbot.Bot, ctx *ext.Context) error {

	if !checkUser(ctx.Update.CallbackQuery.From.Id) {
		return nil
	}

	bookID := ctx.CallbackQuery.Data
	cb := ctx.Update.CallbackQuery

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Processing...",
	})

	if err != nil {
		return fmt.Errorf("failed to answer callback query: %w", err)
	}

	source, err := h.service.SelectSource(bookID)

	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return h.editMessage(b, cb.Message, "Could not find the specified book", nil)
		default:
			log.Printf("Failed to start review: %v", err)
			return h.editMessage(b, cb.Message, "Failed to start review", nil)
		}
	}

	h.setUserData("source_id", int(source.ID))

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

	if err := h.editMessage(b, cb.Message, fmt.Sprintf(
		"📚 <b>Selected Book</b>\n"+
			"━━━━━━━━━━━━━━\n"+
			"<b>Title:</b> %s\n"+
			"<b>Notes:</b> %v",
		source.Title,
		source.TotalNotes,
	), &gotgbot.EditMessageTextOpts{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	}); err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}

func (h *BotHandler) StartReview(b *gotgbot.Bot, ctx *ext.Context) error {
	if !checkUser(ctx.Update.CallbackQuery.From.Id) {
		return nil
	}

	cb := ctx.Update.CallbackQuery

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Starting Review....",
	})

	if err != nil {
		return fmt.Errorf("failed to answer callback query: %w", err)
	}

	id, _ := h.getUserData("source_id")

	status, err := h.service.ClozeStatus(id)

	if status {
		ctx.CallbackQuery.Data = "cloze_yes"
		h.ClozeQuestion(b, ctx)
	}

	keyboard := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{
			{
				Text:         "Yes",
				CallbackData: "cloze_yes",
			},
			{
				Text:         "No",
				CallbackData: "cloze_no",
			},
		}},
	}

	if err := h.editMessage(b, cb.Message, "Want to use cloze style questions?", &gotgbot.EditMessageTextOpts{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	}); err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil

}

func (h *BotHandler) ClozeQuestion(b *gotgbot.Bot, ctx *ext.Context) error {

	if !checkUser(ctx.Update.CallbackQuery.From.Id) {
		return nil
	}

	cb := ctx.Update.CallbackQuery
	data := ctx.CallbackQuery.Data

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Starting Review....",
	})

	if err != nil {
		return fmt.Errorf("failed to answer callback query: %w", err)
	}

	if strings.Split(data, "_")[1] == "yes" {
		h.setUserData("clozeQuestion", 1)
	} else {
		h.setUserData("clozeQuestion", 0)
	}

	id, not_found := h.getUserData("source_id")

	if not_found {
		_, _, _ = cb.Message.EditText(b, "Got invalid response", nil)
	}

	log.Printf("Got book to start review: %v\n", id)

	session, err := h.service.StartSourceReview(id)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return h.editMessage(b, cb.Message, "Could not find the specified book", nil)
		default:
			log.Printf("Failed to start review: %v", err)
			return h.editMessage(b, cb.Message, "Failed to start review", nil)
		}
	}

	if err := h.editMessage(b, cb.Message, fmt.Sprintf(
		"📚 <b>Starting Review</b>\n"+
			"━━━━━━━━━━━━━━\n"+
			"<b>Book:</b> %s\n"+
			"<b>Notes:</b> %v",
		session.Source.Title,
		session.Source.TotalNotes,
	), nil); err != nil {
		return fmt.Errorf("editing message: %w", err)
	}

	h.setUserData("notes_count", session.Count)
	h.setUserData("skip", 0)
	h.notes = session.NoteIDs

	h.HandleReviews(b, ctx)

	return nil
}

func (h *BotHandler) HandleReviews(b *gotgbot.Bot, ctx *ext.Context) error {

	if !checkUser(ctx.Update.CallbackQuery.From.Id) {
		return nil
	}

	cb := ctx.Update.CallbackQuery
	data := ctx.CallbackQuery.Data

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Getting Note",
	})

	if err != nil {
		return fmt.Errorf("Failed to answer callback query: %v", err)
	}

	skip, skipNotFound := h.getUserData("skip")
	// notes_count, not_found := h.getUserData("notes_count")

	if !skipNotFound {
		log.Printf("Not found hit: %v", skip)
		return h.editMessage(b, cb.Message, "Got invalid response", nil)
	}

	fmt.Printf("Notes : %v \n", h.notes)

	clazeEnabled, _ := h.getUserData("clozeQuestion")

	fmt.Printf("Claze enabled: %v \n", clazeEnabled)

	state, err := h.service.ProcessReview(h.notes, skip, data, clazeEnabled == 1)

	if err != nil {
		log.Printf("Error processing review: %v", err)
		return h.editMessage(b, cb.Message, "Something went wrong while processing review", nil)
	}

	if state.IsComplete {
		log.Printf("Review completed")
		return h.editMessage(b, cb.Message, "Review complete", nil)
	}

	// Delete the previous message to avoid spoiler reveal issues
	_, err = cb.Message.Delete(b, nil)
	if err != nil {
		log.Printf("Error deleting previous message: %v", err)
	}

	var noteText string

	if clazeEnabled == 0 {
		noteText = fmt.Sprintf(
			"📝 <b>Note #%v/%v</b>\n\n"+
				"%s\n\n"+
				"📚 <i>From:</i> %v",
			skip+1,
			len(h.notes),
			state.NoteToReview.Content,
			state.NoteToReview.Source.Title,
		)
	} else {
		noteText = fmt.Sprintf(
			"📝 <b>Note #%v/%v</b>\n\n"+
				"Question: %s\n\n"+
				"Answer: <span class=\"tg-spoiler\">%s</span>\n\n"+
				"📚 <i>From:</i> %v",
			skip+1,
			len(h.notes),
			state.NoteToReview.Question,
			state.NoteToReview.Answer,
			state.NoteToReview.Source.Title,
		)
	}

	keyboard := h.buildReviewKeyboard(int64(state.NoteToReview.ID))

	if _, keyboardErr := b.SendMessage(cb.Message.GetChat().Id, noteText, &gotgbot.SendMessageOpts{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	}); keyboardErr != nil {
		log.Printf("Error while sending review keyboard: %v", keyboardErr)
		return fmt.Errorf("editing message with note: %w", keyboardErr)
	}

	h.setUserData("skip", skip+1)

	return nil
}

func (handler *BotHandler) HandleReviewReset(b *gotgbot.Bot, ctx *ext.Context) error {

	if !checkUser(ctx.Update.CallbackQuery.From.Id) {
		return nil
	}

	cb := ctx.Update.CallbackQuery

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Processing",
	})

	if err != nil {
		return fmt.Errorf("Failed to answer callback query: %v", err)
	}

	source_id, not_found := handler.getUserData("source_id")
	if not_found {
		_, _, err := cb.Message.EditText(b, "Got invalid response", nil)

		if err != nil {
			return fmt.Errorf("failed to answer callback query: %v", err)
		}
	}

	handler.service.HandleReset(source_id)

	_, _, msgErr := cb.Message.EditText(b, "Review progress rested",
		&gotgbot.EditMessageTextOpts{
			ParseMode: "HTML",
		})
	if msgErr != nil {
		return fmt.Errorf("failed to edit message: %w", msgErr)
	}
	return nil
}

func (h *BotHandler) StartReviewScheduled(b *gotgbot.Bot, ctx *ext.Context) error {

	if !checkUser(ctx.Update.CallbackQuery.From.Id) {
		return nil
	}

	cb := ctx.Update.CallbackQuery

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Processing",
	})

	if err != nil {
		return fmt.Errorf("failed to answer callback query: %w", err)
	}

	session, err := h.service.ScheduledReview()

	if err != nil {
		log.Printf("Failed to start review: %v", err)
		return h.editMessage(b, cb.Message, "Failed to start review", nil)
	}

	if session.Count == 0 {
		log.Printf("No notes to review today")
		return h.editMessage(b, cb.Message, "No Notes left to review today!", nil)
	}

	if err := h.editMessage(b, cb.Message, fmt.Sprintf(
		"Total review for today: %v",
		session.Count,
	), nil); err != nil {
		return fmt.Errorf("editing message: %w", err)
	}

	h.setUserData("notes_count", session.Count)
	h.setUserData("skip", 0)
	h.notes = session.NoteIDs

	h.HandleReviews(b, ctx)
	return nil
}
