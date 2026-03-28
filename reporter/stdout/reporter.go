package stdout

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/schigh/health"
	"github.com/schigh/health/internal/syncmap"
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

// Reporter reports health status to stdout.
type Reporter struct {
	live         uint32
	ready        uint32
	startup      uint32
	healthChecks syncmap.Map[string, *health.CheckResult]
}

func (r *Reporter) Run(_ context.Context) error {
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

func (r *Reporter) SetStartup(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.startup, v)
}

func (r *Reporter) UpdateHealthChecks(_ context.Context, m map[string]*health.CheckResult) {
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

func (r *Reporter) reportStartup(out io.Writer) {
	startup := no
	if atomic.LoadUint32(&r.startup) == 1 {
		startup = yes
	}
	_, _ = fmt.Fprintf(out, "|startup:%30s|\n", startup)
}

func (r *Reporter) reportHealthChecks(out io.Writer) {
	_, _ = fmt.Fprintln(os.Stdout, header)
	_, _ = fmt.Fprintln(out, "|Health Status                         |")
	_, _ = fmt.Fprintln(out, hl)
	r.reportLiveness(out)
	r.reportReadiness(out)
	r.reportStartup(out)
	_, _ = fmt.Fprintln(out, hl)
	_, _ = fmt.Fprintln(out, "|Checks                                |")

	checks := r.healthChecks.Value()
	for k := range checks {
		hc := checks[k]
		status := hc.Status.String()
		affectsLiveness := no
		affectsReadiness := no
		if hc.AffectsLiveness {
			affectsLiveness = yes
		}
		if hc.AffectsReadiness {
			affectsReadiness = yes
		}
		_, _ = fmt.Fprintln(out, smHeader)
		_, _ = fmt.Fprintf(out, "||name%32s||\n", k)
		_, _ = fmt.Fprintf(out, "||status%30s||\n", status)
		_, _ = fmt.Fprintf(out, "||affects liveness%20s||\n", affectsLiveness)
		_, _ = fmt.Fprintf(out, "||affects readiness%19s||\n", affectsReadiness)
		_, _ = fmt.Fprintln(out, smFooter)
	}

	_, _ = fmt.Fprintln(out, footer)
}
