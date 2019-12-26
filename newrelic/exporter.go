// Package newrelic contains an OpenTelemetry tracing exporter for New Relic.
package newrelic

import (
	"context"
	"encoding/hex"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/sdk/export/trace"
)

const (
	version          = "0.1.0"
	userAgentProduct = "NewRelic-Go-OpenTelemetry"
)

// Java implementation:
// https://github.com/newrelic/newrelic-opentelemetry-java-exporters/tree/master/src/main/java/com/newrelic/telemetry/opentelemetry/export

// Exporter exports spans to New Relic.
type Exporter struct {
	harvester *telemetry.Harvester
	// serviceName is the name of this service or application.
	serviceName string
	ignoreCodes map[uint32]struct{}
}

// NewExporter creates a new Exporter that exports spans to New Relic.
func NewExporter(serviceName, apiKey string, options ...func(*telemetry.Config)) (*Exporter, error) {
	options = append([]func(*telemetry.Config){
		func(cfg *telemetry.Config) {
			cfg.Product = userAgentProduct
			cfg.ProductVersion = version
		},
		telemetry.ConfigAPIKey(apiKey),
	}, options...)
	h, err := telemetry.NewHarvester(options...)
	if nil != err {
		return nil, err
	}
	e := &Exporter{
		harvester:   h,
		serviceName: serviceName,
	}
	e.SetIgnoredStatusCodes(5)
	return e, nil
}

// SetIgnoredStatusCodes controls which SpanData.Status
// (https://godoc.org/google.golang.org/grpc/codes) codes are turned into errors
// on Spans.  Any Status greater than zero is considered an error if it is not
// set here.  The default initialization ignores 5 (NOT_FOUND).  NOTE:  This
// list of codes is not mutex protected, and therefore this must be used
// prior to registering the exporter.
func (e *Exporter) SetIgnoredStatusCodes(codes ...uint32) {
	e.ignoreCodes = make(map[uint32]struct{}, len(codes))
	for _, c := range codes {
		e.ignoreCodes[c] = struct{}{}
	}
}

var (
	_ interface {
		trace.SpanSyncer
		trace.SpanBatcher
	} = &Exporter{}
)

// ExportSpans exports multiple spans to New Relic.
func (e *Exporter) ExportSpans(ctx context.Context, spans []*trace.SpanData) {
	for _, s := range spans {
		e.ExportSpan(ctx, s)
	}
}

// ExportSpan exports a span to New Relic.
func (e *Exporter) ExportSpan(ctx context.Context, span *trace.SpanData) {
	e.harvester.RecordSpan(e.transformSpan(span))
}

func (e *Exporter) responseCodeIsError(code uint32) bool {
	if code == 0 {
		return false
	}
	if _, ok := e.ignoreCodes[code]; ok {
		return false
	}
	return true
}

func transformSpanID(id core.SpanID) string {
	if !id.IsValid() {
		return ""
	}
	return hex.EncodeToString(id[:])
}

func (e *Exporter) makeAttributes(span *trace.SpanData) map[string]interface{} {
	attributes := make(map[string]interface{}, len(span.Attributes)+1)
	for _, pair := range span.Attributes {
		attributes[string(pair.Key)] = pair.Value.AsInterface()
	}

	if e.responseCodeIsError(uint32(span.Status)) {
		attributes["error.message"] = span.Status.String()
	}

	return attributes
}

// https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#Span
// https://godoc.org/go.opentelemetry.io/otel/sdk/export/trace#SpanData
func (e *Exporter) transformSpan(span *trace.SpanData) telemetry.Span {
	return telemetry.Span{
		ID:          span.SpanContext.SpanIDString(),
		TraceID:     span.SpanContext.TraceIDString(),
		Timestamp:   span.StartTime,
		Name:        span.Name,
		ParentID:    transformSpanID(span.ParentSpanID),
		Duration:    span.EndTime.Sub(span.StartTime),
		ServiceName: e.serviceName,
		Attributes:  e.makeAttributes(span),
	}
}
