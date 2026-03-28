package test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/schigh/health/v2"
)

// Reporter is a reporter used for testing.
type Reporter struct {
	hcMx           sync.RWMutex // mutex for health checks
	running        uint32       // flag to determine if the reporter is running
	runningToggles uint32       // number of times running changed state
	live           uint32       // flag to determine if the reporter shows liveness
	liveToggles    uint32       // number of times liveness changed state
	ready          uint32       // flag to determine if the reporter shows readiness
	readyToggles   uint32       // number of times readiness changed state
	startup        uint32       // flag to determine if the reporter shows startup
	startupToggles uint32       // number of times startup changed state
	hcUpdates      uint32       // number of times health checks have been updated

	liveUp     uint32                          // number of times liveness toggled true
	liveDown   uint32                          // number of times liveness toggled false
	readyUp    uint32                          // number of times readiness toggled true
	readyDown  uint32                          // number of times readiness toggled false
	startupUp  uint32                          // number of times startup toggled true
	startupDown uint32                         // number of times startup toggled false
	hc         map[string]*health.CheckResult  // internal map of health checks
}

// Report is a snapshot of this reporter's state.
type Report struct {
	IsRunning                bool                           `json:"-"`
	NumRunningStateChanges   uint32                         `json:"-"`
	IsLive                   bool                           `json:"-"`
	NumLivenessStateChanges  uint32                         `json:"-"`
	NumLivenessSetTrue       uint32                         `json:"-"`
	NumLivenessSetFalse      uint32                         `json:"-"`
	IsReady                  bool                           `json:"-"`
	NumReadinessStateChanges uint32                         `json:"-"`
	NumReadinessSetTrue      uint32                         `json:"-"`
	NumReadinessSetFalse     uint32                         `json:"-"`
	IsStartup                bool                           `json:"-"`
	NumStartupStateChanges   uint32                         `json:"-"`
	NumStartupSetTrue        uint32                         `json:"-"`
	NumStartupSetFalse       uint32                         `json:"-"`
	NumHealthCheckUpdates    uint32                         `json:"-"`
	HealthChecks             map[string]*health.CheckResult `json:"-"`
}

func (r Report) MarshalJSON() ([]byte, error) {
	type alias struct {
		IsRunning                bool              `json:"isRunning"`
		NumRunningStateChanges   uint32            `json:"numRunningStateChanges"`
		IsLive                   bool              `json:"isLive"`
		NumLivenessStateChanges  uint32            `json:"numLivenessStateChanges"`
		NumLivenessSetTrue       uint32            `json:"numLivenessSetTrue"`
		NumLivenessSetFalse      uint32            `json:"numLivenessSetFalse"`
		IsReady                  bool              `json:"isReady"`
		NumReadinessStateChanges uint32            `json:"numReadinessStateChanges"`
		NumReadinessSetTrue      uint32            `json:"numReadinessSetTrue"`
		NumReadinessSetFalse     uint32            `json:"numReadinessSetFalse"`
		IsStartup                bool              `json:"isStartup"`
		NumStartupStateChanges   uint32            `json:"numStartupStateChanges"`
		NumStartupSetTrue        uint32            `json:"numStartupSetTrue"`
		NumStartupSetFalse       uint32            `json:"numStartupSetFalse"`
		NumHealthCheckUpdates    uint32            `json:"numHealthCheckUpdates"`
		HealthChecks             map[string]any    `json:"healthChecks"`
	}
	out := alias{
		IsRunning:                r.IsRunning,
		NumRunningStateChanges:   r.NumRunningStateChanges,
		IsLive:                   r.IsLive,
		NumLivenessStateChanges:  r.NumLivenessStateChanges,
		IsReady:                  r.IsReady,
		NumReadinessStateChanges: r.NumReadinessStateChanges,
		NumHealthCheckUpdates:    r.NumHealthCheckUpdates,
		NumLivenessSetTrue:       r.NumLivenessSetTrue,
		NumLivenessSetFalse:      r.NumLivenessSetFalse,
		NumReadinessSetTrue:      r.NumReadinessSetTrue,
		NumReadinessSetFalse:     r.NumReadinessSetFalse,
		IsStartup:                r.IsStartup,
		NumStartupStateChanges:   r.NumStartupStateChanges,
		NumStartupSetTrue:        r.NumStartupSetTrue,
		NumStartupSetFalse:       r.NumStartupSetFalse,
	}

	hcs := make(map[string]any)
	for k, hc := range r.HealthChecks {
		hcs[k] = hc
	}
	out.HealthChecks = hcs

	return json.Marshal(out)
}

