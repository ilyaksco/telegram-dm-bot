package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// ... (NewAPI, GetUpdates, SendMessage, GetChatAdministrators, GetChat tetap sama) ...
type API struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

func NewAPI(token string) *API {
	return &API{
		token:   token,
		baseURL: fmt.Sprintf("https://api.telegram.org/bot%s", token),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (a *API) sendPostRequest(method string, payload interface{}) error {
	url := fmt.Sprintf("%s/%s", a.baseURL, method)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal %s payload: %w", method, err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create new request for %s: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send %s request: %w", method, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-ok status code on %s (%d): %s", method, resp.StatusCode, string(body))
	}
	log.Printf("successfully sent %s", method)
	return nil
}

func (a *API) SendMessage(payload SendMessagePayload) error {
	return a.sendPostRequest("sendMessage", payload)
}

func (a *API) SendSticker(payload SendStickerPayload) error {
	return a.sendPostRequest("sendSticker", payload)
}

func (a *API) SendDocument(payload SendDocumentPayload) error {
	return a.sendPostRequest("sendDocument", payload)
}

func (a *API) SendAnimation(payload SendAnimationPayload) error {
	return a.sendPostRequest("sendAnimation", payload)
}

func (a *API) SendAudio(payload SendAudioPayload) error {
	return a.sendPostRequest("sendAudio", payload)
}

func (a *API) EditMessageText(payload EditMessageTextPayload) error {
	return a.sendPostRequest("editMessageText", payload)
}

func (a *API) AnswerCallbackQuery(payload AnswerCallbackQueryPayload) error {
	return a.sendPostRequest("answerCallbackQuery", payload)
}

func (a *API) SendPhoto(payload SendPhotoPayload) error {
	return a.sendPostRequest("sendPhoto", payload)
}

func (a *API) GetUpdates(offset int) ([]Update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=30", a.baseURL, offset)
	resp, err := a.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get updates: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp ApiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !apiResp.Ok {
		return nil, fmt.Errorf("telegram API returned an error: %s", string(body))
	}

	var updates []Update
	if err := json.Unmarshal(apiResp.Result, &updates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal updates result: %w", err)
	}

	return updates, nil
}



func (a *API) GetChatAdministrators(chatID int64) ([]ChatMember, error) {
	url := fmt.Sprintf("%s/getChatAdministrators?chat_id=%d", a.baseURL, chatID)
	resp, err := a.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat administrators: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for admins: %w", err)
	}

	var apiResp ApiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal admin response: %w", err)
	}

	if !apiResp.Ok {
		return nil, fmt.Errorf("telegram API returned an error on getChatAdministrators: %s", string(body))
	}

	var admins []ChatMember
	if err := json.Unmarshal(apiResp.Result, &admins); err != nil {
		return nil, fmt.Errorf("failed to unmarshal admins result: %w", err)
	}

	return admins, nil
}

func (a *API) GetChat(chatID interface{}) (*GetChatResponse, error) {
	url := fmt.Sprintf("%s/getChat?chat_id=%v", a.baseURL, chatID)
	resp, err := a.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for getChat: %w", err)
	}

	var apiResp ApiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal getChat response: %w", err)
	}

	if !apiResp.Ok {
		return nil, fmt.Errorf("telegram API returned an error on getChat: %s", string(body))
	}

	var chatInfo GetChatResponse
	if err := json.Unmarshal(apiResp.Result, &chatInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal getChat result: %w", err)
	}

	return &chatInfo, nil
}

// --- AWAL PERUBAHAN ---
// Tambahkan fungsi baru ini bersama fungsi "Get" lainnya

func (a *API) GetMe() (*User, error) {
	url := fmt.Sprintf("%s/getMe", a.baseURL)
	resp, err := a.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call getMe: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read getMe response body: %w", err)
	}

	var apiResp GetMeResponse // Gunakan struct GetMeResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		// Coba unmarshal sebagai error biasa jika getMe gagal
		var baseResp ApiResponse
		if json.Unmarshal(body, &baseResp) == nil && !baseResp.Ok {
			return nil, fmt.Errorf("telegram API error on getMe: %s", string(body))
		}
		return nil, fmt.Errorf("failed to unmarshal getMe response: %w", err)
	}

	if !apiResp.Ok {
		return nil, fmt.Errorf("telegram API returned not ok on getMe: %s", string(body))
	}

	return &apiResp.Result, nil
}
// --- AKHIR PERUBAHAN ---