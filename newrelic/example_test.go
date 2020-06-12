// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic_test

import (
	"log"
	"os"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/newrelic/opentelemetry-exporter-go/newrelic"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/sdk/trace"
)

func Example() {
	exporter, err := newrelic.NewExporter("My Service", os.Getenv("NEW_RELIC_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	tp, err := trace.NewProvider(trace.WithSyncer(exporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
}

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
