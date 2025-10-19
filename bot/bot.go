package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings" 
	"text/template"

	"telegram-dm-bot/config"
	"telegram-dm-bot/i18n"
	"telegram-dm-bot/storage"
)

type Bot struct {
	api    *API
	store  storage.Storage
	states *StateManager
	cache  *AdminCache 
	botUsername string // <-- Tambahkan field baru untuk menyimpan username
}

func NewBot(cfg *config.Config, store storage.Storage) *Bot {
	api := NewAPI(cfg.BotToken)

	// 1. Declare botInfo by calling api.GetMe()
	botInfo, err := api.GetMe()
	if err != nil {
		// If getMe fails, stop the bot immediately (Fatal error)
		log.Fatalf("FATAL: Could not get bot info (getMe failed): %v", err)
	}
	return &Bot{
		api:    NewAPI(cfg.BotToken),
		store:  store,
		states: NewStateManager(),
		cache:  NewAdminCache(), 
		botUsername: botInfo.Username, // <-- Simpan username di sini
	}
}

func (b *Bot) Start() {
	log.Println("bot is starting...")
	var offset int
	for {
		updates, err := b.api.GetUpdates(offset)
		if err != nil {
			log.Printf("error getting updates: %v", err)
			continue
		}
		for _, update := range updates {
			offset = update.ID + 1
			// Jalankan handleUpdate di goroutine baru.
			// Kata kunci 'go' adalah inti dari perubahan ini.
			go func(u Update) {
				if err := b.handleUpdate(u); err != nil {
					log.Printf("error handling update %d: %v", u.ID, err)
				}
			}(update)
		}
	}
}

// handleUpdate sekarang menjadi router utama
func (b *Bot) handleUpdate(update Update) error {
	if update.CallbackQuery != nil {
		return b.handleCallbackQuery(update.CallbackQuery)
	}
	if update.Message != nil {
		return b.handleMessage(update.Message)
	}
	return nil
}

func (b *Bot) getUserLang(userID int64, defaultLang string) string {
	lang, found, err := b.store.GetUserLanguage(userID)
	if err != nil {
		log.Printf("error getting user language: %v", err)
		return defaultLang // Fallback ke default jika ada error
	}
	if found {
		return lang
	}
	return defaultLang
}

// Menangani semua pesan teks
func (b *Bot) handleMessage(msg *Message) error {
	log.Printf("received message from user %d in chat %d: '%s'", msg.From.ID, msg.Chat.ID, msg.Text)
	userLang := b.getUserLang(msg.From.ID, msg.From.LangCode)

	// Perintah non-sesi
	switch {
	case strings.HasPrefix(msg.Text, "/start"):
		return b.handleStartCommand(msg, userLang)
	case strings.HasPrefix(msg.Text, "/help"):
		return b.sendHelpMenu(msg.Chat.ID, userLang)
	case strings.HasPrefix(msg.Text, "/register"):
		return b.handleRegisterCommand(msg, userLang)
	case strings.HasPrefix(msg.Text, "/manage"): // Tambahkan perintah baru
		return b.handleManageCommand(msg, userLang)
	case strings.HasPrefix(msg.Text, "/lang"):
		return b.handleLangCommand(msg, userLang)
	case strings.HasPrefix(msg.Text, "/cancel"):
		return b.handleCancelCommand(msg, userLang)
	}

	// Cek apakah pengguna sedang dalam sesi interaktif
	state, inSession := b.states.GetState(msg.From.ID)
	if inSession {
		return b.handleSessionMessage(msg, state, userLang)
	}

	// Perintah /learn hanya bisa dimulai jika tidak ada sesi
	if strings.HasPrefix(msg.Text, "/learn") {
		return b.handleLearnCommand(msg, userLang)
	}

	// Terakhir, logika balasan otomatis
	if msg.Chat.IsDirectMessages && msg.DirectMessagesTopic.TopicID != 0 {
		return b.handleAutoReply(msg)
	}

	return nil
}

// ----- FUNGSI-FUNGSI COMMAND BARU -----

func (b *Bot) handleStartCommand(msg *Message, lang string) error {
	text := i18n.GetMessage(lang, "start_message", nil)
	addToChannelURL := fmt.Sprintf("https://t.me/%s?startgroup=start&admin=post_messages", b.botUsername)
	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: i18n.GetMessage(lang, "help_button", nil), CallbackData: "help_main"},
				{Text: i18n.GetMessage(lang, "language_button", nil), CallbackData: "lang_prompt"},
			},
			{ // Baris pertama
				{Text: i18n.GetMessage(lang, "add_to_channel_button", nil), URL: addToChannelURL}, // Tombol URL baru
			},
		},
	}
	return b.api.SendMessage(SendMessagePayload{
		ChatID:      msg.Chat.ID,
		Text:        text,
		ParseMode:   "Markdown",
		ReplyMarkup: &keyboard,
	})
}

