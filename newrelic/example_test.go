// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic_test

import (
	"log"
	"os"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/newrelic/opentelemetry-exporter-go/newrelic"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
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
	tp, err := trace.NewProvider(trace.WithSyncer(exporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
}

func ExampleNewExportPipeline() {
	// Include environment in resource.
	r := resource.New(
		label.String("environment", "production"),
		semconv.ServiceNameKey.String("My Service"),
	)

	// Assumes the NEW_RELIC_API_KEY environment variable contains your New
	// Relic Insights insert API key. This will error if it does not.
	traceProvider, controller, err := newrelic.NewExportPipeline(
		"My Service",
		[]trace.ProviderOption{
			trace.WithConfig(trace.Config{
				// Conservative sampler.
				DefaultSampler: trace.ParentSample(trace.NeverSample()),
				// Reduce span events.
				MaxEventsPerSpan: 10,
				Resource:         r,
			}),
		},
		[]push.Option{
			// Increase push frequency.
			push.WithPeriod(time.Second),
			push.WithResource(r),
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	defer controller.Stop()

	global.SetTraceProvider(traceProvider)
	global.SetMeterProvider(controller.Provider())
}

func ExampleInstallNewPipeline() {
	// Assumes the NEW_RELIC_API_KEY environment variable contains your New
	// Relic Insights insert API key. This will error if it does not.
	controller, err := newrelic.InstallNewPipeline("My Service")
	if err != nil {
		log.Fatal(err)
	}
	defer controller.Stop()
}
