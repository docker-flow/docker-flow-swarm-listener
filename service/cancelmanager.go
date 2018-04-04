package service

import (
	"context"
	"sync"
)

// CancelManaging manages canceling of contexts
type CancelManaging interface {
	Add(id string, reqID int64) context.Context
	Delete(id string, reqID int64) bool
	ForceDelete(id string) bool
}

type cancelPair struct {
	Cancel context.CancelFunc
	ReqID  int64
	Cnt    int
}

// CancelManager implements the `CancelManaging` interface that is thread safe
type CancelManager struct {
	v                  map[string]cancelPair
	mux                sync.Mutex
	startingCnt        int
	cancelBeforeAdding bool
}

// NewCancelManager creates a new `CancelManager`
// `startingCnt` is the number of expected request to send out
func NewCancelManager(startingCnt int, cancelBeforeAdding bool) *CancelManager {
	return &CancelManager{
		v:                  map[string]cancelPair{},
		mux:                sync.Mutex{},
		startingCnt:        startingCnt,
		cancelBeforeAdding: cancelBeforeAdding,
	}
}

// Add creates an context for `id` and `reqID` and returns that context.
// If `id` exists in memory and cancelBeforeAdding is true, the task with that `id` will be canceled.
func (m *CancelManager) Add(id string, reqID int64) context.Context {
	m.mux.Lock()
	defer m.mux.Unlock()

	pair, ok := m.v[id]
	if m.cancelBeforeAdding && ok {
		pair.Cancel()
		delete(m.v, id)
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.v[id] = cancelPair{
		Cancel: cancel,
		ReqID:  reqID,
		Cnt:    m.startingCnt,
	}
	return ctx
}

// Delete calls cancel on context with the corresponding `id` and `reqID` and remove 'id'
// from memory If the corresponding `id` and `reqID` are not present, Delete does nothing.
// Returns true if item was deleted
func (m *CancelManager) Delete(id string, reqID int64) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	pair, ok := m.v[id]

	if !ok || pair.ReqID != reqID {
		return false
	}

	pair.Cnt = pair.Cnt - 1

	if pair.Cnt != 0 {
		m.v[id] = pair
		return false
	}

	pair.Cancel()
	delete(m.v, id)
	return true
}

// ForceDelete deletes an id without looking at the `reqID` and count
func (m *CancelManager) ForceDelete(id string) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	pair, ok := m.v[id]

	if !ok {
		return false
	}

	pair.Cancel()
	delete(m.v, id)
	return true
}
