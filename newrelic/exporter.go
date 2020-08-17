// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package newrelic contains an OpenTelemetry tracing exporter for New Relic.
package newrelic

import (
	"context"
	"errors"
	"os"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/standard"
	apitrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/newrelic/opentelemetry-exporter-go/newrelic/internal/transform"
	exportmetric "go.opentelemetry.io/otel/sdk/export/metric"
	exporttrace "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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

// NewExporter creates a new Exporter that exports telemetry to New Relic.
func NewExporter(service, apiKey string, options ...func(*telemetry.Config)) (*Exporter, error) {
	if service == "" {
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
		serviceName: service,
	}, nil
}

// NewExportPipeline creates a new OpenTelemetry telemetry pipeline using a
// New Relic Exporter configured with default setting. It is the callers
// responsibility to stop the returned push Controller. This function uses the
// following environment variables to configure the exporter installed in the
// pipeline:
//
//    * `NEW_RELIC_API_KEY`: New Relic API key.
//    * `NEW_RELIC_METRIC_URL`: URL to New Relic metric endpoint.
//    * `NEW_RELIC_METRIC_URL`: URL to New Relic trace endpoint.
func NewExportPipeline(service string, traceOpt []sdktrace.ProviderOption, pushOpt []push.Option) (apitrace.Provider, *push.Controller, error) {
	apiKey, ok := os.LookupEnv("NEW_RELIC_API_KEY")
	if !ok {
		return nil, nil, errors.New("missing New Relic API key")
	}

	var eOpts []func(*telemetry.Config)
	if u, ok := os.LookupEnv("NEW_RELIC_METRIC_URL"); ok {
		eOpts = append(eOpts, func(cfg *telemetry.Config) {
			cfg.MetricsURLOverride = u
		})
	}
	if u, ok := os.LookupEnv("NEW_RELIC_TRACE_URL"); ok {
		eOpts = append(eOpts, telemetry.ConfigSpansURLOverride(u))
	}

	exporter, err := NewExporter(service, apiKey, eOpts...)
	if err != nil {
		return nil, nil, err
	}

	// Minimally default resource with a service name. This is overwritten if
	// another is passed in traceOpt or pushOpt.
	r := resource.New(standard.ServiceNameKey.String(service))

	tp, err := sdktrace.NewProvider(
		append([]sdktrace.ProviderOption{
			sdktrace.WithSyncer(exporter),
			sdktrace.WithResource(r),
		},
			traceOpt...)...,
	)
	if err != nil {
		return nil, nil, err
	}

	pusher := push.New(
		basic.New(simple.NewWithExactDistribution(), exporter),
		exporter,
		append([]push.Option{push.WithResource(r)}, pushOpt...)...,
	)
	pusher.Start()

	return tp, pusher, nil
}

// InstallNewPipeline installs a New Relic exporter with default settings
// in the global OpenTelemetry telemetry pipeline. It is the callers
// responsibility to stop the returned push Controller. This function uses the
// following environment variables to configure the exporter installed in the
// pipeline:
//
//    * `NEW_RELIC_API_KEY`: New Relic API key.
//    * `NEW_RELIC_METRIC_URL`: URL to New Relic metric endpoint.
//    * `NEW_RELIC_METRIC_URL`: URL to New Relic trace endpoint.
func InstallNewPipeline(service string) (*push.Controller, error) {
	tp, controller, err := NewExportPipeline(service, nil, nil)
	if err != nil {
		return nil, err
	}

	global.SetTraceProvider(tp)
	global.SetMeterProvider(controller.Provider())
	return controller, nil
}

var (
	_ exporttrace.SpanSyncer  = (*Exporter)(nil)
	_ exporttrace.SpanBatcher = (*Exporter)(nil)
	_ exportmetric.Exporter   = (*Exporter)(nil)
)

// ExportSpans exports multiple spans to New Relic.
func (e *Exporter) ExportSpans(ctx context.Context, spans []*exporttrace.SpanData) {
	for _, s := range spans {
		e.ExportSpan(ctx, s)
	}
}

// ExportSpan exports a span to New Relic.
func (e *Exporter) ExportSpan(ctx context.Context, span *exporttrace.SpanData) {
	if nil == e {
		return
	}
	e.harvester.RecordSpan(transform.Span(e.serviceName, span))
}

// Export exports metrics to New Relic.
func (e *Exporter) Export(_ context.Context, cps exportmetric.CheckpointSet) error {
	return cps.ForEach(e, func(record exportmetric.Record) error {
		m, err := transform.Record(e.serviceName, record)
		if err != nil {
			return err
		}
		e.harvester.RecordMetric(m)
		return nil
	})
}

func (e *Exporter) ExportKindFor(_ *metric.Descriptor, _ aggregation.Kind) exportmetric.ExportKind {
	return exportmetric.DeltaExporter
}
