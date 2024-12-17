package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/amalrajan30/spacedgram/internal/bot"
	"github.com/amalrajan30/spacedgram/internal/storage"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
)

func main() {
	envErr := godotenv.Load()

	if envErr != nil {
		log.Fatalf("loading env file failed: %v", envErr)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := gorm.Open(postgres.Open(dsn))

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
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

	repository := storage.NewRepository(db)
	botService := bot.NewBotService(repository)

	botHandler := bot.NewBotHandler(botService)

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Println("Error occurred while handling update:", err.Error())
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})

	updater := ext.NewUpdater(dispatcher, nil)

	dispatcher.AddHandler(handlers.NewCommand("list_topics", botHandler.ListTopics))
	dispatcher.AddHandler(handlers.NewCommand("sync", botHandler.SyncNotes))
	dispatcher.AddHandler(handlers.NewCommand("startreview", botHandler.StartReviewing))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("reset"), botHandler.HandleReviewReset))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("start_review"), botHandler.HandleStartReview))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("review"), botHandler.HandleReviews))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.All, botHandler.HandleSelectSourceCallback))

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
	defer updater.Idle()

	// defer highlights.UploadHandler()
}
