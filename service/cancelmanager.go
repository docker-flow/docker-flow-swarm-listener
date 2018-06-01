package service

import (
	"context"
	"sync"
)

// CancelManaging manages canceling of contexts
type CancelManaging interface {
	Add(rootCtx context.Context, id string, reqID int64) context.Context
	Delete(id string, reqID int64) bool
}

type cancelPair struct {
	Cancel context.CancelFunc
	ReqID  int64
}

// CancelManager implements the `CancelManaging` interface that is thread safe
type CancelManager struct {
	v   map[string]cancelPair
	mux sync.Mutex
}

// NewCancelManager creates a new `CancelManager`
func NewCancelManager() *CancelManager {
	return &CancelManager{
		v:   map[string]cancelPair{},
		mux: sync.Mutex{},
	}
}

// Add creates an context for `id` and `reqID` and returns that context.
// If `id` exists in memory, the task with that `id` will be canceled.
func (m *CancelManager) Add(rootCtx context.Context, id string, reqID int64) context.Context {
	m.mux.Lock()
	defer m.mux.Unlock()

	pair, ok := m.v[id]
	if ok {
		pair.Cancel()
		delete(m.v, id)
	}

	ctx, cancel := context.WithCancel(rootCtx)
	m.v[id] = cancelPair{
		Cancel: cancel,
		ReqID:  reqID,
	}
	return ctx
}

// Delete calls cancel context with the corresponding `id` and `reqID` and
// removes 'id' from map
// If the corresponding `id` and `reqID` are not present, Delete does nothing.
// In all cases, Delete returns true if an item was deleted
func (m *CancelManager) Delete(id string, reqID int64) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	pair, ok := m.v[id]

	if !ok || pair.ReqID != reqID {
		return false
	}

	pair.Cancel()
	delete(m.v, id)
	return true
}
