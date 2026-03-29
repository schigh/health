package std

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/internal/syncmap"
)

// wrapper wraps a checker and its options together.
type wrapper struct {
	opts    health.AddCheckOptions
	checker health.Checker
}

// result helps us keep a tally of the checks.
type result struct {
	cancelLive  bool
	cancelReady bool
}

// Manager is the standard manager for application health checks.
type Manager struct {
	reporters    syncmap.Map[string, health.Reporter]
	checkers     syncmap.Map[string, wrapper]
	checkResults syncmap.Map[string, result]
	checkFunnel  chan *health.CheckResult
	errChan      chan error
	runningPtr   uint32
	livePtr      uint32
	readyPtr     uint32
	startupPtr   uint32
	allChecksRan uint32
	initialReady uint32
	startupDone  uint32

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

	m.initInternals()

	// validate and start subsystems; on failure, reset and return
	if err := m.validateAndStart(ctx); err != nil {
		shouldReset = true
		m.errChan <- err
		return m.errChan
	}

	// this dispatches a goroutine to poll for two signals: a context
	// cancellation (meaning the application is closing) and a health checker
	// response. This goroutine halts when the application is closing.
	go func(h *Manager) {
		for {
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

	m.initStartupState(ctx)

	// set initial liveness
	_ = m.setLive(ctx, true)
	return m.errChan
}

// initInternals initializes logger, channels, and other internal state.
func (m *Manager) initInternals() {
	if m.Logger == nil {
		m.Logger = health.NoOpLogger{}
	}
	if m.checkFunnel == nil {
		// buffered channel to prevent checker goroutines from blocking
		m.checkFunnel = make(chan *health.CheckResult, m.checkers.Size())
	}
	if m.errChan == nil {
		// we use a buffered channel here, so we can push a
		// startup error straight away if we have to
		m.errChan = make(chan error, 1)
	}
}

// validateAndStart ensures at least one checker and reporter exist, then
// starts reporters and dispatches health checks. Returns the first error
// encountered, or nil on success.
func (m *Manager) validateAndStart(ctx context.Context) error {
	if m.checkers.Size() == 0 {
		return fmt.Errorf("%w.manager.std: there are no checkers specified for this manager", health.ErrHealth)
	}
	if m.reporters.Size() == 0 {
		return fmt.Errorf("%w.manager.std: there are no reporters specified for this manager", health.ErrHealth)
	}
	if err := m.dispatchReporters(ctx); err != nil {
		return err
	}
	return m.dispatchHealthChecks(ctx)
}

// initStartupState determines whether any checks affect startup and, if none
// do, marks startup as immediately complete.
func (m *Manager) initStartupState(ctx context.Context) {
	hasStartupChecks := false
	m.checkers.Each(func(_ string, w wrapper) bool {
		if w.opts.AffectsStartup {
			hasStartupChecks = true
			return false
		}
		return true
	})

	if !hasStartupChecks {
		atomic.StoreUint32(&m.startupDone, 1)
		m.setStartup(ctx, true)
	}
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
			errs = append(errs, fmt.Errorf("reporter '%s' failed to stop: %w", key, rErr))
		}
		return true
	})

	if len(errs) > 0 {
		return fmt.Errorf("%w.manager.std: %w", health.ErrHealth, errors.Join(errs...))
	}

	return nil
}

func (m *Manager) AddReporter(name string, r health.Reporter) error {
	if m.running() {
		return fmt.Errorf("%w.manager.std: cannot add a reporter to a running health instance", health.ErrHealth)
	}

	m.reporters.Set(name, r)
	return nil
}

// setLive sets liveness and notifies reporters if it changed.
func (m *Manager) setLive(ctx context.Context, b bool) bool {
	var v uint32
	if b {
		v = 1
	}

	old := atomic.SwapUint32(&m.livePtr, v)
	changed := old != v

	if changed {
		live := v == 1
		m.reporters.Each(func(_ string, value health.Reporter) bool {
			value.SetLiveness(ctx, live)
			return true
		})
	}

	return changed
}

