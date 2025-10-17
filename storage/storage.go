// --- AWAL PERUBAHAN ---
// FUNGSI LENGKAP YANG DIPERBARUI
package storage

type RegisteredChannel struct {
	ChannelID int64
	Title     string
}

type Storage interface {
	// BEFORE
	// Set(channelID int64, trigger, response string) error
	// AFTER
	Set(record TriggerRecord) error
	Get(channelID int64, trigger string) (TriggerRecord, bool, error)
	GetTriggersByChannel(channelID int64) ([]TriggerRecord, error)
	GetTriggerByID(triggerID int64) (TriggerRecord, bool, error) // <-- TAMBAHKAN FUNGSI BARU INI
	DeleteTriggerByID(triggerID int64) error
	SetUserLanguage(userID int64, langCode string) error
	GetUserLanguage(userID int64) (string, bool, error)
	RegisterChannel(channelID int64, title string, userID int64) error
	GetRegisteredChannels() ([]RegisteredChannel, error)
}
// --- AKHIR PERUBAHAN ---