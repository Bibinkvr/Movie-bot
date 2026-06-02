package index

import (
	"context"
	"sync"
	"time"
)

// Manager allows for managing active index operations conveniently.
// Must be initialised using NewManager at app startup.
type Manager struct {
	mu         sync.RWMutex
	operations map[string]*Operation
}

// NewManager intialises a new index manager.
func NewManager() *Manager {
	return &Manager{
		operations: make(map[string]*Operation),
	}
}

// GetOperation fetches the operation with corresponding pid.
func (m *Manager) GetOperation(pid string) (*Operation, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.operations[pid]
	return o, ok
}

// CancelOperation cancels the active operation.
// NOTE: does not delete from database or set status to paused.
func (m *Manager) CancelOperation(pid string) bool {
	m.mu.RLock()
	o, ok := m.operations[pid]
	m.mu.RUnlock()
	if !ok {
		return false
	}

	o.cancelFunc()
	return true
}

// RemoveOperationIfSame deletes the operation from the active operations map
// only if the stored pointer matches the given one. This prevents an old
// goroutine from accidentally removing a replacement operation.
func (m *Manager) RemoveOperationIfSame(pid string, o *Operation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.operations[pid]; ok && current == o {
		delete(m.operations, pid)
	}
}

// InsertOperation adds an operation to the actove operations map.
func (m *Manager) InsertOperation(o *Operation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operations[o.ID] = o
}

// RunOperation starts the index operation from the current message in the background.
// If an operation with the same ID already exists, it queues the new one after the old one exits.
func (m *Manager) RunOperation(ctx context.Context, o *Operation) {
	m.mu.Lock()
	old, exists := m.operations[o.ID]
	if exists {
		old.cancelFunc()
		m.mu.Unlock()
		go func() {
			select {
			case <-old.completedSignal:
			case <-time.After(30 * time.Second): // safety timeout
			}
			m.mu.Lock()
			m.operations[o.ID] = o
			m.mu.Unlock()
			o.run(ctx)
		}()
		return
	}
	m.operations[o.ID] = o
	m.mu.Unlock()
	go o.run(ctx)
}
