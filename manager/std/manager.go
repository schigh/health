package std

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/schigh/health"
	healthpb "github.com/schigh/health/pkg/v1"
)

// this wraps a checker and its options together.
type wrapper struct {
	opts    health.AddCheckOptions
	checker health.Checker
}

// this helps us keep a tally of the checks.
type result struct {
	cancelLive  bool
	cancelReady bool
}

// Manager is the standard manager for application health checks.
type Manager struct {
	reporters    *reporterMap
	checkers     *checkerMap
	checkResults *checkResultMap
	checkFunnel  chan *healthpb.Check
	errChan      chan error
	runningPtr   uint32
	livePtr      uint32
	readyPtr     uint32
	allChecksRan uint32
	initialReady uint32

	Logger health.Logger
}

func (m *Manager) isLive() bool {
	return atomic.LoadUint32(&m.livePtr) == 1
}

func (m *Manager) isReady() bool {
	return atomic.LoadUint32(&m.readyPtr) == 1
}

func (m *Manager) running() bool {
	return atomic.LoadUint32(&m.runningPtr) == 1
}

func (m *Manager) AddCheck(name string, checker health.Checker, opts ...health.AddCheckOption) error {
	if m.running() {
		return fmt.Errorf("%w.manager.std: cannot add a health check to a running health instance", health.ErrHealth)
	}

	o := health.AddCheckOptions{}
	for i := range opts {
		opts[i](&o)
	}

	// if no frequency set, check only once
	if o.Frequency == 0 {
		o.Frequency = health.CheckOnce
	}

	if m.checkers == nil {
		m.checkers = &checkerMap{}
	}

	m.checkers.Set(name, wrapper{
		opts:    o,
		checker: checker,
	})

	return nil
}

func (m *Manager) Run(ctx context.Context) <-chan error {
	// if we're already running, get out
	if !atomic.CompareAndSwapUint32(&m.runningPtr, 0, 1) {
		return m.errChan
	}

	// if we get an error while starting up, set running back to false
	var shouldReset bool
	defer func(reset *bool) {
		if *reset {
			atomic.StoreUint32(&m.runningPtr, 0)
		}
	}(&shouldReset)

	// set up internals
	// ---------------------------------------------------
	if m.Logger == nil {
		m.Logger = health.NoOpLogger{}
	}
	if m.checkResults == nil {
		m.checkResults = &checkResultMap{}
	}

	if m.checkFunnel == nil {
		// this relays results from individual check
		m.checkFunnel = make(chan *healthpb.Check)
	}

	if m.errChan == nil {
		// we use a buffered channel here, so we can push a
		// startup error straight away if we have to
		m.errChan = make(chan error, 1)
	}

	// make sure we have at least one checker and one reporters
	if m.checkers == nil || m.checkers.Size() == 0 {
		shouldReset = true
		m.errChan <- fmt.Errorf("%w.manager.std: there are no checkers specified for this manager", health.ErrHealth)
		return m.errChan
	}

	if m.reporters == nil || m.reporters.Size() == 0 {
		shouldReset = true
		m.errChan <- fmt.Errorf("%w.manager.std: there are no reporters specified for this manager", health.ErrHealth)
		return m.errChan
	}

	// start reporters
	if err := m.dispatchReporters(ctx); err != nil {
		shouldReset = true
		m.errChan <- err
		return m.errChan
	}

	// start checkers
	if err := m.dispatchHealthChecks(ctx); err != nil {
		shouldReset = true
		m.errChan <- err
		return m.errChan
	}

	// this dispatches a goroutine to poll for two signals: a context
	// cancellation (meaning the application is closing) and a health checker
	// response. This goroutine halts when the application is closing.
	go func(h *Manager) {
		for {
			// blocks
			select {
			case <-ctx.Done():
				_ = h.Stop(ctx)
				return
			case hc := <-h.checkFunnel:
				h.processHealthCheck(ctx, hc)
				h.evaluateFitness(ctx)
			}
		}
	}(m)

	// set initial liveness
	_ = m.setLive(ctx, true)
	return m.errChan
}

