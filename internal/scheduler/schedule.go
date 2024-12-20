package scheduler

import (
	"log"
	"os"
	"strconv"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type Scheduler struct {
	// service     *bot.BotService
	botInstance gotgbot.Bot
}

func NewScheduler(bot gotgbot.Bot) *Scheduler {
	return &Scheduler{
		botInstance: bot,
	}
}

func (s Scheduler) RunScheduled() {

	log.Println("Running scheduled")
	userID, userIDErr := strconv.Atoi(os.Getenv("USER_ID"))

	if userIDErr != nil {
		log.Printf("failed to load user id: %v", userIDErr)
		return
	}

	keyboard := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{
			{
				Text:         "Start Review",
				CallbackData: "start_review_schedule",
			},
		}},
	}

	_, err := s.botInstance.SendMessage(int64(userID), "Ready to start todays review?", &gotgbot.SendMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: keyboard,
	})

	if err != nil {
		log.Printf("failed to send scheduled remainder: %v", err)
	}
}
