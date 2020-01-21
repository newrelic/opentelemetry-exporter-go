// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"os"

	"github.com/newrelic/opentelemetry-exporter-go/newrelic"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	anotherKey = key.New("ex.com/another")
)

func initTracer() {
	exporter, err := newrelic.NewExporter("My Service", os.Getenv("NEW_RELIC_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
}

func main() {
	initTracer()

	ctx := context.Background()
	tracer := global.TraceProvider().Tracer("example.com/Service")

	err := tracer.WithSpan(ctx, "op1", func(ctx context.Context) error {
		trace.SpanFromContext(ctx).SetAttributes(anotherKey.String("Exists"))

		// do something
		return nil
	})

	if err != nil {
		panic(err)
	}
}
