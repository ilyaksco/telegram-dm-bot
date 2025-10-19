package bot

import (
	"sync"
)

type UserState struct {
	Step         string // "awaiting_registration_forward", "awaiting_trigger", dll.	ChannelID    int64
	ChannelID    int64  // <-- TAMBAHKAN KEMBALI FIELD INI
	ChannelTitle string
	Trigger      string
	ResponseType string
}

type StateManager struct {
	mu    sync.RWMutex // Gembok Baca-Tulis
	users map[int64]*UserState
}

func NewStateManager() *StateManager {
	return &StateManager{
		users: make(map[int64]*UserState),
	}
}

func (sm *StateManager) SetState(userID int64, state *UserState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.users[userID] = state
}

func (sm *StateManager) GetState(userID int64) (*UserState, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	state, found := sm.users[userID]
	if found {
		sCopy := *state
		return &sCopy, true
	}
	return state, found
}

func (sm *StateManager) ClearState(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.users, userID)
}