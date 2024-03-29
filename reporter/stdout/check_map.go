// Code generated by github.com/schigh/carto.  DO NOT EDIT.
package stdout

import (
	"sync"

	v1 "github.com/schigh/health/pkg/v1"
)

// healthCheckMap wraps map[string]*v1.Check, and locks reads and writes with a mutex
type healthCheckMap struct {
	mx        sync.RWMutex
	impl      map[string]*v1.Check
	onceToken sync.Once
}

// Get gets the *v1.Check keyed by string.
func (m *healthCheckMap) Get(key string) (value *v1.Check) {
	defer m.mx.RUnlock()
	m.mx.RLock()

	value = m.impl[key]

	return
}

// Keys will return all keys in the healthCheckMap's internal map
func (m *healthCheckMap) Keys() (keys []string) {
	defer m.mx.RUnlock()
	m.mx.RLock()

	keys = make([]string, len(m.impl))
	var i int
	for k := range m.impl {
		keys[i] = k
		i++
	}

	return
}

// Set will add an element to the healthCheckMap's internal map with the specified key
func (m *healthCheckMap) Set(key string, value *v1.Check) {
	defer m.mx.Unlock()
	m.mx.Lock()

	m.onceToken.Do(func() {
		m.impl = make(map[string]*v1.Check)
	})
	m.impl[key] = value
}

// Absorb will take all the keys and values from another healthCheckMap's internal map and
// overwrite any existing keys
func (m *healthCheckMap) Absorb(otherMap *healthCheckMap) {
	defer otherMap.mx.RUnlock()
	defer m.mx.Unlock()
	m.mx.Lock()
	otherMap.mx.RLock()

	m.onceToken.Do(func() {
		m.impl = make(map[string]*v1.Check)
	})
	for k, v := range otherMap.impl {
		m.impl[k] = v
	}
}

// AbsorbMap will take all the keys and values from another map and overwrite any existing keys
func (m *healthCheckMap) AbsorbMap(regularMap map[string]*v1.Check) {
	defer m.mx.Unlock()
	m.mx.Lock()

	m.onceToken.Do(func() {
		m.impl = make(map[string]*v1.Check)
	})
	for k, v := range regularMap {
		m.impl[k] = v
	}
}

// Delete will remove a *v1.Check from the map by key
func (m *healthCheckMap) Delete(key string) {
	defer m.mx.Unlock()
	m.mx.Lock()

	m.onceToken.Do(func() {
		m.impl = make(map[string]*v1.Check)
	})
	delete(m.impl, key)
}

// Clear will remove all elements from the map
func (m *healthCheckMap) Clear() {
	defer m.mx.Unlock()
	m.mx.Lock()

	m.impl = make(map[string]*v1.Check)
}

// Value returns a copy of the underlying map[string]*v1.Check
func (m *healthCheckMap) Value() map[string]*v1.Check {
	defer m.mx.RUnlock()
	m.mx.RLock()

	out := make(map[string]*v1.Check, len(m.impl))
	for k, v := range m.impl {
		out[k] = v
	}

	return out
}

// Size returns the number of elements in the underlying map[string]*v1.Check
func (m *healthCheckMap) Size() int {
	defer m.mx.RUnlock()
	m.mx.RLock()

	return len(m.impl)
}

// Each runs a function over each key/value pair in the healthCheckMap
// If the function returns false, the interation through the underlying
// map will halt.
// This function does not mutate the underlying map, although the values
// of the map may be mutated in place
//
//	!!! Warning: calls to any mutating functions of healthCheckMap
//	!!! will deadlock if called from within the supplied function
func (m *healthCheckMap) Each(f func(key string, value *v1.Check) bool) {
	defer m.mx.Unlock()
	m.mx.Lock()

	for _k := range m.impl {
		_v := m.impl[_k]
		if !f(_k, _v) {
			return
		}
	}
}
