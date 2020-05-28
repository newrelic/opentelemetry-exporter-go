// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	metricapi "go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	integrator "go.opentelemetry.io/otel/sdk/metric/integrator/simple"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestServiceNameMissing(t *testing.T) {
	e, err := NewExporter("", "apiKey")
	if e != nil {
		t.Error(e)
	}
	if err != errServiceNameEmpty {
		t.Error(err)
	}
}

func TestNilExporter(t *testing.T) {
	span := &trace.SpanData{}
	var e *Exporter

	e.ExportSpan(context.Background(), span)
	e.ExportSpans(context.Background(), []*trace.SpanData{span})
}

// MockTransport caches decompressed request bodies
type MockTransport struct {
	Data []Data
}

func (c *MockTransport) Spans() []Span {
	var spans []Span
	for _, data := range c.Data {
		spans = append(spans, data.Spans...)
	}
	return spans
}

func (c *MockTransport) Metrics() []Metric {
	var metrics []Metric
	for _, data := range c.Data {
		metrics = append(metrics, data.Metrics...)
	}
	return metrics
}

func (c *MockTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// telemetry sdk gzip compresses json payloads
	gz, err := gzip.NewReader(r.Body)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	contents, err := ioutil.ReadAll(gz)
	if err != nil {
		return nil, err
	}

	if !json.Valid(contents) {
		return nil, errors.New("error validating request body json")
	}
	err = c.ParseRequest(contents)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(&bytes.Buffer{}),
	}, nil
}

func (c *MockTransport) ParseRequest(b []byte) error {
	var data []Data
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	c.Data = append(c.Data, data...)
	return nil
}

type Data struct {
	Common  Common                 `json:"common"`
	Spans   []Span                 `json:"spans"`
	Metrics []Metric               `json:"metrics"`
	XXX     map[string]interface{} `json:"-"`
}

type Common struct {
	timestamp  interface{}       `json:"-"`
	interval   interface{}       `json:"-"`
	Attributes map[string]string `json:"attributes"`
}

type Span struct {
	ID         string                 `json:"id"`
	TraceID    string                 `json:"trace.id"`
	Attributes map[string]interface{} `json:"attributes"`
	timestamp  interface{}            `json:"-"`
}

type Metric struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Value      interface{}            `json:"value"`
	timestamp  interface{}            `json:"-"`
	Attributes map[string]interface{} `json:"attributes"`
}

func TestEndToEndTracer(t *testing.T) {
	numSpans := 4
	serviceName := "opentelemetry-service"
	mockt := &MockTransport{
		Data: make([]Data, 0, numSpans),
	}
	e, err := NewExporter(
		serviceName,
		"apiKey",
		telemetry.ConfigHarvestPeriod(0),
		telemetry.ConfigBasicErrorLogger(os.Stderr),
		telemetry.ConfigBasicDebugLogger(os.Stderr),
		telemetry.ConfigBasicAuditLogger(os.Stderr),
		func(cfg *telemetry.Config) {
			cfg.MetricsURLOverride = "localhost"
			cfg.SpansURLOverride = "localhost"
			cfg.Client.Transport = mockt
		},
	)
	if err != nil {
		t.Fatalf("failed to instantiate exporter: %v", err)
	}

	traceProvider, err := sdktrace.NewProvider(
		sdktrace.WithBatcher(e, sdktrace.WithScheduleDelayMillis(15), sdktrace.WithMaxExportBatchSize(10)),
	)
	if err != nil {
		t.Fatalf("failed to instantiate trace provider: %v", err)
	}

	tracer := traceProvider.Tracer("test-tracer")

	var decend func(context.Context, int)
	decend = func(ctx context.Context, n int) {
		if n <= 0 {
			return
		}
		depth := numSpans - n
		ctx, span := tracer.Start(ctx, fmt.Sprintf("Span %d", depth))
		span.SetAttributes(kv.Key("depth").Int(depth))
		decend(ctx, n-1)
		span.End()
	}
	decend(context.Background(), numSpans)

	// Wait >2 cycles.
	<-time.After(40 * time.Millisecond)
	e.harvester.HarvestNow(context.Background())

	gotSpans := mockt.Spans()
	if got := len(gotSpans); got != numSpans {
		t.Fatalf("expecting %d spans, got %d", numSpans, got)
	}

	var traceID, parentID string
	// Reverse order to start at the beginning of the trace.
	for i := len(gotSpans) - 1; i >= 0; i-- {
		depth := numSpans - i - 1
		s := gotSpans[i]
		name := s.Attributes["name"]
		if traceID != "" {
			if got := s.TraceID; got != traceID {
				t.Errorf("span trace ID for %s: got %q, want %q", name, got, traceID)
			}
			if got := s.Attributes["parent.id"]; got != parentID {
				t.Errorf("span parent ID for %s: got %q, want %q", name, got, parentID)
			}
			parentID = s.ID
		} else {
			traceID = s.TraceID
			parentID = s.ID
		}
		if got, want := name, fmt.Sprintf("Span %d", depth); got != want {
			t.Errorf("span name: got %q, want %q", got, want)
		}
		if got := s.Attributes["service.name"]; got != serviceName {
			t.Errorf("span service name for %s: got %q, want %q", name, got, serviceName)
		}
		if got := s.Attributes["depth"].(float64); got != float64(depth) {
			t.Errorf("span 'depth' for %s: got %g, want %d", name, got, depth)
		}
	}
}

