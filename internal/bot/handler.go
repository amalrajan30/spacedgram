package bot

import (
	"fmt"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func ListTopics(b *gotgbot.Bot, ctx *ext.Context) error {

	fmt.Println("Got list all topic")

	_, err := ctx.EffectiveMessage.Reply(b, "Hallo, this is in work!!!", &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})

	if err != nil {
		return fmt.Errorf("Failed to sent list message: %w", err)
	}

	return nil
}