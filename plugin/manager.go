package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrDuplicatePlugin = errors.New("plugin: duplicate plugin name")
	ErrMissingDep      = errors.New("plugin: missing dependency")
	ErrCyclicDep       = errors.New("plugin: cyclic dependency")
)

// Manager manages the lifecycle of plugins.
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	order   []string
}

// NewManager creates a new plugin Manager.
func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
	}
}

// Register adds a plugin to the manager.
func (m *Manager) Register(p Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := p.Name()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicatePlugin, name)
	}
	m.plugins[name] = p
	return nil
}

// Get returns a plugin by name.
func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

// Initialize initializes all registered plugins in dependency order.
func (m *Manager) Initialize(ctx context.Context, params InitParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, err := m.topologicalSort()
	if err != nil {
		return err
	}
	m.order = order

	params.Get = func(name string) (Plugin, bool) {
		p, ok := m.plugins[name]
		return p, ok
	}

	for _, name := range m.order {
		p := m.plugins[name]
		if err := p.Initialize(ctx, params); err != nil {
			return fmt.Errorf("plugin: initialize %s: %w", name, err)
		}
	}
	return nil
}

// Close closes all plugins in reverse initialization order.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for i := len(m.order) - 1; i >= 0; i-- {
		name := m.order[i]
		if p, ok := m.plugins[name]; ok {
			if err := p.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// Plugins returns all registered plugins.
func (m *Manager) Plugins() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Plugin, 0, len(m.plugins))
	for _, name := range m.order {
		if p, ok := m.plugins[name]; ok {
			result = append(result, p)
		}
	}
	return result
}

// topologicalSort sorts plugins by dependencies (Kahn's algorithm).
func (m *Manager) topologicalSort() ([]string, error) {
	// Check all dependencies exist
	for name, p := range m.plugins {
		for _, dep := range p.Dependencies() {
			if _, ok := m.plugins[dep]; !ok {
				return nil, fmt.Errorf("%w: %s requires %s", ErrMissingDep, name, dep)
			}
		}
	}

	// Build in-degree map
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)

	for name := range m.plugins {
		inDegree[name] = 0
	}
	for name, p := range m.plugins {
		for _, dep := range p.Dependencies() {
			inDegree[name]++
			dependents[dep] = append(dependents[dep], name)
		}
	}

	// Collect nodes with zero in-degree
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	var order []string
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		order = append(order, name)

		for _, dep := range dependents[name] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(order) != len(m.plugins) {
		return nil, ErrCyclicDep
	}

	return order, nil
}