// --- AWAL PERUBAHAN ---
// FUNGSI BARU
func (b *Bot) handleManageCommand(msg *Message, lang string) error {
	// Logikanya mirip dengan /learn: ambil channel terdaftar dimana user adalah admin
	registeredChannels, err := b.store.GetRegisteredChannels()
	if err != nil {
		log.Printf("error getting registered channels: %v", err)
		return err
	}

	var userAdminChannels []storage.RegisteredChannel
	for _, channel := range registeredChannels {
		isAdmin, _ := b.isUserAdmin(channel.ChannelID, msg.From.ID)
		if isAdmin {
			userAdminChannels = append(userAdminChannels, channel)
		}
	}

	if len(userAdminChannels) == 0 {
		text := i18n.GetMessage(lang, "learn_no_channels_found", nil)
		return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text, ParseMode: "Markdown"})
	}

	// Buat tombol, tapi dengan callback data yang berbeda
	var keyboard [][]InlineKeyboardButton
	for _, channel := range userAdminChannels {
		button := InlineKeyboardButton{
			Text:         channel.Title,
			CallbackData: fmt.Sprintf("manage_ch_%d_page_1", channel.ChannelID), // Arahkan ke halaman 1
		}
		keyboard = append(keyboard, []InlineKeyboardButton{button})
	}

	text := i18n.GetMessage(lang, "manage_prompt", nil)
	return b.api.SendMessage(SendMessagePayload{
		ChatID:      msg.Chat.ID,
		Text:        text,
		ReplyMarkup: &InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
}
// --- AKHIR PERUBAHAN ---

func (b *Bot) sendHelpMenu(chatID int64, lang string) error {
	text := i18n.GetMessage(lang, "help_main_text", nil)
	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: i18n.GetMessage(lang, "help_register_button", nil), CallbackData: "help_register"},
				{Text: i18n.GetMessage(lang, "help_learn_button", nil), CallbackData: "help_learn"},
			},
			{
				{Text: i18n.GetMessage(lang, "help_manage_button", nil), CallbackData: "help_manage"},
				{Text: i18n.GetMessage(lang, "help_formatting_button", nil), CallbackData: "help_formatting"},
			},
			{
				{Text: i18n.GetMessage(lang, "help_lang_button", nil), CallbackData: "help_lang"},
				{Text: i18n.GetMessage(lang, "help_cancel_button", nil), CallbackData: "help_cancel"},
			},
		},
	}
	return b.api.SendMessage(SendMessagePayload{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "Markdown",
		ReplyMarkup: &keyboard,
	})
}

func (b *Bot) handleLangCommand(msg *Message, lang string) error {
	text := i18n.GetMessage(lang, "lang_prompt", nil)
	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "🇬🇧 English", CallbackData: "lang_en"}},
			{{Text: "🇮🇩 Indonesia", CallbackData: "lang_id"}},
			{{Text: "🇷🇺 Русский", CallbackData: "lang_ru"}},
		},
	}
	return b.api.SendMessage(SendMessagePayload{
		ChatID:      msg.Chat.ID,
		Text:        text,
		ReplyMarkup: &keyboard,
	})
}