// setReady sets readiness and notifies reporters if it changed.
func (m *Manager) setReady(ctx context.Context, b bool) bool {
	// we can only be ready after all health checks have reported in
	if b && atomic.LoadUint32(&m.allChecksRan) == 0 {
		return false
	}
	// we can only be ready if startup is complete
	if b && atomic.LoadUint32(&m.startupDone) == 0 {
		return false
	}

	var v uint32
	if b {
		v = 1
	}

	old := atomic.SwapUint32(&m.readyPtr, v)
	changed := old != v
	if changed {
		ready := v == 1
		m.reporters.Each(func(_ string, reporter health.Reporter) bool {
			reporter.SetReadiness(ctx, ready)
			return true
		})
	}

	return changed
}

// setStartup sets startup and notifies reporters if it changed.
func (m *Manager) setStartup(ctx context.Context, b bool) bool {
	var v uint32
	if b {
		v = 1
	}

	old := atomic.SwapUint32(&m.startupPtr, v)
	changed := old != v
	if changed {
		startup := v == 1
		m.reporters.Each(func(_ string, reporter health.Reporter) bool {
			reporter.SetStartup(ctx, startup)
			return true
		})
	}

	return changed
}

// process a health check result received from a checker.
func (m *Manager) processHealthCheck(ctx context.Context, hc *health.CheckResult) {
	if hc == nil {
		m.Logger.Error("received nil health check result")
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	var r result
	defer func(m *Manager, hc *health.CheckResult, r *result) {
		m.checkResults.Set(hc.Name, *r)

		// check if all registered checks have reported in
		if atomic.LoadUint32(&m.allChecksRan) == 0 {
			if m.checkResults.Size() == m.checkers.Size() {
				atomic.StoreUint32(&m.allChecksRan, 1)
			}
		}
	}(m, hc, &r)

	if hc.Error != nil {
		m.Logger.Error("health check returned an error", "check", hc.Name, "error", hc.Error)
	}

	switch hc.Status {
	case health.StatusUnhealthy:
		switch {
		case hc.AffectsLiveness:
			r.cancelReady = true
			r.cancelLive = true
		case hc.AffectsReadiness:
			r.cancelReady = true
		}
	case health.StatusDegraded:
		// degraded checks are reported but do not fail probes
		m.Logger.Warn("health check degraded", "check", hc.Name)
	}

	// relay check result to reporters
	m.reporters.Each(func(_ string, value health.Reporter) bool {
		value.UpdateHealthChecks(ctx, map[string]*health.CheckResult{
			hc.Name: hc,
		})
		return true
	})
}

// start up reporters.
func (m *Manager) dispatchReporters(ctx context.Context) error {
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

// applyCheckOptions overrides check result fields from the registered options.
func applyCheckOptions(hc *health.CheckResult, name string, opts *health.AddCheckOptions) {
	hc.Name = name
	hc.AffectsLiveness = opts.AffectsLiveness
	hc.AffectsReadiness = opts.AffectsReadiness
	hc.AffectsStartup = opts.AffectsStartup
	hc.Group = opts.Group
	hc.ComponentType = opts.ComponentType
	hc.DependsOn = opts.DependsOn
}

// dispatchIntervalCheck dispatches health checks at a regular interval.
func (m *Manager) dispatchIntervalCheck(ctx context.Context, name string, w *wrapper) {
	if w.opts.Frequency&health.CheckAfter == health.CheckAfter {
		m.Logger.Debug("delaying checker", "checker", name, "delay", w.opts.Delay)
		time.Sleep(w.opts.Delay)
	}

	t := time.NewTicker(w.opts.Interval)
	m.Logger.Debug("running interval checker", "checker", name, "interval", w.opts.Interval)
	for {
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			hc := m.safeCheck(ctx, name, w)
			if hc == nil {
				continue
			}
			applyCheckOptions(hc, name, &w.opts)
			m.checkFunnel <- hc
		}
	}
}

// dispatchOneTimeCheck dispatches a one-time health check.
func (m *Manager) dispatchOneTimeCheck(ctx context.Context, name string, w *wrapper) {
	if w.opts.Frequency&health.CheckAfter == health.CheckAfter {
		m.Logger.Debug("delaying checker", "checker", name, "delay", w.opts.Delay)
		time.Sleep(w.opts.Delay)
	}

	m.Logger.Debug("running one-time checker", "checker", name)
	select {
	case <-ctx.Done():
		return
	default:
		hc := m.safeCheck(ctx, name, w)
		if hc == nil {
			return
		}
		applyCheckOptions(hc, name, &w.opts)
		m.checkFunnel <- hc
	}
}

// evaluateFitness evaluates all check results and updates liveness, readiness,
// and startup state. It is run after every health check result is processed.
func (m *Manager) evaluateFitness(ctx context.Context) {
	// don't evaluate if the manager has been stopped
	if !m.running() {
		return
	}

	// only do this if all checks have been performed at least once
	if atomic.LoadUint32(&m.allChecksRan) == 0 {
		return
	}

	// evaluate startup checks first; bail if startup is still pending
	if !m.evaluateStartup(ctx) {
		return
	}

	actuallyLive, actuallyReady := m.aggregateResults()

	reportedReadiness := m.isReady()

	// at this point, all checkers have checked in - if they all report ok for
	// readiness, and this is the first evaluation, we need to flip the ready
	// switch. the block below guarded by the atomic will only ever execute once
	if actuallyReady {
		if atomic.CompareAndSwapUint32(&m.initialReady, 0, 1) {
			reportedReadiness = true
			_ = m.setReady(ctx, true)
		}
	}

	// cant be ready if you aren't live
	actuallyReady = actuallyReady && actuallyLive

	m.applyLivenessChange(ctx, actuallyLive)
	m.applyReadinessChange(ctx, reportedReadiness, actuallyReady)
}

// evaluateStartup checks whether all startup-affecting checks have passed.
// Returns true if startup is complete (either already done or just completed),
// false if startup is still pending.
func (m *Manager) evaluateStartup(ctx context.Context) bool {
	if atomic.LoadUint32(&m.startupDone) != 0 {
		return true
	}

	startupPassed := true
	m.checkers.Each(func(name string, w wrapper) bool {
		if !w.opts.AffectsStartup {
			return true
		}
		r, ok := m.checkResults.Get(name)
		if !ok || r.cancelLive || r.cancelReady {
			startupPassed = false
			return false
		}
		return true
	})

	if !startupPassed {
		return false
	}

	atomic.StoreUint32(&m.startupDone, 1)
	m.setStartup(ctx, true)
	m.Logger.Info("startup complete")
	return true
}

// aggregateResults iterates all check results and returns the aggregate
// liveness and readiness signals.
func (m *Manager) aggregateResults() (live, ready bool) {
	live, ready = true, true
	m.checkResults.Each(func(_ string, value result) bool {
		live = live && !value.cancelLive
		ready = ready && !value.cancelReady
		return true
	})
	return live, ready
}

// applyLivenessChange updates liveness if it has changed and logs the transition.
func (m *Manager) applyLivenessChange(ctx context.Context, actuallyLive bool) {
	if m.isLive() == actuallyLive {
		return
	}
	if !actuallyLive {
		m.Logger.Error("liveness set false")
	} else {
		m.Logger.Info("liveness set true")
	}
	_ = m.setLive(ctx, actuallyLive)
}

// applyReadinessChange updates readiness if it has changed and logs the transition.
func (m *Manager) applyReadinessChange(ctx context.Context, reportedReadiness, actuallyReady bool) {
	if reportedReadiness == actuallyReady {
		return
	}
	if !actuallyReady {
		m.Logger.Warn("readiness set false")
	} else {
		m.Logger.Info("readiness set true")
	}
	_ = m.setReady(ctx, actuallyReady)
}

// safeCheck runs a checker with panic recovery. Returns nil if the checker
// panics or returns nil.
func (m *Manager) safeCheck(ctx context.Context, name string, w *wrapper) (result *health.CheckResult) {
	defer func() {
		if r := recover(); r != nil {
			m.Logger.Error("checker panicked", "checker", name, "panic", r)
			result = &health.CheckResult{
				Name:      name,
				Status:    health.StatusUnhealthy,
				Error:     fmt.Errorf("checker panicked: %v", r),
				Timestamp: time.Now(),
			}
		}
	}()

	hc := w.checker.Check(ctx)
	if hc == nil {
		m.Logger.Error("checker returned nil result", "checker", name)
		return nil
	}
	return hc
}
