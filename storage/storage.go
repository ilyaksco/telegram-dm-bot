package storage

type RegisteredChannel struct {
	ChannelID int64
	Title     string
}

type Storage interface {
	Set(channelID int64, trigger, response string) error
	Get(channelID int64, trigger string) (string, bool, error)
	// GetAllTriggers tidak kita butuhkan lagi, kita ganti dengan yang lebih spesifik
	GetTriggersByChannel(channelID int64) ([]TriggerRecord, error)
	DeleteTriggerByID(triggerID int64) error
	SetUserLanguage(userID int64, langCode string) error
	GetUserLanguage(userID int64) (string, bool, error)
	RegisterChannel(channelID int64, title string, userID int64) error
	GetRegisteredChannels() ([]RegisteredChannel, error)

}