// FromReporter is a convenience func for converting a health.Reporter implementation into a *Reporter instance.
func FromReporter(r health.Reporter) (*Reporter, bool) {
	reporter, ok := r.(*Reporter)
	if ok {
		return reporter, true
	}

	return &Reporter{}, false
}

// Report returns a summary of the test reporter.
func (t *Reporter) Report() Report {
	defer t.hcMx.Unlock()
	t.hcMx.Lock()

	healthChecks := make(map[string]*health.CheckResult)
	for k := range t.hc {
		healthChecks[k] = t.hc[k]
	}

	return Report{
		IsRunning:                atomic.LoadUint32(&t.running) == 1,
		NumRunningStateChanges:   atomic.LoadUint32(&t.runningToggles),
		IsLive:                   atomic.LoadUint32(&t.live) == 1,
		NumLivenessStateChanges:  atomic.LoadUint32(&t.liveToggles),
		IsReady:                  atomic.LoadUint32(&t.ready) == 1,
		NumReadinessStateChanges: atomic.LoadUint32(&t.readyToggles),
		NumHealthCheckUpdates:    atomic.LoadUint32(&t.hcUpdates),
		NumLivenessSetTrue:       atomic.LoadUint32(&t.liveUp),
		NumLivenessSetFalse:      atomic.LoadUint32(&t.liveDown),
		NumReadinessSetTrue:      atomic.LoadUint32(&t.readyUp),
		NumReadinessSetFalse:     atomic.LoadUint32(&t.readyDown),
		IsStartup:                atomic.LoadUint32(&t.startup) == 1,
		NumStartupStateChanges:   atomic.LoadUint32(&t.startupToggles),
		NumStartupSetTrue:        atomic.LoadUint32(&t.startupUp),
		NumStartupSetFalse:       atomic.LoadUint32(&t.startupDown),
		HealthChecks:             healthChecks,
	}
}

func boop(ptr *uint32, b bool, truePtr, falsePtr *uint32) {
	if b {
		atomic.StoreUint32(ptr, 1)
		if truePtr != nil {
			atomic.AddUint32(truePtr, 1)
		}
		return
	}
	atomic.StoreUint32(ptr, 0)
	if falsePtr != nil {
		atomic.AddUint32(falsePtr, 1)
	}
}

func (t *Reporter) Run(_ context.Context) error {
	defer atomic.AddUint32(&t.runningToggles, 1)
	boop(&t.running, true, nil, nil)
	return nil
}

func (t *Reporter) Stop(_ context.Context) error {
	defer atomic.AddUint32(&t.runningToggles, 1)
	boop(&t.running, false, nil, nil)
	return nil
}

func (t *Reporter) SetLiveness(_ context.Context, b bool) {
	defer atomic.AddUint32(&t.liveToggles, 1)
	boop(&t.live, b, &t.liveUp, &t.liveDown)
}

func (t *Reporter) SetReadiness(_ context.Context, b bool) {
	defer atomic.AddUint32(&t.readyToggles, 1)
	boop(&t.ready, b, &t.readyUp, &t.readyDown)
}

func (t *Reporter) SetStartup(_ context.Context, b bool) {
	defer atomic.AddUint32(&t.startupToggles, 1)
	boop(&t.startup, b, &t.startupUp, &t.startupDown)
}

func (t *Reporter) UpdateHealthChecks(_ context.Context, m map[string]*health.CheckResult) {
	defer atomic.AddUint32(&t.hcUpdates, 1)
	defer t.hcMx.Unlock()
	t.hcMx.Lock()
	if t.hc == nil {
		t.hc = make(map[string]*health.CheckResult)
	}

	for k := range m {
		t.hc[k] = m[k]
	}
}
