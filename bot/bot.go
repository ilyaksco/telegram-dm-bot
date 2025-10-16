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
	cache  *AdminCache // Tambahkan cache
}

func NewBot(cfg *config.Config, store storage.Storage) *Bot {
	return &Bot{
		api:    NewAPI(cfg.BotToken),
		store:  store,
		states: NewStateManager(),
		cache:  NewAdminCache(), // Inisialisasi cache
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
	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: i18n.GetMessage(lang, "help_button", nil), CallbackData: "help_main"},
				{Text: i18n.GetMessage(lang, "language_button", nil), CallbackData: "lang_prompt"},
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
			{{Text: i18n.GetMessage(lang, "help_register_button", nil), CallbackData: "help_register"}},
			{{Text: i18n.GetMessage(lang, "help_learn_button", nil), CallbackData: "help_learn"}},
			{{Text: i18n.GetMessage(lang, "help_lang_button", nil), CallbackData: "help_lang"}},
			{{Text: i18n.GetMessage(lang, "help_cancel_button", nil), CallbackData: "help_cancel"}},
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
func (b *Bot) handleRegisterCommand(msg *Message, lang string) error {
	parts := strings.Split(msg.Text, " ")
	if len(parts) != 2 || !strings.HasPrefix(parts[1], "@") {
		text := i18n.GetMessage(lang, "register_usage", nil)
		return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
	}

	channelUsername := parts[1]
	channelInfo, err := b.api.GetChat(channelUsername)
	if err != nil {
		log.Printf("register failed for %s: could not get chat info: %v", channelUsername, err)
		text := i18n.GetMessage(lang, "register_fail", nil)
		return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
	}

	isAdmin, err := b.isUserAdmin(channelInfo.ID, msg.From.ID)
	if err != nil || !isAdmin {
		log.Printf("register failed for %s: user %d is not admin or error occurred: %v", channelUsername, msg.From.ID, err)
		text := i18n.GetMessage(lang, "register_fail", nil)
		return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text})
	}

	if err := b.store.RegisterChannel(channelInfo.ID, channelInfo.Title, msg.From.ID); err != nil {
		log.Printf("register failed for %s: could not save to storage: %v", channelUsername, err)
		return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: "An internal error occurred."})
	}

	data := struct{ ChannelTitle string }{ChannelTitle: channelInfo.Title}
	text := i18n.GetMessage(lang, "register_success", data)
	return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text, ParseMode: "Markdown"})
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

	if strings.HasPrefix(data, "manage_ch_") {
		parts := strings.Split(data, "_")
		channelID, _ := strconv.ParseInt(parts[2], 10, 64)
		page, _ := strconv.Atoi(parts[4])

		return b.sendManagementDashboard(chatID, messageID, lang, channelID, page)
	}

	if strings.HasPrefix(data, "delete_trigger_") {
		parts := strings.Split(data, "_")
		triggerID, _ := strconv.ParseInt(parts[2], 10, 64)
		channelID, _ := strconv.ParseInt(parts[4], 10, 64)
		page, _ := strconv.Atoi(parts[6])
		triggerText := parts[7] // Kita butuh ini untuk ditampilkan di prompt

		textData := struct{ Trigger string }{Trigger: triggerText}
		text := i18n.GetMessage(lang, "confirm_delete_prompt", textData)
		keyboard := InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: i18n.GetMessage(lang, "confirm_delete_button", nil), CallbackData: fmt.Sprintf("confirm_delete_%d_ch_%d_page_%d_text_%s", triggerID, channelID, page, triggerText)},
					{Text: i18n.GetMessage(lang, "cancel_delete_button", nil), CallbackData: fmt.Sprintf("manage_ch_%d_page_%d", channelID, page)},
				},
			},
		}
		return b.api.EditMessageText(EditMessageTextPayload{
			ChatID: chatID, MessageID: messageID, Text: text, ParseMode: "Markdown", ReplyMarkup: &keyboard,
		})
	}

	if strings.HasPrefix(data, "confirm_delete_") {
		parts := strings.Split(data, "_")
		triggerID, _ := strconv.ParseInt(parts[2], 10, 64)
		channelID, _ := strconv.ParseInt(parts[4], 10, 64)
		page, _ := strconv.Atoi(parts[6])
		triggerText := parts[7]

		if err := b.store.DeleteTriggerByID(triggerID); err != nil {
			log.Printf("failed to delete trigger %d: %v", triggerID, err)
			// Handle error, mungkin dengan alert
		} else {
			// Tampilkan notifikasi pop-up
			alertData := struct{ Trigger string }{Trigger: triggerText}
			alertText := i18n.GetMessage(lang, "delete_success_alert", alertData)
			b.api.AnswerCallbackQuery(AnswerCallbackQueryPayload{CallbackQueryID: cb.ID, Text: alertText, ShowAlert: true})
		}
		// Segarkan kembali dasbor
		return b.sendManagementDashboard(chatID, messageID, lang, channelID, page)
	}

	// Router untuk menu Bantuan
	if strings.HasPrefix(data, "help_") {
		// Jika "help_main", tampilkan menu utama (dari /start)
		if data == "help_main" {
			// Edit pesan /start menjadi menu help
			text := i18n.GetMessage(lang, "help_main_text", nil)
			keyboard := InlineKeyboardMarkup{
				InlineKeyboard: [][]InlineKeyboardButton{
					{{Text: i18n.GetMessage(lang, "help_register_button", nil), CallbackData: "help_register"}},
					{{Text: i18n.GetMessage(lang, "help_learn_button", nil), CallbackData: "help_learn"}},
					{{Text: i18n.GetMessage(lang, "help_lang_button", nil), CallbackData: "help_lang"}},
					{{Text: i18n.GetMessage(lang, "help_cancel_button", nil), CallbackData: "help_cancel"}},
				},
			}
			return b.api.EditMessageText(EditMessageTextPayload{
				ChatID: chatID, MessageID: messageID, Text: text, ParseMode: "Markdown", ReplyMarkup: &keyboard,
			})
		}
		// Jika tombol fitur spesifik, tampilkan detailnya
		var helpDetailKey string
		switch data {
		case "help_register":
			helpDetailKey = "help_register_text"
		case "help_learn":
			helpDetailKey = "help_learn_text"
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

		text := "✅ Channel dipilih.\n\nSekarang, kirimkan **kata atau frasa pemicunya**."
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
				ChatID: chatID, MessageID: messageID, Text: "Sesi Anda telah berakhir. Silakan mulai lagi dengan /learn.",
			})
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
			{Text: displayTrigger, CallbackData: "noop"}, // <-- BERI AKSI KOSONG
			{Text: i18n.GetMessage(lang, "delete_button", nil), CallbackData: fmt.Sprintf("delete_trigger_%d_ch_%d_page_%d_text_%s", trigger.ID, channelID, page, trigger.TriggerText)},
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
		return b.sendChannelSelection(chatID, cachedChannels)
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

	return b.sendChannelSelection(chatID, userAdminChannels)
}

