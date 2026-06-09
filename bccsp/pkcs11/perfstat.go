/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pkcs11

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Perf instrumentation (cbdc-perf branch): emit PKCS11 session-slot and sign
// timings into the SAME OTel histogram the cbdc-biz view-phase recorder uses
// (metric cbdc.view.phase.duration, scope cbdc-biz.views, attribute "phase").
// Running inside the institution process this hits the already-configured
// global MeterProvider, so the phases export to the metrics pipeline and show
// up alongside the view phases — no log scraping. The scope/metric/attr/bucket
// values must stay in lockstep with token-sdk's utils/phase recorder.
const (
	p11Scope        = "cbdc-biz.views"
	p11DurationName = "cbdc.view.phase.duration"
	p11AttrPhase    = "phase"
)

var p11Buckets = []float64{
	0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

var (
	p11InitOnce sync.Once
	p11Hist     metric.Float64Histogram
)

func p11EnsureInit() {
	p11InitOnce.Do(func() {
		h, err := otel.Meter(p11Scope).Float64Histogram(
			p11DurationName,
			metric.WithUnit("s"),
			metric.WithDescription("Per-phase latency for view operations"),
			metric.WithExplicitBucketBoundaries(p11Buckets...),
		)
		if err != nil {
			return
		}
		p11Hist = h
	})
}

// recordP11Phase records the elapsed time since start under the given phase
// name into the shared view-phase histogram.
func recordP11Phase(phase string, start time.Time) {
	p11EnsureInit()
	if p11Hist == nil {
		return
	}
	p11Hist.Record(context.Background(), time.Since(start).Seconds(), metric.WithAttributes(attribute.String(p11AttrPhase, phase)))
}
