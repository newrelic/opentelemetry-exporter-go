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
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/sdk/export/trace"
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
		span.SetAttributes(core.Key("depth").Int(depth))
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
