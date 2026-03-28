package std_test

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/schigh/health/v2"
)

// MockChecker is a simple mock that returns results from a pre-configured sequence.
type MockChecker struct {
	mu      sync.Mutex
	results []*health.CheckResult
	index   int
}

func NewMockChecker(results ...*health.CheckResult) *MockChecker {
	return &MockChecker{results: results}
}

func (m *MockChecker) Check(_ context.Context) *health.CheckResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.index >= len(m.results) {
		// return last result if we've exhausted the sequence
		return m.results[len(m.results)-1]
	}
	r := m.results[m.index]
	m.index++
	return r
}

// MockReporter is a minimal mock reporter for testing.
type MockReporter struct {
	running        uint32
	live           uint32
	ready          uint32
	startup        uint32
	runCount       uint32
	stopCount      uint32
	liveCount      uint32
	readyCount     uint32
	startupCount   uint32
	updateCount    uint32
	hcMx           sync.RWMutex
	hcs            map[string]*health.CheckResult
}

func (m *MockReporter) Run(_ context.Context) error {
	atomic.StoreUint32(&m.running, 1)
	atomic.AddUint32(&m.runCount, 1)
	return nil
}

func (m *MockReporter) Stop(_ context.Context) error {
	atomic.StoreUint32(&m.running, 0)
	atomic.AddUint32(&m.stopCount, 1)
	return nil
}

func (m *MockReporter) SetLiveness(_ context.Context, b bool) {
	if b {
		atomic.StoreUint32(&m.live, 1)
	} else {
		atomic.StoreUint32(&m.live, 0)
	}
	atomic.AddUint32(&m.liveCount, 1)
}

func (m *MockReporter) SetReadiness(_ context.Context, b bool) {
	if b {
		atomic.StoreUint32(&m.ready, 1)
	} else {
		atomic.StoreUint32(&m.ready, 0)
	}
	atomic.AddUint32(&m.readyCount, 1)
}

func (m *MockReporter) SetStartup(_ context.Context, b bool) {
	if b {
		atomic.StoreUint32(&m.startup, 1)
	} else {
		atomic.StoreUint32(&m.startup, 0)
	}
	atomic.AddUint32(&m.startupCount, 1)
}

func (m *MockReporter) UpdateHealthChecks(_ context.Context, checks map[string]*health.CheckResult) {
	m.hcMx.Lock()
	defer m.hcMx.Unlock()

	if m.hcs == nil {
		m.hcs = make(map[string]*health.CheckResult)
	}
	for k, v := range checks {
		m.hcs[k] = v
	}
	atomic.AddUint32(&m.updateCount, 1)
}
