// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic_test

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/akulnurislam/opentelemetry-exporter-go/newrelic"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
)

func ExampleNewExporter() {
	// To enable Infinite Tracing on the New Relic Edge, use the
	// telemetry.ConfigSpansURLOverride along with the URL for your Trace
	// Observer, including scheme and path.  See
	// https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/enable-configure/enable-distributed-tracing
	exporter, err := newrelic.NewExporter(
		"My Service", os.Getenv("NEW_RELIC_API_KEY"),
		telemetry.ConfigSpansURLOverride("https://nr-internal.aws-us-east-1.tracing.edge.nr-data.net/trace/v1"),
	)
	if err != nil {
		log.Fatal(err)
	}
	otel.SetTracerProvider(
		trace.NewTracerProvider(trace.WithSyncer(exporter)),
	)
}

func ExampleNewExportPipeline() {
	// Include environment in resource.
	r := resource.NewWithAttributes(
		attribute.String("environment", "production"),
		semconv.ServiceNameKey.String("My Service"),
	)

	// Assumes the NEW_RELIC_API_KEY environment variable contains your New
	// Relic Event API key. This will error if it does not.
	traceProvider, controller, err := newrelic.NewExportPipeline(
		"My Service",
		[]trace.TracerProviderOption{
			trace.WithConfig(trace.Config{
				// Conservative sampler.
				DefaultSampler: trace.ParentBased(trace.NeverSample()),
				// Reduce span events.
				SpanLimits: trace.SpanLimits{
					EventCountLimit: 10,
				},
				Resource: r,
			}),
		},
		[]controller.Option{
			// Increase push frequency.
			controller.WithCollectPeriod(time.Second),
			controller.WithResource(r),
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	defer controller.Stop(context.Background())

	otel.SetTracerProvider(traceProvider)
	global.SetMeterProvider(controller.MeterProvider())
}

func ExampleInstallNewPipeline() {
	// Assumes the NEW_RELIC_API_KEY environment variable contains your New
	// Relic Event API key. This will error if it does not.
	controller, err := newrelic.InstallNewPipeline("My Service")
	if err != nil {
		log.Fatal(err)
	}
	defer controller.Stop(context.Background())
}