func TestEndToEndMeter(t *testing.T) {
	serviceName := "opentelemetry-service"
	type data struct {
		iKind metric.Kind
		nKind metric.NumberKind
		val   int64
	}
	instruments := map[string]data{
		"test-int64-counter":    {metric.CounterKind, metric.Int64NumberKind, 1},
		"test-float64-counter":  {metric.CounterKind, metric.Float64NumberKind, 1},
		"test-int64-measure":    {metric.MeasureKind, metric.Int64NumberKind, 2},
		"test-float64-measure":  {metric.MeasureKind, metric.Float64NumberKind, 2},
		"test-int64-observer":   {metric.ObserverKind, metric.Int64NumberKind, 3},
		"test-float64-observer": {metric.ObserverKind, metric.Float64NumberKind, 3},
	}

	mockt := &MockTransport{
		Data: make([]Data, 0, len(instruments)),
	}
	e, err := NewExporter(
		serviceName,
		"apiKey",
		telemetry.ConfigHarvestPeriod(0),
		telemetry.ConfigBasicErrorLogger(os.Stderr),
		telemetry.ConfigBasicDebugLogger(os.Stderr),
		telemetry.ConfigBasicAuditLogger(os.Stderr),
		func(cfg *telemetry.Config) {
			cfg.MetricsURLOverride = "localhost"
			cfg.SpansURLOverride = "localhost"
			cfg.Client.Transport = mockt
		},
	)
	if err != nil {
		t.Fatalf("failed to instantiate exporter: %v", err)
	}

	aggSelector := selector.NewWithExactMeasure()
	batcher := integrator.New(aggSelector, true)
	pusher := push.New(batcher, e, 60*time.Second)
	pusher.Start()

	ctx := context.Background()
	meter := pusher.Meter("test-meter")

	for name, data := range instruments {
		switch data.iKind {
		case metric.CounterKind:
			switch data.nKind {
			case metric.Int64NumberKind:
				metricapi.Must(meter).NewInt64Counter(name).Add(ctx, data.val)
			case metric.Float64NumberKind:
				metricapi.Must(meter).NewFloat64Counter(name).Add(ctx, float64(data.val))
			default:
				t.Fatal("unsupported number testing kind", data.nKind.String())
			}
		case metric.MeasureKind:
			switch data.nKind {
			case metric.Int64NumberKind:
				metricapi.Must(meter).NewInt64Measure(name).Record(ctx, data.val)
			case metric.Float64NumberKind:
				metricapi.Must(meter).NewFloat64Measure(name).Record(ctx, float64(data.val))
			default:
				t.Fatal("unsupported number testing kind", data.nKind.String())
			}
		case metric.ObserverKind:
			switch data.nKind {
			case metric.Int64NumberKind:
				callback := func(v int64) metricapi.Int64ObserverCallback {
					return metricapi.Int64ObserverCallback(func(result metricapi.Int64ObserverResult) { result.Observe(v) })
				}(data.val)
				metricapi.Must(meter).RegisterInt64Observer(name, callback)
			case metric.Float64NumberKind:
				callback := func(v float64) metricapi.Float64ObserverCallback {
					return metricapi.Float64ObserverCallback(func(result metricapi.Float64ObserverResult) { result.Observe(v) })
				}(float64(data.val))
				metricapi.Must(meter).RegisterFloat64Observer(name, callback)
			default:
				t.Fatal("unsupported number testing kind", data.nKind.String())
			}
		default:
			t.Fatal("unsupported metrics testing kind", data.iKind.String())
		}
	}

	// Wait >2 cycles.
	<-time.After(40 * time.Millisecond)

	// Flush and close.
	pusher.Stop()
	e.harvester.HarvestNow(ctx)

	gotMetrics := mockt.Metrics()
	if got, want := len(gotMetrics), len(instruments); got != want {
		t.Fatalf("expecting %d spans, got %d", want, got)
	}
	seen := make(map[string]struct{}, len(instruments))
	for _, m := range gotMetrics {
		want, ok := instruments[m.Name]
		if !ok {
			t.Fatal("unknown metrics", m.Name)
			continue
		}
		seen[m.Name] = struct{}{}

		switch want.iKind {
		case metric.CounterKind:
			if m.Type != "count" {
				t.Errorf("metric type for %s: got %q, want \"counter\"", m.Name, m.Type)
			}
			if got := m.Value.(float64); got != float64(want.val) {
				t.Errorf("metric value for %s: got %g, want %d", m.Name, m.Value, want.val)
			}
		case metric.MeasureKind, metric.ObserverKind:
			if m.Type != "summary" {
				t.Errorf("metric type for %s: got %q, want \"summary\"", m.Name, m.Type)
			}
			value := m.Value.(map[string]interface{})
			if got := value["count"].(float64); got != 1 {
				t.Errorf("metric value for %s: got %g, want %d", m.Name, m.Value, 1)
			}
			if got := value["sum"].(float64); got != float64(want.val) {
				t.Errorf("metric value for %s: got %g, want %d", m.Name, m.Value, want.val)
			}
			if got := value["min"].(float64); got != float64(want.val) {
				t.Errorf("metric value for %s: got %g, want %d", m.Name, m.Value, want.val)
			}
			if got := value["max"].(float64); got != float64(want.val) {
				t.Errorf("metric value for %s: got %g, want %d", m.Name, m.Value, want.val)
			}
		}
	}

	for i := range instruments {
		if _, ok := seen[i]; !ok {
			t.Errorf("no metric(s) exported for %q", i)
		}
	}
}
