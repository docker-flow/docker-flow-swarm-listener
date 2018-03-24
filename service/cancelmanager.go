package service

import (
	"context"
	"sync"
)

// CancelManaging manages canceling of contexts
type CancelManaging interface {
	Add(id string, reqID int64) context.Context
	Delete(id string, reqID int64)
}

type cancelPair struct {
	Cancel context.CancelFunc
	ReqID  int64
	Cnt    int
}

// CancelManager implements the `CancelManaging` interface that is thread safe
type CancelManager struct {
	v           map[string]cancelPair
	mux         sync.Mutex
	startingCnt int
}

// NewCancelManager creates a new `CancelManager`
// `startingCnt` is the number of expected request to send out
func NewCancelManager(startingCnt int) *CancelManager {
	return &CancelManager{
		v:           map[string]cancelPair{},
		mux:         sync.Mutex{},
		startingCnt: startingCnt,
	}
}

// Add creates an context for `id` and `reqID` and returns that context.
// If `id` exists in memory, that task will be canceled.
// If `id` does not exist, a new task and context will be created.
func (m *CancelManager) Add(id string, reqID int64) context.Context {
	m.mux.Lock()
	defer m.mux.Unlock()

	pair, ok := m.v[id]
	if ok {
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
func (m *CancelManager) Delete(id string, reqID int64) {
	m.mux.Lock()
	defer m.mux.Unlock()

	pair, ok := m.v[id]

	if !ok || pair.ReqID != reqID {
		return
	}

	pair.Cnt = pair.Cnt - 1

	if pair.Cnt != 0 {
		m.v[id] = pair
		return
	}

	pair.Cancel()
	delete(m.v, id)
}