// Fungsi helper baru untuk menghindari duplikasi kode
func (b *Bot) sendChannelSelection(chatID int64, channels []storage.RegisteredChannel) error {
	var keyboard [][]InlineKeyboardButton
	for _, ch := range channels {
		button := InlineKeyboardButton{
			Text:         ch.Title,
			CallbackData: fmt.Sprintf("learn_channel_%d", ch.ChannelID),
		}
		keyboard = append(keyboard, []InlineKeyboardButton{button})
	}

	text := "Silakan pilih channel yang ingin Anda ajari:"
	return b.api.SendMessage(SendMessagePayload{
		ChatID: chatID,
		Text:   text,
		ReplyMarkup: &InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
}

// Menangani pesan lanjutan dalam sesi (memasukkan trigger/response)
func (b *Bot) handleSessionMessage(msg *Message, state *UserState, lang string) error {
	userID := msg.From.ID

	if state.Step == "awaiting_trigger" {
		state.Trigger = msg.Text
		state.Step = "awaiting_response"
		b.states.SetState(userID, state)

		textData := struct{ Trigger string }{Trigger: msg.Text}
		text := i18n.GetMessage(lang, "learn_awaiting_response", textData)
		
		// Buat tombol bantuan
		helpButton := InlineKeyboardButton{
			Text:         i18n.GetMessage(lang, "placeholder_button", nil),
			CallbackData: "show_placeholder_help",
		}
		keyboard := InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{{helpButton}},
		}

		return b.api.SendMessage(SendMessagePayload{
			ChatID:      msg.Chat.ID,
			Text:        text,
			ParseMode:   "Markdown",
			ReplyMarkup: &keyboard,
		})
	}

	if state.Step == "awaiting_response" {
		response := msg.Text
		err := b.store.Set(state.ChannelID, state.Trigger, response)
		b.states.ClearState(userID) // Sesi selesai

		if err != nil {
			log.Printf("failed to save final trigger: %v", err)
			return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: "Terjadi kesalahan saat menyimpan."})
		}

		text := fmt.Sprintf("✅ **Berhasil!**\n\nPemicu: `%s`\nRespon: `%s`\n\nTelah disimpan.", state.Trigger, response)
		return b.api.SendMessage(SendMessagePayload{ChatID: msg.Chat.ID, Text: text, ParseMode: "Markdown"})
	}

	return nil
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
func (b *Bot) handleAutoReply(msg *Message) error {
	var searchID int64
	var response string
	var found bool
	var err error

	// 1. Dapatkan info detail dari DM chat untuk mencari parent_chat
	dmChatInfo, err := b.api.GetChat(msg.Chat.ID)
	if err != nil {
		log.Printf("could not get detailed info for DM chat %d: %v", msg.Chat.ID, err)
		return nil // Gagal mendapat info, tidak bisa lanjut
	}

	// 2. Tentukan ID yang akan digunakan untuk mencari di database
	if dmChatInfo.ParentChat != nil && dmChatInfo.ParentChat.ID != 0 {
		searchID = dmChatInfo.ParentChat.ID
		log.Printf("DM chat %d belongs to parent channel %d. searching with parent ID.", msg.Chat.ID, searchID)
	} else {
		searchID = msg.Chat.ID
		log.Printf("could not find parent channel for DM chat %d. searching with its own ID.", searchID)
	}

	// 3. Cari di Supabase menggunakan ID yang sudah ditentukan
	response, found, err = b.store.Get(searchID, msg.Text)
	if err != nil {
		log.Printf("error retrieving trigger from storage for searchID %d: %v", searchID, err)
		return err
	}

	if !found {
		return nil // Tidak ada trigger yang cocok
	}

	userFirstName := msg.From.FirstName
	// Ganti placeholder di dalam teks balasan
	finalResponse := strings.Replace(response, "{{user_first_name}}", userFirstName, -1)
	// ---------------------------------------------------

	// 4. Kirim balasan
	log.Printf("found match for trigger '%s' using searchID %d. replying to topic %d in chat %d", msg.Text, searchID, msg.DirectMessagesTopic.TopicID, msg.Chat.ID) // <-- PERBAIKAN DI SINI
	return b.api.SendMessage(SendMessagePayload{
		ChatID:                msg.Chat.ID,
		Text:                  finalResponse, // Kirim balasan yang sudah diproses
		DirectMessagesTopicID: msg.DirectMessagesTopic.TopicID, // <-- DAN DI SINI
		ParseMode:             "Markdown",
	})
}

