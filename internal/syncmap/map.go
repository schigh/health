package syncmap

import "sync"

// Map is a generic thread-safe map.
type Map[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

// Get returns the value for the given key.
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.data[key]
	return v, ok
}

// Set adds or updates a key-value pair.
func (m *Map[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data == nil {
		m.data = make(map[K]V)
	}
	m.data[key] = value
}

// Delete removes a key from the map.
func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
}

// Size returns the number of entries.
func (m *Map[K, V]) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.data)
}

// Keys returns all keys in the map.
func (m *Map[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]K, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

// Each iterates over each key-value pair. If fn returns false, iteration stops.
// Do not call mutating methods from within fn — it will deadlock.
func (m *Map[K, V]) Each(fn func(K, V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for k, v := range m.data {
		if !fn(k, v) {
			return
		}
	}
}

// Clear removes all entries.
func (m *Map[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[K]V)
}

// Value returns a shallow copy of the underlying map.
func (m *Map[K, V]) Value() map[K]V {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[K]V, len(m.data))
	for k, v := range m.data {
		out[k] = v
	}
	return out
}

// AbsorbMap merges all entries from src into the map, overwriting existing keys.
func (m *Map[K, V]) AbsorbMap(src map[K]V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data == nil {
		m.data = make(map[K]V)
	}
	for k, v := range src {
		m.data[k] = v
	}
}
