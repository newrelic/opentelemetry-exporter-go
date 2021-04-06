// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package newrelic provides an OpenTelemetry exporter for New Relic.
package newrelic

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/newrelic/opentelemetry-exporter-go/newrelic/internal/transform"
	exportmetric "go.opentelemetry.io/otel/sdk/export/metric"
	exporttrace "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	version          = "0.18.0"
	userAgentProduct = "NewRelic-Go-OpenTelemetry"
)

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
// New Relic Exporter configured with default setting. It is the caller's
// responsibility to stop the returned OTel Controller. This function uses the
// following environment variables to configure the exporter installed in the
// pipeline:
//
//    * `NEW_RELIC_API_KEY`: New Relic Event API key.
//    * `NEW_RELIC_METRIC_URL`: Override URL to New Relic metric endpoint.
//    * `NEW_RELIC_TRACE_URL`: Override URL to New Relic trace endpoint.
//
// More information about the New Relic Event API key can be found
// here: https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys#event-insert-key.
//
// The exporter will send telemetry to the default New Relic metric and trace
// API endpoints in the United States. These can be overwritten with the above
// environment variables. These are useful if you wish to send to our EU
// endpoints:
//
//    * EU metric API endpoint: metric-api.eu.newrelic.com/metric/v1
//    * EU trace API endpoint: trace-api.eu.newrelic.com/trace/v1
func NewExportPipeline(service string, traceOpt []sdktrace.TracerProviderOption, cOpt []controller.Option) (trace.TracerProvider, *controller.Controller, error) {
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
	r := resource.NewWithAttributes(semconv.ServiceNameKey.String(service))

	tp := sdktrace.NewTracerProvider(
		append([]sdktrace.TracerProviderOption{
			sdktrace.WithSyncer(exporter),
			sdktrace.WithResource(r),
		},
			traceOpt...)...,
	)

	pusher := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			exporter,
		),
		append([]controller.Option{controller.WithResource(r)}, cOpt...)...,
	)
	pusher.Start(context.TODO())

	return tp, pusher, nil
}

// InstallNewPipeline installs a New Relic exporter with default settings
// in the global OpenTelemetry telemetry pipeline. It is the caller's
// responsibility to stop the returned push Controller.
// ## Prerequisites
// For details, check out the "Get Started" section of [New Relic Go OpenTelemetry exporter](https://github.com/newrelic/opentelemetry-exporter-go/blob/master/README.md#get-started).
// ## Environment variables
// This function uses the following environment variables to configure
// the exporter installed in the pipeline:
//    * `NEW_RELIC_API_KEY`: New Relic Insights insert key.
//    * `NEW_RELIC_METRIC_URL`: Override URL to New Relic metric endpoint.
//    * `NEW_RELIC_TRACE_URL`: Override URL to New Relic trace endpoint.
// The exporter will send telemetry to the default New Relic metric and trace
// API endpoints in the United States:
// * Traces: https://trace-api.newrelic.com/trace/v1
// * Metrics: https://metric-api.newrelic.com/metric/v1
// You can overwrite these with the above environment variables
// to send data to our EU endpoints or to set up Infinite Tracing.
// For information about changing endpoints, see [OpenTelemetry: Advanced configuration](https://docs.newrelic.com/docs/integrations/open-source-telemetry-integrations/opentelemetry/opentelemetry-advanced-configuration#h2-change-endpoints).

func InstallNewPipeline(service string) (*controller.Controller, error) {
	tp, controller, err := NewExportPipeline(service, nil, nil)
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(tp)
	global.SetMeterProvider(controller.MeterProvider())
	return controller, nil
}

var (
	_ exporttrace.SpanExporter = (*Exporter)(nil)
	_ exportmetric.Exporter    = (*Exporter)(nil)
)

// ExportSpans exports span data to New Relic.
func (e *Exporter) ExportSpans(ctx context.Context, spans []*exporttrace.SpanSnapshot) error {
	if nil == e {
		return nil
	}

	var errs []string
	for _, s := range spans {
		if err := e.harvester.RecordSpan(transform.Span(e.serviceName, s)); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("export span: %s", strings.Join(errs, ", "))
	}
	return nil
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
	return exportmetric.DeltaExportKind
}

func (e *Exporter) Shutdown(ctx context.Context) error {
	e.harvester.HarvestNow(ctx)
	return nil
}
