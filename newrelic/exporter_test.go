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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/sdk/export/trace"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
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
	span := &trace.SpanSnapshot{}
	var e *Exporter

	e.ExportSpans(context.Background(), []*trace.SpanSnapshot{span})
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
	timestamp  interface{}
	interval   interface{}
	Attributes map[string]string `json:"attributes"`
}

type Span struct {
	ID         string                 `json:"id"`
	TraceID    string                 `json:"trace.id"`
	Attributes map[string]interface{} `json:"attributes"`
	timestamp  interface{}
}

type Metric struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Value      interface{} `json:"value"`
	timestamp  interface{}
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

	r := resource.NewWithAttributes(semconv.ServiceNameKey.String(serviceName))
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(e, sdktrace.WithBatchTimeout(15), sdktrace.WithMaxExportBatchSize(10)),
		sdktrace.WithResource(r),
	)

	tracer := tracerProvider.Tracer("test-tracer")

	var descend func(context.Context, int)
	descend = func(ctx context.Context, n int) {
		if n <= 0 {
			return
		}
		depth := numSpans - n
		ctx, span := tracer.Start(ctx, fmt.Sprintf("Span %d", depth))
		span.SetAttributes(attribute.Int("depth", depth))
		descend(ctx, n-1)
		span.End()
	}
	descend(context.Background(), numSpans)

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
		iKind metric.InstrumentKind
		nKind number.Kind
		val   int64
	}
	instruments := map[string]data{
		"test-int64-counter":                {metric.CounterInstrumentKind, number.Int64Kind, 1},
		"test-float64-counter":              {metric.CounterInstrumentKind, number.Float64Kind, 1},
		"test-int64-up-down-counter":        {metric.UpDownCounterInstrumentKind, number.Int64Kind, 1},
		"test-float64-up-down-counter":      {metric.UpDownCounterInstrumentKind, number.Float64Kind, 1},
		"test-int64-measure":                {metric.ValueRecorderInstrumentKind, number.Int64Kind, 2},
		"test-float64-measure":              {metric.ValueRecorderInstrumentKind, number.Float64Kind, 2},
		"test-int64-observer":               {metric.ValueObserverInstrumentKind, number.Int64Kind, 3},
		"test-float64-observer":             {metric.ValueObserverInstrumentKind, number.Float64Kind, 3},
		"test-int64-sum-observer":           {metric.SumObserverInstrumentKind, number.Int64Kind, 3},
		"test-float64-sum-observer":         {metric.SumObserverInstrumentKind, number.Float64Kind, 3},
		"test-int64-up-down-sum-observer":   {metric.UpDownSumObserverInstrumentKind, number.Int64Kind, 3},
		"test-float64-up-down-sum-observer": {metric.UpDownSumObserverInstrumentKind, number.Float64Kind, 3},
	}

	mockt := &MockTransport{
		Data: make([]Data, 0, len(instruments)),
	}
	exp, err := NewExporter(
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

	ctx := context.Background()
	control := controller.New(
		processor.New(
			selector.NewWithInexpensiveDistribution(),
			exp, // passed as an ExportKindSelector.
		),
		// Set collection period longer than this test will run for.
		controller.WithCollectPeriod(10*time.Second),
		controller.WithPushTimeout(time.Millisecond),
		controller.WithPusher(exp),
	)

	if err := control.Start(ctx); err != nil {
		t.Fatalf("starting controller: %v", err)
	}

	meter := control.MeterProvider().Meter("test-meter")

	newInt64ObserverCallback := func(v int64) metric.Int64ObserverFunc {
		return func(ctx context.Context, result metric.Int64ObserverResult) { result.Observe(v) }
	}
	newFloat64ObserverCallback := func(v float64) metric.Float64ObserverFunc {
		return func(ctx context.Context, result metric.Float64ObserverResult) { result.Observe(v) }
	}

	for name, data := range instruments {
		switch data.iKind {
		case metric.CounterInstrumentKind:
			switch data.nKind {
			case number.Int64Kind:
				metric.Must(meter).NewInt64Counter(name).Add(ctx, data.val)
			case number.Float64Kind:
				metric.Must(meter).NewFloat64Counter(name).Add(ctx, float64(data.val))
			default:
				t.Fatal("unsupported number testing kind", data.nKind.String())
			}
		case metric.UpDownCounterInstrumentKind:
			switch data.nKind {
			case number.Int64Kind:
				metric.Must(meter).NewInt64UpDownCounter(name).Add(ctx, data.val)
			case number.Float64Kind:
				metric.Must(meter).NewFloat64UpDownCounter(name).Add(ctx, float64(data.val))
			default:
				t.Fatal("unsupported number testing kind", data.nKind.String())
			}
		case metric.ValueRecorderInstrumentKind:
			switch data.nKind {
			case number.Int64Kind:
				metric.Must(meter).NewInt64ValueRecorder(name).Record(ctx, data.val)
			case number.Float64Kind:
				metric.Must(meter).NewFloat64ValueRecorder(name).Record(ctx, float64(data.val))
			default:
				t.Fatal("unsupported number testing kind", data.nKind.String())
			}
		case metric.ValueObserverInstrumentKind:
			switch data.nKind {
			case number.Int64Kind:
				metric.Must(meter).NewInt64ValueObserver(name, newInt64ObserverCallback(data.val))
			case number.Float64Kind:
				metric.Must(meter).NewFloat64ValueObserver(name, newFloat64ObserverCallback(float64(data.val)))
			default:
				t.Fatal("unsupported number testing kind", data.nKind.String())
			}
		case metric.SumObserverInstrumentKind:
			switch data.nKind {
			case number.Int64Kind:
				metric.Must(meter).NewInt64SumObserver(name, newInt64ObserverCallback(data.val))
			case number.Float64Kind:
				metric.Must(meter).NewFloat64SumObserver(name, newFloat64ObserverCallback(float64(data.val)))
			default:
				t.Fatal("unsupported number testing kind", data.nKind.String())
			}
		case metric.UpDownSumObserverInstrumentKind:
			switch data.nKind {
			case number.Int64Kind:
				metric.Must(meter).NewInt64UpDownSumObserver(name, newInt64ObserverCallback(data.val))
			case number.Float64Kind:
				metric.Must(meter).NewFloat64UpDownSumObserver(name, newFloat64ObserverCallback(float64(data.val)))
			default:
				t.Fatal("unsupported number testing kind", data.nKind.String())
			}
		default:
			t.Fatal("unsupported metrics testing kind", data.iKind.String())
		}
	}

	// Flush and stop the conroller.
	if err := control.Stop(ctx); err != nil {
		t.Fatalf("stopping controller: %v", err)
	}

	// Flush and stop the exporter.
	if err := exp.Shutdown(ctx); err != nil {
		t.Fatalf("shutting down exporter: %v", err)
	}

	gotMetrics := mockt.Metrics()
	if got, want := len(gotMetrics), len(instruments); got != want {
		t.Fatalf("expecting %d metrics, got %d", want, got)
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
		case metric.CounterInstrumentKind:
			if m.Type != "count" {
				t.Errorf("metric type for %s: got %q, want \"counter\"", m.Name, m.Type)
				continue
			}
			if got := m.Value.(float64); got != float64(want.val) {
				t.Errorf("metric value for %s: got %g, want %d", m.Name, m.Value, want.val)
			}
		case metric.ValueObserverInstrumentKind:
			if m.Type != "gauge" {
				t.Errorf("metric type for %s: got %q, want \"gauge\"", m.Name, m.Type)
				continue
			}
			if got := m.Value.(float64); got != float64(want.val) {
				t.Errorf("metric value for %s: got %g, want %d", m.Name, m.Value, want.val)
			}
		case metric.ValueRecorderInstrumentKind:
			if m.Type != "summary" {
				t.Errorf("metric type for %s: got %q, want \"summary\"", m.Name, m.Type)
				continue
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
