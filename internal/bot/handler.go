package bot

import (
	"fmt"

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
