# GoTele-Master: Telegram DM Auto-Reply Bot

![Go Version](https://img.shields.io/badge/Go-1.18%2B-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)

A powerful and scalable Telegram bot built with pure Go (no external Telegram libraries) to automatically reply to messages in a channel's Direct Messages. This bot is multi-channel ready, managed by channel admins, and uses Supabase for persistent data storage.

## ✨ Features

- **Interactive Setup**: Easy-to-use interactive menus for teaching (`/learn`), managing (`/manage`), and configuration.
- **Multi-Channel Support**: A single bot instance can serve countless channels, with each having its own separate knowledge base.
- **Admin-Managed**: Only channel administrators can register a channel and manage its triggers and responses.
- **Persistent Storage**: Uses [Supabase](https://supabase.com) (PostgreSQL) to ensure no data is lost on restart.
- **Dynamic Replies**: Supports placeholders (e.g., `{{user_first_name}}`) and Markdown formatting in replies.
- **Concurrency Ready**: Built with Goroutines to handle many users simultaneously without lag.
- **Multi-Language Support**: All bot-facing text can be easily translated via JSON files (`locales/`).

## 🚀 Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) (version 1.18 or higher)
- A [Telegram Bot Token](https://core.telegram.org/bots#6-botfather)
- A free [Supabase](https://supabase.com) account

### 1. Setup Supabase

1.  Create a new project in Supabase.
2.  Go to the **SQL Editor** and run the SQL scripts found in the `schema.sql` file to create the necessary tables (`channels`, `triggers`, `users`).
3.  Go to **Project Settings > API** to get your **Project URL** and `public` `anon` **Key**.

### 2. Installation

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/your-username/telegram-dm-bot.git](https://github.com/your-username/telegram-dm-bot.git)
    cd telegram-dm-bot
    ```

2.  **Install dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Configure environment variables:**
    -   Copy the example `.env.example` file to a new file named `.env`.
        ```bash
        cp .env.example .env
        ```
    -   Fill in the `.env` file with your credentials:
        ```dotenv
        TELEGRAM_BOT_TOKEN="YOUR_TELEGRAM_BOT_TOKEN"
        SUPABASE_URL="YOUR_SUPABASE_URL"
        SUPABASE_KEY="YOUR_SUPABASE_KEY"
        ```

4.  **Run the bot:**
    ```bash
    go run main.go
    ```

## 🤖 How to Use

1.  **Add the Bot to Your Channel**: Add your bot as an **Administrator** to your public or private Telegram channel.
2.  **Register the Channel**: Start a private (1-on-1) chat with your bot and use the `/register` command.
    ```
    /register @your_channel_username
    ```
3.  **Teach the Bot**: Use the interactive `/learn` command in the private chat to teach the bot new triggers and replies.
4.  **Manage Triggers**: Use the `/manage` command to view, paginate, and delete existing triggers for a channel.

The bot will now automatically reply to users in your channel's Direct Messages!