// --- AWAL PERUBAHAN ---
// FUNGSI BARU
// --- AWAL PERUBAHAN ---
// FUNGSI LENGKAP YANG DIPERBARUI
func (b *Bot) handleRegisterCommand(msg *Message, lang string) error {
	parts := strings.Split(msg.Text, " ")

	if len(parts) == 1 {
		b.states.SetState(msg.From.ID, &UserState{Step: "awaiting_registration_forward"})
		text := i18n.GetMessage(lang, "register_prompt_forward", nil)
		return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text, ParseMode: "Markdown"})
	}

	if len(parts) == 2 {
		var chatIdentifier interface{}
		arg := parts[1]

		if chatID, err := strconv.ParseInt(arg, 10, 64); err == nil {
			chatIdentifier = chatID
		} else if strings.HasPrefix(arg, "@") {
			chatIdentifier = arg
		} else {
			text := i18n.GetMessage(lang, "register_usage", nil)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}

		channelInfo, err := b.api.GetChat(chatIdentifier)
		if err != nil {
			log.Printf("register failed for %v: could not get chat info: %v", chatIdentifier, err)
			text := i18n.GetMessage(lang, "register_fail_not_found", nil)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}

		isAdmin, err := b.isUserAdmin(channelInfo.ID, msg.From.ID)
		if err != nil {
			log.Printf("error checking admin status for %v: %v", chatIdentifier, err)
			errorText := fmt.Sprintf("❌ Failed to verify admin status. Make sure I am an administrator in '%s' and try again in a few seconds. (API Error: %v)", channelInfo.Title, err)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: errorText})
		}
		if !isAdmin {
			log.Printf("register failed for %v: user %d is not admin.", chatIdentifier, msg.From.ID)
			text := i18n.GetMessage(lang, "register_fail_not_admin", nil)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}

		if err := b.store.RegisterChannel(channelInfo.ID, channelInfo.Title, msg.From.ID); err != nil {
			log.Printf("register failed for %v: could not save to storage: %v", chatIdentifier, err)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: "An internal error occurred."})
		}

		// BEFORE
		// b.cache.Set(msg.From.ID, nil) // Invalidate cache after successful direct registration

		// AFTER
		b.cache.Invalidate(msg.From.ID) // Hapus cache setelah registrasi berhasil

		data := struct{ ChannelTitle string }{ChannelTitle: channelInfo.Title}
		text := i18n.GetMessage(lang, "register_success", data)
		return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text, ParseMode: "Markdown"})
	}

	text := i18n.GetMessage(lang, "register_usage", nil)
	return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
}
// --- AKHIR PERUBAHAN ---

func (b *Bot) handleCancelCommand(msg *Message, lang string) error {
	_, inSession := b.states.GetState(msg.From.ID)
	if inSession {
		b.states.ClearState(msg.From.ID)
		text := i18n.GetMessage(lang, "cancel_message", nil)
		return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
	}

	text := i18n.GetMessage(lang, "cancel_fail", nil)
	return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
}

