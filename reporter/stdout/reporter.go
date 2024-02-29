package stdout

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"

	healthpb "github.com/schigh/health/pkg/v1"
)

var w = os.Stdout //nolint:gochecknoglobals // global ok

const (
	header   = "╭--------------------------------------╮"
	smHeader = "|╭------------------------------------╮|"
	footer   = "╰--------------------------------------╯"
	smFooter = "|╰------------------------------------╯|"
	hl       = "|--------------------------------------|"
	yes      = "yes"
	no       = "no"
)

// Reporter reports.
type Reporter struct {
	live         uint32
	ready        uint32
	healthChecks *healthCheckMap
}

func (r *Reporter) Run(_ context.Context) error {
	if r.healthChecks == nil {
		r.healthChecks = &healthCheckMap{}
	}

	return nil
}

func (r *Reporter) Stop(_ context.Context) error {
	return nil
}

func (r *Reporter) SetLiveness(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.live, v)
}

func (r *Reporter) SetReadiness(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.ready, v)
}

func (r *Reporter) UpdateHealthChecks(_ context.Context, m map[string]*healthpb.Check) {
	r.healthChecks.AbsorbMap(m)
	r.reportHealthChecks(w)
}

func (r *Reporter) reportLiveness(out io.Writer) {
	live := no
	if atomic.LoadUint32(&r.live) == 1 {
		live = yes
	}
	_, _ = fmt.Fprintf(out, "|live:%33s|\n", live)
}

func (r *Reporter) reportReadiness(out io.Writer) {
	ready := no
	if atomic.LoadUint32(&r.ready) == 1 {
		ready = yes
	}
	_, _ = fmt.Fprintf(out, "|ready:%32s|\n", ready)
}

func (r *Reporter) reportHealthChecks(out io.Writer) {
	_, _ = fmt.Fprintln(os.Stdout, header)
	_, _ = fmt.Fprintln(out, "|Health Status                         |")
	_, _ = fmt.Fprintln(out, hl)
	r.reportLiveness(out)
	r.reportReadiness(out)
	_, _ = fmt.Fprintln(out, hl)
	_, _ = fmt.Fprintln(out, "|Checks                                |")

	checks := r.healthChecks.Value()
	for k := range checks {
		hc := checks[k]
		healthy := yes
		if hc.GetError() != nil {
			healthy = no
		}
		affectsLiveness := no
		affectsReadiness := no
		if hc.GetAffectsLiveness() {
			affectsLiveness = yes
		}
		if hc.GetAffectsReadiness() {
			affectsReadiness = yes
		}
		_, _ = fmt.Fprintln(out, smHeader)
		_, _ = fmt.Fprintf(out, "||name%32s||\n", hc.GetName())
		_, _ = fmt.Fprintf(out, "||healthy%29s||\n", healthy)
		_, _ = fmt.Fprintf(out, "||affects liveness%20s||\n", affectsLiveness)
		_, _ = fmt.Fprintf(out, "||affects readiness%19s||\n", affectsReadiness)
		_, _ = fmt.Fprintln(out, smFooter)
	}

	_, _ = fmt.Fprintln(out, footer)
}
