package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken    string
	SupabaseURL string
	SupabaseKey string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment variables")
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("environment variable TELEGRAM_BOT_TOKEN is required")
	}

	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		log.Fatal("environment variable SUPABASE_URL is required")
	}

	supabaseKey := os.Getenv("SUPABASE_KEY")
	if supabaseKey == "" {
		log.Fatal("environment variable SUPABASE_KEY is required")
	}

	return &Config{
		BotToken:    botToken,
		SupabaseURL: supabaseURL,
		SupabaseKey: supabaseKey,
	}, nil
}