func (m *Manager) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&m.runningPtr, 1, 0) {
		return nil
	}

	_ = m.setReady(ctx, false)
	var errs []error

	m.reporters.Each(func(key string, reporter health.Reporter) bool {
		rErr := reporter.Stop(ctx)
		if rErr != nil && !errors.Is(rErr, context.Canceled) {
			errs = append(errs, fmt.Errorf("%w.manager.std: reporter '%s' failed to stop: %w", health.ErrHealth, key, rErr))
		}
		return true
	})

	if len(errs) > 0 {
		errStrs := make([]string, 0, len(errs))
		for i := range errs {
			errStrs = append(errStrs, errs[i].Error())
		}

		// TODO: come up with a better way to convey 0...N errors at once
		return fmt.Errorf("%w.manager.std: "+strings.Join(errStrs, "\n"), health.ErrHealth)
	}

	return nil
}

func (m *Manager) AddReporter(name string, r health.Reporter) error {
	if m.running() {
		return fmt.Errorf("%w.manager.std: cannot add a reporters to a running health instance", health.ErrHealth)
	}

	if m.reporters == nil {
		m.reporters = &reporterMap{}
	}

	m.reporters.Set(name, r)
	return nil
}

// set liveness.
func (m *Manager) setLive(ctx context.Context, b bool) bool { //nolint:unparam
	var v uint32
	if b {
		v = 1
	}

	old := atomic.SwapUint32(&m.livePtr, v)
	changed := old != v

	// liveness changed state
	if changed {
		live := v == 1
		m.reporters.Each(func(_ string, value health.Reporter) bool {
			value.SetLiveness(ctx, live)
			return true
		})
	}

	return changed
}

// set readiness.
func (m *Manager) setReady(ctx context.Context, b bool) bool { //nolint:unparam
	// we can only be ready after all health checks have reported in
	if b && atomic.LoadUint32(&m.allChecksRan) == 0 {
		return false
	}
	var v uint32
	if b {
		v = 1
	}

	old := atomic.SwapUint32(&m.readyPtr, v)
	changed := old != v
	// readiness changed state
	if changed {
		ready := v == 1
		m.reporters.Each(func(_ string, reporter health.Reporter) bool {
			reporter.SetReadiness(ctx, ready)
			return true
		})
	}

	return changed
}

// process a health check result received from a checker.
func (m *Manager) processHealthCheck(ctx context.Context, hc *healthpb.Check) {
	// TODO: this will block subsequent reads on m.checkFunnel.
	//  We might have to dispatch this func in its own goroutine
	//  if there is a bottleneck here.

	select {
	case <-ctx.Done():
		return
	default:
	}

	var r result
	defer func(m *Manager, hc *healthpb.Check, r *result) {
		m.checkResults.Set(hc.GetName(), *r)

		// this is true when all checks ran at least once
		allChecksRan := atomic.LoadUint32(&m.allChecksRan)
		if allChecksRan == 0 {
			// this will verify that we got at least one
			// result from each registered health checker
			cKeys := m.checkers.Keys()
			ccKeys := m.checkResults.Keys()
			if len(cKeys) == len(ccKeys) {
				sort.Strings(cKeys)
				sort.Strings(ccKeys)
				for i := range cKeys {
					if cKeys[i] != ccKeys[i] {
						// this is kind of a big deal...should never happen
						m.Logger.Error("health.manager.std: key corruption - registered_checks: %v, found checks: %v", cKeys, ccKeys)
						// TODO: not sure how to handle this...technically the manager is corrupted
					}
				}
				atomic.StoreUint32(&m.allChecksRan, 1)
			}
		}
	}(m, hc, &r)

	// the health check result contained an error
	if err := hc.GetError(); err != nil {
		m.Logger.Error("health.managers.std: health check '%s' returned an error: %v", hc, err)
	}

	// the health check returned not healthy
	if !hc.GetHealthy() {
		switch {
		case hc.GetAffectsLiveness():
			r.cancelReady = true
			r.cancelLive = true
		case hc.GetAffectsReadiness():
			r.cancelReady = true
		}
	}

	// relay check result to reporters
	m.reporters.Each(func(_ string, value health.Reporter) bool {
		value.UpdateHealthChecks(ctx, map[string]*healthpb.Check{
			hc.GetName(): hc,
		})
		return true
	})
}

