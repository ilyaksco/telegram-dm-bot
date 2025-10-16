package main

import (
	"log"

	"telegram-dm-bot/bot"
	"telegram-dm-bot/config"
	"telegram-dm-bot/i18n"
	"telegram-dm-bot/storage"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("could not load configuration: %v", err)
	}

	if err := i18n.LoadTranslations("locales"); err != nil {
		log.Fatalf("could not load translations: %v", err)
	}

	store, err := storage.NewSupabaseStorage(cfg.SupabaseURL, cfg.SupabaseKey)
	if err != nil {
		log.Fatalf("could not initialize supabase storage: %v", err)
	}

	telegramBot := bot.NewBot(cfg, store)

	telegramBot.Start()
}