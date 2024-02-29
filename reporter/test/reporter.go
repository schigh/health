package test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"

	"google.golang.org/protobuf/encoding/protojson"

	healthpb "github.com/schigh/health/pkg/v1"

	"github.com/schigh/health"
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
	hcUpdates      uint32       // number of times health checks have been updated
	//nolint:unused
	hcFetches uint32 // number of times health checks have been fetched

	liveUp    uint32                     // number of times liveness toggled true
	liveDown  uint32                     // number of times liveness toggled false
	readyUp   uint32                     // number of times readiness toggled true
	readyDown uint32                     // number of times readiness toggled false
	hc        map[string]*healthpb.Check // internal map of health checks
}

// Report is a snapshot of this reporter's state.
type Report struct {
	IsRunning                bool                       `json:"-"`
	NumRunningStateChanges   uint32                     `json:"-"`
	IsLive                   bool                       `json:"-"`
	NumLivenessStateChanges  uint32                     `json:"-"`
	NumLivenessSetTrue       uint32                     `json:"-"`
	NumLivenessSetFalse      uint32                     `json:"-"`
	IsReady                  bool                       `json:"-"`
	NumReadinessStateChanges uint32                     `json:"-"`
	NumReadinessSetTrue      uint32                     `json:"-"`
	NumReadinessSetFalse     uint32                     `json:"-"`
	NumHealthCheckUpdates    uint32                     `json:"-"`
	HealthChecks             map[string]*healthpb.Check `json:"-"`
}

func (r Report) MarshalJSON() ([]byte, error) {
	type alias struct {
		IsRunning                bool                       `json:"isRunning"`
		NumRunningStateChanges   uint32                     `json:"numRunningStateChanges"`
		IsLive                   bool                       `json:"isLive"`
		NumLivenessStateChanges  uint32                     `json:"numLivenessStateChanges"`
		NumLivenessSetTrue       uint32                     `json:"numLivenessSetTrue"`
		NumLivenessSetFalse      uint32                     `json:"numLivenessSetFalse"`
		IsReady                  bool                       `json:"isReady"`
		NumReadinessStateChanges uint32                     `json:"numReadinessStateChanges"`
		NumReadinessSetTrue      uint32                     `json:"numReadinessSetTrue"`
		NumReadinessSetFalse     uint32                     `json:"numReadinessSetFalse"`
		NumHealthCheckUpdates    uint32                     `json:"numHealthCheckUpdates"`
		HealthChecks             map[string]json.RawMessage `json:"healthChecks"`
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
	}

	hcs := make(map[string]json.RawMessage)
	for k := range r.HealthChecks {
		hc := r.HealthChecks[k]
		data, err := protojson.Marshal(hc)
		if err != nil {
			return nil, err
		}
		hcs[k] = data
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

	healthChecks := make(map[string]*healthpb.Check)
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

func (t *Reporter) UpdateHealthChecks(_ context.Context, m map[string]*healthpb.Check) {
	defer atomic.AddUint32(&t.hcUpdates, 1)
	defer t.hcMx.Unlock()
	t.hcMx.Lock()
	if t.hc == nil {
		t.hc = make(map[string]*healthpb.Check)
	}

	for k := range m {
		t.hc[k] = m[k]
	}
}