// Menangani klik tombol
// --- AWAL PERUBAHAN ---
// FUNGSI LENGKAP YANG DIPERBARUI
func (b *Bot) handleCallbackQuery(cb *CallbackQuery) error {
	userID := cb.From.ID
	lang := b.getUserLang(userID, cb.From.LangCode)
	data := cb.Data
	chatID := cb.Message.Chat.ID
	messageID := cb.Message.ID

	log.Printf("received callback from user %d with data: %s", userID, data)

	// Jawab callback-nya dulu agar tidak loading
	if data == "noop" {
		return b.api.AnswerCallbackQuery(AnswerCallbackQueryPayload{CallbackQueryID: cb.ID})
	}

	if strings.HasPrefix(data, "learn_type_") {
		responseType := strings.TrimPrefix(data, "learn_type_")
		state, found := b.states.GetState(userID)
		if !found { // Sesi hilang, batalkan
			b.api.EditMessageText(EditMessageTextPayload{ChatID: chatID, MessageID: messageID, Text: "Session expired."})
			return nil
		}
		
		state.Step = fmt.Sprintf("awaiting_%s", responseType) // misal: "awaiting_sticker"
		state.ResponseType = responseType
		b.states.SetState(userID, state)
		
		promptTextKey := fmt.Sprintf("learn_awaiting_%s", responseType) // misal: "learn_awaiting_sticker"
		promptText := i18n.GetMessage(lang, promptTextKey, nil)
		
		return b.api.EditMessageText(EditMessageTextPayload{
			ChatID: chatID, MessageID: messageID, Text: promptText, ParseMode: "Markdown",
		})
	}

	if strings.HasPrefix(data, "del_prompt_") {
		parts := strings.Split(data, "_")
		triggerID, _ := strconv.ParseInt(parts[2], 10, 64)
		channelID, _ := strconv.ParseInt(parts[4], 10, 64)
		page, _ := strconv.Atoi(parts[6])

		// Ambil info trigger dari database untuk mendapatkan teksnya
		triggerRecord, found, err := b.store.GetTriggerByID(triggerID)
		if err != nil || !found {
			// Handle error jika trigger tidak ditemukan
			return nil
		}

		textData := struct{ Trigger string }{Trigger: triggerRecord.TriggerText}
		text := i18n.GetMessage(lang, "confirm_delete_prompt", textData)
		keyboard := InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: i18n.GetMessage(lang, "confirm_delete_button", nil), CallbackData: fmt.Sprintf("del_confirm_%d_ch_%d_pg_%d", triggerID, channelID, page)},
					{Text: i18n.GetMessage(lang, "cancel_delete_button", nil), CallbackData: fmt.Sprintf("manage_ch_%d_page_%d", channelID, page)},
				},
			},
		}
		return b.api.EditMessageText(EditMessageTextPayload{
			ChatID: chatID, MessageID: messageID, Text: text, ParseMode: "Markdown", ReplyMarkup: &keyboard,
		})
	}
	if strings.HasPrefix(data, "del_confirm_") {
		parts := strings.Split(data, "_")
		triggerID, _ := strconv.ParseInt(parts[2], 10, 64)
		channelID, _ := strconv.ParseInt(parts[4], 10, 64)
		page, _ := strconv.Atoi(parts[6])

		// Ambil record dulu untuk dapatkan teksnya sebelum dihapus
		triggerRecord, found, _ := b.store.GetTriggerByID(triggerID)

		if err := b.store.DeleteTriggerByID(triggerID); err != nil {
			log.Printf("failed to delete trigger %d: %v", triggerID, err)
		} else if found {
			// Tampilkan notifikasi pop-up dengan teks trigger
			alertData := struct{ Trigger string }{Trigger: triggerRecord.TriggerText}
			alertText := i18n.GetMessage(lang, "delete_success_alert", alertData)
			b.api.AnswerCallbackQuery(AnswerCallbackQueryPayload{CallbackQueryID: cb.ID, Text: alertText, ShowAlert: true})
		}
		// Segarkan kembali dasbor
		return b.sendManagementDashboard(chatID, messageID, lang, channelID, page)
	}


	if strings.HasPrefix(data, "manage_ch_") {
		parts := strings.Split(data, "_")
		channelID, _ := strconv.ParseInt(parts[2], 10, 64)
		page, _ := strconv.Atoi(parts[4])
		return b.sendManagementDashboard(chatID, messageID, lang, channelID, page)
	}

	// Router untuk menu Bantuan
	if strings.HasPrefix(data, "help_") {
		// Jika "help_main", tampilkan menu utama (dari /start)
		if data == "help_main" {
			text := i18n.GetMessage(lang, "help_main_text", nil)
			keyboard := InlineKeyboardMarkup{
				InlineKeyboard: [][]InlineKeyboardButton{
					{
						{Text: i18n.GetMessage(lang, "help_register_button", nil), CallbackData: "help_register"},
						{Text: i18n.GetMessage(lang, "help_learn_button", nil), CallbackData: "help_learn"},
					},
					{
						{Text: i18n.GetMessage(lang, "help_manage_button", nil), CallbackData: "help_manage"},
						{Text: i18n.GetMessage(lang, "help_formatting_button", nil), CallbackData: "help_formatting"},
					},
					{
						{Text: i18n.GetMessage(lang, "help_lang_button", nil), CallbackData: "help_lang"},
						{Text: i18n.GetMessage(lang, "help_cancel_button", nil), CallbackData: "help_cancel"},
					},
				},
			}
			return b.api.EditMessageText(EditMessageTextPayload{
				ChatID: chatID, MessageID: messageID, Text: text, ParseMode: "Markdown", ReplyMarkup: &keyboard,
			})
		}
		var helpDetailKey string
		switch data {
		case "help_register":
			helpDetailKey = "help_register_text"
		case "help_learn":
			helpDetailKey = "help_learn_text"
		case "help_manage": // Tambahkan case baru
			helpDetailKey = "help_manage_text"
		case "help_formatting": // Tambahkan case baru
			helpDetailKey = "help_formatting_text"
		case "help_lang":
			helpDetailKey = "help_lang_text"
		case "help_cancel":
			helpDetailKey = "help_cancel_text"
		}
		
		if helpDetailKey != "" {
			text := i18n.GetMessage(lang, helpDetailKey, nil)
			keyboard := InlineKeyboardMarkup{
				InlineKeyboard: [][]InlineKeyboardButton{
					{{Text: i18n.GetMessage(lang, "back_button", nil), CallbackData: "help_main"}},
				},
			}
			return b.api.EditMessageText(EditMessageTextPayload{
				ChatID: chatID, MessageID: messageID, Text: text, ParseMode: "Markdown", ReplyMarkup: &keyboard,
			})
		}
	}
	
	// Router untuk prompt bahasa (dari /start)
	if data == "lang_prompt" {
		return b.handleLangCommand(cb.Message, lang)
	}

	// Sisa logika callback query (lang_en, learn_channel_, dll) tetap sama
	if strings.HasPrefix(data, "lang_") {
		langCode := strings.TrimPrefix(data, "lang_")
		if err := b.store.SetUserLanguage(userID, langCode); err != nil {
			log.Printf("failed to set user language: %v", err)
			return b.api.AnswerCallbackQuery(AnswerCallbackQueryPayload{CallbackQueryID: cb.ID})
		}
		text := i18n.GetMessage(langCode, "lang_updated", nil)
		b.api.AnswerCallbackQuery(AnswerCallbackQueryPayload{
			CallbackQueryID: cb.ID, Text: text, ShowAlert: true,
		})
		return b.api.EditMessageText(EditMessageTextPayload{
			ChatID: chatID, MessageID: messageID, Text: text,
		})
	}

	if strings.HasPrefix(data, "learn_channel_") {
		channelIDStr := strings.TrimPrefix(data, "learn_channel_")
		channelID, _ := strconv.ParseInt(channelIDStr, 10, 64)

		b.states.SetState(userID, &UserState{
			Step: "awaiting_trigger", ChannelID: channelID,
		})

		text := i18n.GetMessage(lang, "learn_channel_selected", nil)
		return b.api.EditMessageText(EditMessageTextPayload{
			ChatID: chatID, MessageID: messageID, Text: text, ParseMode: "Markdown",
		})
	}
	
	if data == "show_placeholder_help" {
		helpText := i18n.GetMessage(lang, "placeholder_help_text", nil)
		backButton := InlineKeyboardButton{
			Text: i18n.GetMessage(lang, "back_button", nil), CallbackData: "back_to_response_prompt",
		}
		keyboard := InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{{backButton}},
		}

		return b.api.EditMessageText(EditMessageTextPayload{
			ChatID: chatID, MessageID: messageID, Text: helpText, ParseMode: "Markdown", ReplyMarkup: &keyboard,
		})
	}
	
	if data == "back_to_response_prompt" {
		state, found := b.states.GetState(userID)
		if !found || state.Step != "awaiting_response" {
			return b.api.EditMessageText(EditMessageTextPayload{
				ChatID: chatID, MessageID: messageID, Text: i18n.GetMessage(lang, "session_expired", nil),			})
		}

		textData := struct{ Trigger string }{Trigger: state.Trigger}
		text := i18n.GetMessage(lang, "learn_awaiting_response", textData)
		helpButton := InlineKeyboardButton{
			Text: i18n.GetMessage(lang, "placeholder_button", nil), CallbackData: "show_placeholder_help",
		}
		keyboard := InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{{helpButton}},
		}

		return b.api.EditMessageText(EditMessageTextPayload{
			ChatID: chatID, MessageID: messageID, Text: text, ParseMode: "Markdown", ReplyMarkup: &keyboard,
		})
	}

	return nil
}
// --- AKHIR PERUBAHAN ---

