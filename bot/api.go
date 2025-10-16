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

func (a *API) SendMessage(payload SendMessagePayload) error {
	url := fmt.Sprintf("%s/sendMessage", a.baseURL)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal send message payload: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-ok status code (%d): %s", resp.StatusCode, string(body))
	}
	log.Printf("successfully sent message to chat id %d", payload.ChatID)
	return nil
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

// ----- FUNGSI BARU -----
func (a *API) EditMessageText(payload EditMessageTextPayload) error {
	url := fmt.Sprintf("%s/editMessageText", a.baseURL)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal edit message payload: %w", err)
	}
	// ... (Sisa request HTTP mirip dengan SendMessage)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send edit message request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-ok status code on edit (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (a *API) AnswerCallbackQuery(payload AnswerCallbackQueryPayload) error {
	url := fmt.Sprintf("%s/answerCallbackQuery", a.baseURL)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal answer callback query payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create new request for answer callback query: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send answer callback query request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-ok status code on answer callback query (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}