// start up reporters.
func (m *Manager) dispatchReporters(ctx context.Context) error {
	// start reporting
	var outErr error
	m.reporters.Each(func(_ string, reporter health.Reporter) bool {
		if startErr := reporter.Run(ctx); startErr != nil {
			outErr = startErr
			return false
		}
		return true
	})

	return outErr
}

// start up individual checks.
func (m *Manager) dispatchHealthChecks(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.checkers.Each(func(key string, w wrapper) bool {
		if w.opts.Frequency&health.CheckAtInterval == health.CheckAtInterval {
			go m.dispatchIntervalCheck(ctx, key, &w)
			return true
		}
		go m.dispatchOneTimeCheck(ctx, key, &w)
		return true
	})

	return nil
}

// this dispatches health checks at a regular interval.
func (m *Manager) dispatchIntervalCheck(ctx context.Context, name string, w *wrapper) {
	if w.opts.Frequency&health.CheckAfter == health.CheckAfter {
		m.Logger.Debug("health.manager.std: delaying checker - checker: %s, delay: %s", name, w.opts.Delay)
		time.Sleep(w.opts.Delay)
	}

	t := time.NewTicker(w.opts.Interval)
	m.Logger.Debug("health.manager.std: running interval checker - checker: %s, interval: %s", name, w.opts.Interval)
	for {
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			hc := w.checker.Check(ctx)
			// the checker shouldn't set these values, but if
			// it does, they are overridden here
			hc.AffectsLiveness = w.opts.AffectsLiveness
			hc.AffectsReadiness = w.opts.AffectsReadiness
			m.checkFunnel <- hc
		}
	}
}

// this dispatches a one-time health check.
func (m *Manager) dispatchOneTimeCheck(ctx context.Context, name string, w *wrapper) {
	if w.opts.Frequency&health.CheckAfter == health.CheckAfter {
		m.Logger.Debug("health.manager.std: delaying checker - checker: %s, delay: %s", name, w.opts.Delay)
		time.Sleep(w.opts.Delay)
	}

	m.Logger.Debug("health.manager.std: running one-time checker - checker: %s", name)
	select {
	case <-ctx.Done():
		return
	default:
		m.checkFunnel <- w.checker.Check(ctx)
	}
}

// this will evaluate the health check results for liveness and readiness.
// It is run after every health checker result is passed into the work funnel
// for this manager.
func (m *Manager) evaluateFitness(ctx context.Context) {
	// only do this if all checks have been performed at least once
	if atomic.LoadUint32(&m.allChecksRan) == 0 {
		return
	}

	// get current liveness and readiness
	reportedLiveness := m.isLive()
	reportedReadiness := m.isReady()

	// baseline...we will keep and-ing these to the reported liveness
	// and readiness of each check result
	actuallyLive, actuallyReady := true, true

	// go over each check result and check if any results can propagate
	// a liveness or readiness change
	m.checkResults.Each(func(_ string, value result) bool {
		actuallyLive = actuallyLive && !value.cancelLive
		actuallyReady = actuallyReady && !value.cancelReady
		return true
	})

	// at this point, all checkers have checked in - if they all report ok for
	// readiness, and this is the first evaluation, we need to flip the ready
	// switch.  the block below guarded by the atomic will only ever execute once
	if actuallyReady {
		if atomic.CompareAndSwapUint32(&m.initialReady, 0, 1) {
			reportedReadiness = true
			_ = m.setReady(ctx, true)
		}
	}

	// cant be ready if you aren't live
	actuallyReady = actuallyReady && actuallyLive

	// this indicates that liveness or readiness have changed
	livenessChanged := reportedLiveness != actuallyLive
	readinessChanged := reportedReadiness != actuallyReady
	if livenessChanged {
		if !actuallyLive {
			m.Logger.Error("health.manager.std: liveness set false")
		} else {
			m.Logger.Info("health.manager.std: liveness set true")
		}
		_ = m.setLive(ctx, actuallyLive)
	}
	if readinessChanged {
		if !actuallyReady {
			m.Logger.Warn("health.manager.std: readiness set false")
		} else {
			m.Logger.Info("health.manager.std: readiness set true")
		}
		_ = m.setReady(ctx, actuallyReady)
	}
}
