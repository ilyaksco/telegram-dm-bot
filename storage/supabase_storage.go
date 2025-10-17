package storage

import (
	"fmt"
	"log"
	"strings"

	supa "github.com/supabase-community/supabase-go"
)

type SupabaseStorage struct {
	client *supa.Client
}

type TriggerRecord struct {
	ID           int64  `json:"id"` // Tambahkan ID
	ChannelID    int64  `json:"channel_id"`
	TriggerText  string `json:"trigger_text"`
	ResponseType   string `json:"response_type"`
	ResponseText string `json:"response_text"`
	ResponseFileID string `json:"response_file_id,omitempty"`
}

type UserRecord struct {
	UserID   int64  `json:"user_id"`
	LangCode string `json:"lang_code"`
}

func NewSupabaseStorage(url, key string) (*SupabaseStorage, error) {
	client, err := supa.NewClient(url, key, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create supabase client: %w", err)
	}
	log.Println("successfully connected to supabase")
	return &SupabaseStorage{client: client}, nil
}

func (s *SupabaseStorage) Set(record TriggerRecord) error {
	data := map[string]interface{}{
		"channel_id":       record.ChannelID,
		"trigger_text":     strings.ToLower(record.TriggerText),
		"response_type":    record.ResponseType,
		"response_text":    record.ResponseText,
		"response_file_id": record.ResponseFileID,
	}

	// Gunakan nama kolom yang unik untuk on_conflict, bukan nama constraint.
	_, _, err := s.client.From("triggers").
		Upsert(data, "channel_id,trigger_text", "", ""). // <-- PERBAIKAN DI SINI
		Execute()

	if err != nil {
		return fmt.Errorf("failed to upsert trigger: %w", err)
	}

	log.Printf("successfully stored trigger for channel %d", record.ChannelID)
	return nil
}

func (s *SupabaseStorage) Get(channelID int64, trigger string) (TriggerRecord, bool, error) {
	lowerTrigger := strings.ToLower(trigger)
	var results []TriggerRecord
	var emptyRecord TriggerRecord

	_, err := s.client.From("triggers").
		Select("*", "0", false).
		Eq("channel_id", fmt.Sprintf("%d", channelID)).
		Eq("trigger_text", lowerTrigger).
		ExecuteTo(&results)

	if err != nil {
		return emptyRecord, false, fmt.Errorf("failed to get trigger: %w", err)
	}
	if len(results) == 0 {
		return emptyRecord, false, nil
	}
	return results[0], true, nil
}

// ----- FUNGSI BARU -----
// Mengambil semua data trigger yang ada di database.
func (s *SupabaseStorage) GetTriggersByChannel(channelID int64) ([]TriggerRecord, error) {
	var results []TriggerRecord
	_, err := s.client.From("triggers").
		Select("*", "0", false).
		Eq("channel_id", fmt.Sprintf("%d", channelID)).
		ExecuteTo(&results)

	if err != nil {
		return nil, fmt.Errorf("failed to get triggers for channel: %w", err)
	}
	return results, nil
}

func (s *SupabaseStorage) DeleteTriggerByID(triggerID int64) error {
	_, _, err := s.client.From("triggers").
		Delete("", "").
		Eq("id", fmt.Sprintf("%d", triggerID)).
		Execute()

	if err != nil {
		return fmt.Errorf("failed to delete trigger by id: %w", err)
	}
	return nil
}

func (s *SupabaseStorage) SetUserLanguage(userID int64, langCode string) error {
	record := UserRecord{
		UserID:   userID,
		LangCode: langCode,
	}
	_, _, err := s.client.From("users").Upsert(record, "user_id", "", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to upsert user language: %w", err)
	}
	return nil
}

func (s *SupabaseStorage) GetUserLanguage(userID int64) (string, bool, error) {
	var results []UserRecord
	_, err := s.client.From("users").
		Select("lang_code", "0", false).
		Eq("user_id", fmt.Sprintf("%d", userID)).
		ExecuteTo(&results)

	if err != nil {
		return "", false, fmt.Errorf("failed to get user language: %w", err)
	}

	if len(results) == 0 {
		return "", false, nil
	}

	return results[0].LangCode, true, nil
}

type RegisteredChannelRecord struct {
	ChannelID int64  `json:"channel_id"`
	Title     string `json:"title"`
}

func (s *SupabaseStorage) RegisterChannel(channelID int64, title string, userID int64) error {
	record := map[string]interface{}{
		"channel_id":            channelID,
		"title":                 title,
		"registered_by_user_id": userID,
	}
	_, _, err := s.client.From("channels").Upsert(record, "channel_id", "", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to upsert channel: %w", err)
	}
	return nil
}

func (s *SupabaseStorage) GetRegisteredChannels() ([]RegisteredChannel, error) {
	var results []RegisteredChannelRecord
	_, err := s.client.From("channels").Select("channel_id, title", "0", false).ExecuteTo(&results)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered channels: %w", err)
	}

	// Konversi dari struct internal ke struct interface
	var channels []RegisteredChannel
	for _, rec := range results {
		channels = append(channels, RegisteredChannel{
			ChannelID: rec.ChannelID,
			Title:     rec.Title,
		})
	}
	return channels, nil
}

// --- AWAL PERUBAHAN ---
// Tambahkan fungsi baru ini di akhir file
func (s *SupabaseStorage) GetTriggerByID(triggerID int64) (TriggerRecord, bool, error) {
	var results []TriggerRecord
	var emptyRecord TriggerRecord

	_, err := s.client.From("triggers").
		Select("*", "0", false).
		Eq("id", fmt.Sprintf("%d", triggerID)).
		ExecuteTo(&results)

	if err != nil {
		return emptyRecord, false, fmt.Errorf("failed to get trigger by id: %w", err)
	}

	if len(results) == 0 {
		return emptyRecord, false, nil
	}
	return results[0], true, nil
}
// --- AKHIR PERUBAHAN ---