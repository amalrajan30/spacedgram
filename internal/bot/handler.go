package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

type BotHandler struct {
	service *BotService
}

func NewBotHandler(service *BotService) *BotHandler {
	return &BotHandler{
		service: service,
	}
}

func (handler BotHandler) ListTopics(b *gotgbot.Bot, ctx *ext.Context) error {

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

func (handler BotHandler) SyncNotes(b *gotgbot.Bot, ctx *ext.Context) error {

	handler.service.SyncHighlights()

	_, err := ctx.EffectiveMessage.Reply(b, "All notes synced", &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})

	if err != nil {
		return fmt.Errorf("Failed to sent list message: %w", err)
	}

	return nil
}

func (handler BotHandler) StartReviewing(b *gotgbot.Bot, ctx *ext.Context) error {

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

func (handler BotHandler) HandleCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	bookID := ctx.CallbackQuery.Data
	cb := ctx.Update.CallbackQuery

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Processing...",
	})

	if err != nil {
		return fmt.Errorf("Failed to answer callback query: %v", err)
	}

	id, err := strconv.Atoi(bookID)

	if err != nil {
		_, _, err = cb.Message.EditText(b, "Got invalid response", nil)
	}

	source := handler.service.repo.GetSource(id)

	_, _, err = cb.Message.EditText(b, fmt.Sprintf(
		"Selected book: %s\nNo of notes: %v",
		source.Title, source.TotalNotes,
	), &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})

	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}
