package bot

import "encoding/json"

// Update sekarang bisa berisi Message atau CallbackQuery
type Update struct {
	ID            int           `json:"update_id"`
	Message       *Message      `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	ID                  int                 `json:"message_id"`
	From                User                `json:"from"`
	Chat                Chat                `json:"chat"`
	Date                int                 `json:"date"`
	Text                string              `json:"text"`
	Caption             string              `json:"caption,omitempty"` // <-- TAMBAHKAN INI
	Photo               []*PhotoSize        `json:"photo,omitempty"` // Tambahkan ini
	Sticker             *Sticker            `json:"sticker,omitempty"`
	Document            *Document           `json:"document,omitempty"`
	Animation           *Animation          `json:"animation,omitempty"`
	Audio               *Audio              `json:"audio,omitempty"`
	DirectMessagesTopic DirectMessagesTopic `json:"direct_messages_topic,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
	LangCode  string `json:"language_code"`
}

type Chat struct {
	ID               int64  `json:"id"`
	Type             string `json:"type"`
	Title            string `json:"title,omitempty"`
	Username         string `json:"username,omitempty"`
	FirstName        string `json:"first_name,omitempty"`
	IsDirectMessages bool   `json:"is_direct_messages,omitempty"`
}

type DirectMessagesTopic struct {
	TopicID int  `json:"topic_id"`
	User    User `json:"user,omitempty"`
}

type ApiResponse struct {
	Ok     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

type ChatMember struct {
	User   User   `json:"user"`
	Status string `json:"status"`
}

type GetChatResponse struct {
	ID         int64       `json:"id"`
	Title      string      `json:"title"` // Tambahkan Title
	ParentChat *ParentChat `json:"parent_chat,omitempty"`
}

type ParentChat struct {
	ID int64 `json:"id"`
}

// ----- STRUKTUR BARU UNTUK INTERAKSI TOMBOL -----

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type SendMessagePayload struct {
	ChatID                int64                 `json:"chat_id"`
	Text                  string                `json:"text"`
	DirectMessagesTopicID int                   `json:"direct_messages_topic_id,omitempty"`
	ParseMode             string                `json:"parse_mode,omitempty"`
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// Struct baru untuk mengedit pesan yang sudah ada
type EditMessageTextPayload struct {
	ChatID      int64                 `json:"chat_id"`
	MessageID   int                   `json:"message_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type AnswerCallbackQueryPayload struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}

type Sticker struct {
	FileID string `json:"file_id"`
}

type Document struct {
	FileID string `json:"file_id"`
}

type Animation struct { // Untuk GIF
	FileID string `json:"file_id"`
}

type Audio struct {
	FileID string `json:"file_id"`
}

type SendStickerPayload struct {
	ChatID                int64  `json:"chat_id"`
	Sticker               string `json:"sticker"`
	DirectMessagesTopicID int    `json:"direct_messages_topic_id,omitempty"`
}

type SendDocumentPayload struct {
	ChatID                int64  `json:"chat_id"`
	Document              string `json:"document"`
	Caption               string `json:"caption,omitempty"`
	DirectMessagesTopicID int    `json:"direct_messages_topic_id,omitempty"`
}

type SendAnimationPayload struct {
	ChatID                int64  `json:"chat_id"`
	Animation             string `json:"animation"`
	Caption               string `json:"caption,omitempty"`
	DirectMessagesTopicID int    `json:"direct_messages_topic_id,omitempty"`
}

type SendAudioPayload struct {
	ChatID                int64  `json:"chat_id"`
	Audio                 string `json:"audio"`
	Caption               string `json:"caption,omitempty"`
	DirectMessagesTopicID int    `json:"direct_messages_topic_id,omitempty"`
}

type PhotoSize struct {
	FileID   string `json:"file_id"`
	FileSize int    `json:"file_size"`
}

type SendPhotoPayload struct {
	ChatID                int64  `json:"chat_id"`
	Photo                 string `json:"photo"`
	Caption               string `json:"caption,omitempty"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DirectMessagesTopicID int    `json:"direct_messages_topic_id,omitempty"`
}