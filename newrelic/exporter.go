// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package newrelic contains an OpenTelemetry tracing exporter for New Relic.
package newrelic

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/sdk/export/trace"
)

const (
	version          = "0.1.0"
	userAgentProduct = "NewRelic-Go-OpenTelemetry"

	errorCodeAttrKey    = "error.code"
	errorMessageAttrKey = "error.message"

	instrumentationProviderAttrKey   = "instrumentation.provider"
	instrumentationProviderAttrValue = "opentelemetry"

	collectorNameAttrKey   = "collector.name"
	collectorNameAttrValue = "newrelic-opentelemetry-exporter"
)

// Java implementation:
// https://github.com/newrelic/newrelic-opentelemetry-java-exporters/tree/master/src/main/java/com/newrelic/telemetry/opentelemetry/export

// Exporter exports spans to New Relic.
type Exporter struct {
	harvester *telemetry.Harvester
	// serviceName is the name of this service or application.
	serviceName string
}

var (
	errServiceNameEmpty = errors.New("service name is required")
)

// NewExporter creates a new Exporter that exports spans to New Relic.
func NewExporter(serviceName, apiKey string, options ...func(*telemetry.Config)) (*Exporter, error) {
	if serviceName == "" {
		return nil, errServiceNameEmpty
	}
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
	return &Exporter{
		harvester:   h,
		serviceName: serviceName,
	}, nil
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
	if nil == e {
		return
	}
	e.harvester.RecordSpan(e.transformSpan(span))
}

func (e *Exporter) responseCodeIsError(code uint32) bool {
	if code == 0 {
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
	isError := e.responseCodeIsError(uint32(span.Status))
	numAttrs := len(span.Attributes) + 2
	if isError {
		numAttrs += 2
	}
	if 0 == numAttrs {
		return nil
	}
	attrs := make(map[string]interface{}, numAttrs)
	for _, pair := range span.Attributes {
		attrs[string(pair.Key)] = pair.Value.AsInterface()
	}
	attrs[instrumentationProviderAttrKey] = instrumentationProviderAttrValue
	attrs[collectorNameAttrKey] = collectorNameAttrValue
	if isError {
		attrs[errorCodeAttrKey] = uint32(span.Status)
		attrs[errorMessageAttrKey] = span.Status.String()
	}
	return attrs
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