// --- AWAL PERUBAHAN ---
// FUNGSI LENGKAP YANG DIPERBARUI

func (b *Bot) sendManagementDashboard(chatID int64, messageID int, lang string, channelID int64, page int) error {
	const pageSize = 5 // 5 trigger per halaman

	triggers, err := b.store.GetTriggersByChannel(channelID)
	if err != nil {
		return err
	}
	
	channelInfo, _ := b.api.GetChat(channelID)
	
	// Logika Paginasi
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(triggers) {
		start = 0
		end = start + pageSize
		page = 1
	}
	if end > len(triggers) {
		end = len(triggers)
	}
	
	paginatedTriggers := triggers[start:end]
	totalPages := (len(triggers) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	// Bangun teks pesan
	titleData := struct {
		ChannelTitle string
		CurrentPage  int
		TotalPages   int
	}{channelInfo.Title, page, totalPages}
	
	var textBuilder strings.Builder
	templateMessage := i18n.GetMessage(lang, "manage_title", titleData)

	// BEFORE
	// tmpl, _ := text.New("title").Parse(templateMessage)

	// AFTER
	tmpl, _ := template.New("title").Parse(templateMessage) // <-- PERBAIKAN DI SINI
	
	_ = tmpl.Execute(&textBuilder, titleData)
	textBuilder.WriteString("\n\n")

	if len(paginatedTriggers) == 0 {
		textBuilder.WriteString("Tidak ada trigger yang ditemukan untuk channel ini.")
	}
	
	// Bangun tombol untuk setiap trigger
	var keyboard [][]InlineKeyboardButton
	for _, trigger := range paginatedTriggers {
		displayTrigger := trigger.TriggerText
		if len(displayTrigger) > 20 {
			displayTrigger = displayTrigger[:17] + "..."
		}
		
		row := []InlineKeyboardButton{
			{Text: displayTrigger, CallbackData: "noop"},
			{Text: i18n.GetMessage(lang, "delete_button", nil), CallbackData: fmt.Sprintf("del_prompt_%d_ch_%d_pg_%d", trigger.ID, channelID, page)},
		}
		keyboard = append(keyboard, row)
	}

	// Bangun tombol navigasi
	var navRow []InlineKeyboardButton
	if page > 1 {
		navRow = append(navRow, InlineKeyboardButton{Text: i18n.GetMessage(lang, "prev_button", nil), CallbackData: fmt.Sprintf("manage_ch_%d_page_%d", channelID, page-1)})
	}
	if page < totalPages {
		navRow = append(navRow, InlineKeyboardButton{Text: i18n.GetMessage(lang, "next_button", nil), CallbackData: fmt.Sprintf("manage_ch_%d_page_%d", channelID, page+1)})
	}
	if len(navRow) > 0 {
		keyboard = append(keyboard, navRow)
	}

	backToHelpRow := []InlineKeyboardButton{
		{Text: i18n.GetMessage(lang, "back_to_main_menu_button", nil), CallbackData: "help_main"},
	}
	keyboard = append(keyboard, backToHelpRow)
	
	return b.api.EditMessageText(EditMessageTextPayload{
		ChatID: chatID, MessageID: messageID, Text: textBuilder.String(), ParseMode: "Markdown", ReplyMarkup: &InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
}
// --- AKHIR PERUBAHAN ---

// Langkah 1 dari sesi /learn
func (b *Bot) handleLearnCommand(msg *Message, lang string) error {
	userID := msg.From.ID
	chatID := msg.Chat.ID
	
	// Langkah 1: Coba ambil dari cache terlebih dahulu
	cachedChannels, found := b.cache.Get(userID)
	if found {
		log.Printf("cache hit for user %d", userID)
		// Jika ditemukan, langsung gunakan data dari cache (super cepat)
		return b.sendChannelSelection(chatID, cachedChannels, lang)
	}

	log.Printf("cache miss for user %d, performing full check", userID)
	
	// Langkah 2: Jika tidak ada di cache (lambat, hanya terjadi sesekali)
	allChannels, err := b.store.GetRegisteredChannels()
	if err != nil {
		log.Printf("error getting registered channels: %v", err)
		return err
	}

	var userAdminChannels []storage.RegisteredChannel
	for _, channel := range allChannels {
		isAdmin, _ := b.isUserAdmin(channel.ChannelID, userID)
		if isAdmin {
			userAdminChannels = append(userAdminChannels, channel)
		}
	}

	// Langkah 3: Simpan hasilnya ke cache untuk penggunaan selanjutnya
	b.cache.Set(userID, userAdminChannels)

	if len(userAdminChannels) == 0 {
		text := i18n.GetMessage(lang, "learn_no_channels_found", nil)
		return b.api.SendMessage(SendMessagePayload{ChatID: chatID, Text: text, ParseMode: "Markdown"})
	}

	return b.sendChannelSelection(chatID, userAdminChannels, lang)
}

// Fungsi helper baru untuk menghindari duplikasi kode
func (b *Bot) sendChannelSelection(chatID int64, channels []storage.RegisteredChannel, lang string) error {
	if len(channels) == 0 {
		text := i18n.GetMessage(lang, "learn_no_channels_found", nil)
		return b.api.SendMessage(SendMessagePayload{ChatID: chatID, Text: text, ParseMode: "Markdown"})
	}
	
	var keyboard [][]InlineKeyboardButton
	for _, ch := range channels {
		button := InlineKeyboardButton{
			Text:         ch.Title,
			CallbackData: fmt.Sprintf("learn_channel_%d", ch.ChannelID),
		}
		keyboard = append(keyboard, []InlineKeyboardButton{button})
	}
	text := i18n.GetMessage(lang, "learn_prompt_channel", nil)

	return b.api.SendMessage(SendMessagePayload{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: &InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
}

// Menangani pesan lanjutan dalam sesi (memasukkan trigger/response)
func (b *Bot) handleSessionMessage(msg *Message, state *UserState, lang string) error {
	userID := msg.From.ID

	switch state.Step {

	case "awaiting_registration_forward":
		if msg.ForwardFromChat == nil {
			text := i18n.GetMessage(lang, "register_invalid_forward", nil)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}

		channelInfo := msg.ForwardFromChat
		isAdmin, err := b.isUserAdmin(channelInfo.ID, userID) // userID sudah didefinisikan di awal switch

		if err != nil {
			log.Printf("error checking admin status for channel %d user %d: %v", channelInfo.ID, userID, err)
			errorText := fmt.Sprintf("❌ Failed to verify admin status. Make sure I am an administrator in '%s' and try again in a few seconds. (API Error: %v)", channelInfo.Title, err)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: errorText})
		}
		if !isAdmin {
			log.Printf("register failed for channel %d: user %d is not admin.", channelInfo.ID, userID)
			text := i18n.GetMessage(lang, "register_fail_not_admin", nil)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}

		if err := b.store.RegisterChannel(channelInfo.ID, channelInfo.Title, userID); err != nil {
			log.Printf("register failed for %d: could not save to storage: %v", channelInfo.ID, err)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: "An internal error occurred."})
		}

		// BEFORE
		// b.cache.Set(userID, nil)
		// AFTER
		b.cache.Invalidate(userID) // Hapus cache setelah registrasi berhasil
		b.states.ClearState(userID)

		data := struct{ ChannelTitle string }{ChannelTitle: channelInfo.Title}
		text := i18n.GetMessage(lang, "register_success", data)
		return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text, ParseMode: "Markdown"})
// --- AKHIR PERUBAHAN ---

	case "awaiting_trigger":
		state.Trigger = msg.Text
		state.Step = "awaiting_response_type"
		b.states.SetState(userID, state)

		textData := struct{ Trigger string }{Trigger: msg.Text}
		text := i18n.GetMessage(lang, "learn_awaiting_response_type", textData)
		keyboard := InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: i18n.GetMessage(lang, "reply_type_text", nil), CallbackData: "learn_type_text"},
					{Text: i18n.GetMessage(lang, "reply_type_photo", nil), CallbackData: "learn_type_photo"},
					{Text: i18n.GetMessage(lang, "reply_type_sticker", nil), CallbackData: "learn_type_sticker"},
				},
				{
					{Text: i18n.GetMessage(lang, "reply_type_document", nil), CallbackData: "learn_type_document"},
					{Text: i18n.GetMessage(lang, "reply_type_gif", nil), CallbackData: "learn_type_animation"},
				},
				{
					{Text: i18n.GetMessage(lang, "reply_type_audio", nil), CallbackData: "learn_type_audio"},
				},
			},
		}
		return b.api.SendMessage(SendMessagePayload{
			ChatID: msg.Chat.ID, Text: text, ParseMode: "Markdown", ReplyMarkup: &keyboard,
		})

	case "awaiting_text":
		// BEFORE
		// (Tidak ada penanganan error)
		// AFTER
		if msg.Text != "" {
			record := storage.TriggerRecord{
				ChannelID:    state.ChannelID,
				TriggerText:  state.Trigger,
				ResponseType: "text",
				ResponseText: msg.Text,
			}
			return b.finalizeLearnSession(userID, msg.Chat.ID, lang, record)
		} else {
			data := struct{ ExpectedType string }{"text"}
			text := i18n.GetMessage(lang, "learn_wrong_file_type", data)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}

	case "awaiting_photo":
		if msg.Photo != nil && len(msg.Photo) > 0 {
			bestPhoto := msg.Photo[0]
			for _, photo := range msg.Photo {
				if photo.FileSize > bestPhoto.FileSize {
					bestPhoto = photo
				}
			}
			record := storage.TriggerRecord{
				ChannelID:      state.ChannelID,
				TriggerText:    state.Trigger,
				ResponseType:   "photo",
				ResponseFileID: bestPhoto.FileID,
				ResponseText:   msg.Caption,
			}
			return b.finalizeLearnSession(userID, msg.Chat.ID, lang, record)
		} else {
			data := struct{ ExpectedType string }{"image"}
			text := i18n.GetMessage(lang, "learn_wrong_file_type", data)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}

	case "awaiting_sticker":
		if msg.Sticker != nil {
			record := storage.TriggerRecord{
				ChannelID:      state.ChannelID,
				TriggerText:    state.Trigger,
				ResponseType:   "sticker",
				ResponseFileID: msg.Sticker.FileID,
			}
			return b.finalizeLearnSession(userID, msg.Chat.ID, lang, record)
		} else {
			data := struct{ ExpectedType string }{"sticker"}
			text := i18n.GetMessage(lang, "learn_wrong_file_type", data)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}
		
	// Lakukan hal yang sama untuk semua jenis media lainnya...
	case "awaiting_document":
		if msg.Document != nil {
			record := storage.TriggerRecord{
				ChannelID:      state.ChannelID,
				TriggerText:    state.Trigger,
				ResponseType:   "document",
				ResponseFileID: msg.Document.FileID,
				ResponseText:   msg.Caption,
			}
			return b.finalizeLearnSession(userID, msg.Chat.ID, lang, record)
		} else {
			data := struct{ ExpectedType string }{"document"}
			text := i18n.GetMessage(lang, "learn_wrong_file_type", data)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}

	case "awaiting_animation":
		if msg.Animation != nil {
			record := storage.TriggerRecord{
				ChannelID:      state.ChannelID,
				TriggerText:    state.Trigger,
				ResponseType:   "animation",
				ResponseFileID: msg.Animation.FileID,
				ResponseText:   msg.Caption,
			}
			return b.finalizeLearnSession(userID, msg.Chat.ID, lang, record)
		} else {
			data := struct{ ExpectedType string }{"GIF"}
			text := i18n.GetMessage(lang, "learn_wrong_file_type", data)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}

	case "awaiting_audio":
		if msg.Audio != nil {
			record := storage.TriggerRecord{
				ChannelID:      state.ChannelID,
				TriggerText:    state.Trigger,
				ResponseType:   "audio",
				ResponseFileID: msg.Audio.FileID,
				ResponseText:   msg.Caption,
			}
			return b.finalizeLearnSession(userID, msg.Chat.ID, lang, record)
		} else {
			data := struct{ ExpectedType string }{"audio file"}
			text := i18n.GetMessage(lang, "learn_wrong_file_type", data)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
		}
	}
	return nil

	
}

