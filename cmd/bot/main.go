package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/amalrajan30/spacedgram/internal/bot"
	"github.com/joho/godotenv"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

func main() {
	envErr := godotenv.Load()

	if envErr != nil {
		log.Fatal("loading env file failed")
	}


	fmt.Println("Hello, World!")
	token := os.Getenv("BOT_TOKEN")

	if token == "" {
		panic("BOT_TOKEN environment variable is empty")
	}

	b, err := gotgbot.NewBot(token, nil)

	if err != nil {
		panic("Failed to create new bot: " + err.Error())
	}

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Println("Error occurred while handling update:", err.Error())
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})

	updater := ext.NewUpdater(dispatcher, nil)

	dispatcher.AddHandler(handlers.NewCommand("list_topics", bot.ListTopics))

	err = updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
		},
	})

	if err != nil {
		panic("failed to start polling: " + err.Error())
	}

	log.Printf("%s has been started...\n", b.User.Username)

	// Idle, to keep updates coming in, and avoid bot stopping.
	updater.Idle()
}