// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package newrelic contains an OpenTelemetry tracing exporter for New Relic.
package newrelic

import (
	"context"
	"errors"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/newrelic/opentelemetry-exporter-go/newrelic/internal/transform"
	metricsdk "go.opentelemetry.io/otel/sdk/export/metric"
	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
)

const (
	version          = "0.1.0"
	userAgentProduct = "NewRelic-Go-OpenTelemetry"
)

// Java implementation:
// https://github.com/newrelic/newrelic-opentelemetry-java-exporters/tree/master/src/main/java/com/newrelic/telemetry/opentelemetry/export

// Exporter exports OpenTelemetry data to New Relic.
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
	_ tracesdk.SpanSyncer  = (*Exporter)(nil)
	_ tracesdk.SpanBatcher = (*Exporter)(nil)
	_ metricsdk.Exporter   = (*Exporter)(nil)
)

// ExportSpans exports multiple spans to New Relic.
func (e *Exporter) ExportSpans(ctx context.Context, spans []*tracesdk.SpanData) {
	for _, s := range spans {
		e.ExportSpan(ctx, s)
	}
}

// ExportSpan exports a span to New Relic.
func (e *Exporter) ExportSpan(ctx context.Context, span *tracesdk.SpanData) {
	if nil == e {
		return
	}
	e.harvester.RecordSpan(transform.Span(e.serviceName, span))
}

// Export exports metrics to New Relic.
func (e *Exporter) Export(_ context.Context, cps metricsdk.CheckpointSet) error {
	return cps.ForEach(func(record metricsdk.Record) error {
		m, err := transform.Record(e.serviceName, record)
		if err != nil {
			return err
		}
		e.harvester.RecordMetric(m)
		return nil
	})
}