// Fungsi helper baru untuk menyelesaikan sesi
func (b *Bot) finalizeLearnSession(userID, chatID int64, lang string, record storage.TriggerRecord) error {
	if err := b.store.Set(record); err != nil {
		log.Printf("failed to save final trigger: %v", err)
		b.api.SendMessage(SendMessagePayload{ChatID: chatID, Text: "An error occurred."})
		return err
	}
	
	b.states.ClearState(userID)
	
	textData := struct{ Trigger string }{Trigger: record.TriggerText}
	text := i18n.GetMessage(lang, "learn_success", textData)
	return b.api.SendMessage(SendMessagePayload{ChatID: chatID, Text: text, ParseMode: "Markdown"})
}

// ... (handleAutoReply, isUserAdmin, dll tidak berubah)
func (b *Bot) isUserAdmin(chatID, userID int64) (bool, error) {
	admins, err := b.api.GetChatAdministrators(chatID)
	if err != nil {
		return false, err
	}
	for _, admin := range admins {
		if admin.User.ID == userID {
			return true, nil
		}
	}
	return false, nil
}

// --- AWAL PERUBAHAN ---
// FUNGSI LENGKAP YANG DIPERBARUI
func (b *Bot) handleAutoReply(msg *Message) error {
	var searchID int64
	dmChatInfo, err := b.api.GetChat(msg.Chat.ID)
	if err != nil {
		log.Printf("could not get detailed info for DM chat %d: %v", msg.Chat.ID, err)
		return nil
	}
	if dmChatInfo.ParentChat != nil && dmChatInfo.ParentChat.ID != 0 {
		searchID = dmChatInfo.ParentChat.ID
	} else {
		searchID = msg.Chat.ID
	}

	record, found, err := b.store.Get(searchID, msg.Text)
	if err != nil || !found {
		return err
	}

	log.Printf("found match for trigger '%s'. replying with type '%s'", msg.Text, record.ResponseType)

	topicID := msg.DirectMessagesTopic.TopicID

	// Ganti switch lama dengan yang ini
	switch record.ResponseType {
	case "text":
		finalText := strings.Replace(record.ResponseText, "{{user_first_name}}", msg.From.FirstName, -1)
		return b.api.SendMessage(SendMessagePayload{
			ChatID: msg.Chat.ID, Text: finalText, ParseMode: "Markdown", DirectMessagesTopicID: topicID,
		})
	case "photo":
		return b.api.SendPhoto(SendPhotoPayload{
			ChatID: msg.Chat.ID, Photo: record.ResponseFileID, Caption: record.ResponseText, ParseMode: "Markdown", DirectMessagesTopicID: topicID,
		})
	case "sticker":
		return b.api.SendSticker(SendStickerPayload{
			ChatID: msg.Chat.ID, Sticker: record.ResponseFileID, DirectMessagesTopicID: topicID,
		})
	case "document":
		return b.api.SendDocument(SendDocumentPayload{
			ChatID: msg.Chat.ID, Document: record.ResponseFileID, Caption: record.ResponseText, DirectMessagesTopicID: topicID,
		})
	case "animation":
		return b.api.SendAnimation(SendAnimationPayload{
			ChatID: msg.Chat.ID, Animation: record.ResponseFileID, Caption: record.ResponseText, DirectMessagesTopicID: topicID,
		})
	case "audio":
		return b.api.SendAudio(SendAudioPayload{
			ChatID: msg.Chat.ID, Audio: record.ResponseFileID, Caption: record.ResponseText, DirectMessagesTopicID: topicID,
		})
	}

	return nil
}
// --- AKHIR PERUBAHAN ---