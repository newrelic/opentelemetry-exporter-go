// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package transform

import (
	"reflect"
	"testing"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	exporttrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

const (
	service              = "myService"
	sampleTraceIDString  = "4bf92f3577b34da6a3ce929d0e0e4736"
	sampleSpanIDString   = "00f067aa0ba902b7"
	sampleParentIDString = "83887e5d7da921ba"
)

var (
	sampleTraceID, _  = trace.TraceIDFromHex(sampleTraceIDString)
	sampleSpanID, _   = trace.SpanIDFromHex(sampleSpanIDString)
	sampleParentID, _ = trace.SpanIDFromHex(sampleParentIDString)
)

func TestTransformSpans(t *testing.T) {
	now := time.Now()
	testcases := []struct {
		testname string
		input    *tracetest.SpanStub
		expect   telemetry.Span
	}{
		{
			testname: "basic span",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: sampleTraceID,
					SpanID:  sampleSpanID,
				}),
				StartTime: now,
				EndTime:   now.Add(2 * time.Second),
				Name:      "mySpan",
			},
			expect: telemetry.Span{
				Name:        "mySpan",
				ID:          sampleSpanIDString,
				TraceID:     sampleTraceIDString,
				Timestamp:   now,
				Duration:    2 * time.Second,
				ServiceName: service,
				Attributes: map[string]interface{}{
					instrumentationProviderAttrKey: instrumentationProviderAttrValue,
					collectorNameAttrKey:           collectorNameAttrValue,
				},
			},
		},
		{
			testname: "span with parent",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: sampleTraceID,
					SpanID:  sampleSpanID,
				}),
				Parent: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: sampleTraceID,
					SpanID:  sampleParentID,
				}),
				StartTime: now,
				EndTime:   now.Add(2 * time.Second),
				Name:      "mySpan",
			},
			expect: telemetry.Span{
				Name:        "mySpan",
				ID:          sampleSpanIDString,
				TraceID:     sampleTraceIDString,
				ParentID:    sampleParentID.String(),
				Timestamp:   now,
				Duration:    2 * time.Second,
				ServiceName: service,
				Attributes: map[string]interface{}{
					instrumentationProviderAttrKey: instrumentationProviderAttrValue,
					collectorNameAttrKey:           collectorNameAttrValue,
				},
			},
		},
		{
			testname: "span with error",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: sampleTraceID,
					SpanID:  sampleSpanID,
				}),
				Status: exporttrace.Status{
					Code:        codes.Error,
					Description: "ResourceExhausted",
				},
				StartTime: now,
				EndTime:   now.Add(2 * time.Second),
				Name:      "mySpan",
			},
			expect: telemetry.Span{
				Name:        "mySpan",
				ID:          sampleSpanIDString,
				TraceID:     sampleTraceIDString,
				Timestamp:   now,
				Duration:    2 * time.Second,
				ServiceName: service,
				Attributes: map[string]interface{}{
					instrumentationProviderAttrKey: instrumentationProviderAttrValue,
					collectorNameAttrKey:           collectorNameAttrValue,
					errorCodeAttrKey:               uint32(codes.Error),
					errorMessageAttrKey:            "ResourceExhausted",
				},
			},
		},
		{
			testname: "span with attributes",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: sampleTraceID,
					SpanID:  sampleSpanID,
				}),
				StartTime: now,
				EndTime:   now.Add(2 * time.Second),
				Name:      "mySpan",
				Attributes: []attribute.KeyValue{
					attribute.Bool("x0", true),
					attribute.Float64("x1", 1.0),
					attribute.Int("x2", 2),
					attribute.Int64("x3", 3),
					attribute.String("x4", "4"),
				},
			},
			expect: telemetry.Span{
				Name:        "mySpan",
				ID:          sampleSpanIDString,
				TraceID:     sampleTraceIDString,
				Timestamp:   now,
				Duration:    2 * time.Second,
				ServiceName: service,
				Attributes: map[string]interface{}{
					"x0":                           true,
					"x1":                           float64(1.0),
					"x2":                           int64(2),
					"x3":                           int64(3),
					"x4":                           "4",
					instrumentationProviderAttrKey: instrumentationProviderAttrValue,
					collectorNameAttrKey:           collectorNameAttrValue,
				},
			},
		},
		{
			testname: "span with attributes and error",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: sampleTraceID,
					SpanID:  sampleSpanID,
				}),
				Status: exporttrace.Status{
					Code:        codes.Error,
					Description: "ResourceExhausted",
				},
				StartTime: now,
				EndTime:   now.Add(2 * time.Second),
				Name:      "mySpan",
				Attributes: []attribute.KeyValue{
					attribute.Bool("x0", true),
				},
			},
			expect: telemetry.Span{
				Name:        "mySpan",
				ID:          sampleSpanIDString,
				TraceID:     sampleTraceIDString,
				Timestamp:   now,
				Duration:    2 * time.Second,
				ServiceName: service,
				Attributes: map[string]interface{}{
					"x0":                           true,
					instrumentationProviderAttrKey: instrumentationProviderAttrValue,
					collectorNameAttrKey:           collectorNameAttrValue,
					errorCodeAttrKey:               uint32(codes.Error),
					errorMessageAttrKey:            "ResourceExhausted",
				},
			},
		},
		{
			testname: "span with service name in resource",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: sampleTraceID,
					SpanID:  sampleSpanID,
				}),
				StartTime: now,
				EndTime:   now.Add(2 * time.Second),
				Name:      "mySpan",
				Resource: resource.NewSchemaless(
					attribute.String("service.name", "resource service"),
				),
			},
			expect: telemetry.Span{
				Name:        "mySpan",
				ID:          sampleSpanIDString,
				TraceID:     sampleTraceIDString,
				Timestamp:   now,
				Duration:    2 * time.Second,
				ServiceName: "resource service",
				Attributes: map[string]interface{}{
					"service.name":                 "resource service",
					instrumentationProviderAttrKey: instrumentationProviderAttrValue,
					collectorNameAttrKey:           collectorNameAttrValue,
				},
			},
		},
		{
			testname: "span with a kind",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: sampleTraceID,
					SpanID:  sampleSpanID,
				}),
				SpanKind:  trace.SpanKindClient,
				StartTime: now,
				EndTime:   now.Add(2 * time.Second),
				Name:      "mySpan",
				Resource: resource.NewSchemaless(
					attribute.String("service.name", "resource service"),
				),
			},
			expect: telemetry.Span{
				Name:        "mySpan",
				ID:          sampleSpanIDString,
				TraceID:     sampleTraceIDString,
				Timestamp:   now,
				Duration:    2 * time.Second,
				ServiceName: "resource service",
				Attributes: map[string]interface{}{
					"service.name":                 "resource service",
					"span.kind":                    "client",
					instrumentationProviderAttrKey: instrumentationProviderAttrValue,
					collectorNameAttrKey:           collectorNameAttrValue,
				},
			},
		},
		{
			testname: "span with service name in attributes",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: sampleTraceID,
					SpanID:  sampleSpanID,
				}),
				StartTime: now,
				EndTime:   now.Add(2 * time.Second),
				Name:      "mySpan",
				Resource: resource.NewSchemaless(
					attribute.String("service.name", "resource service"),
				),
				Attributes: []attribute.KeyValue{
					attribute.String("service.name", "attributes service"),
				},
			},
			expect: telemetry.Span{
				Name:        "mySpan",
				ID:          sampleSpanIDString,
				TraceID:     sampleTraceIDString,
				Timestamp:   now,
				Duration:    2 * time.Second,
				ServiceName: "attributes service",
				Attributes: map[string]interface{}{
					"service.name":                 "attributes service",
					instrumentationProviderAttrKey: instrumentationProviderAttrValue,
					collectorNameAttrKey:           collectorNameAttrValue,
				},
			},
		},
	}
	for _, tc := range testcases {
		if got := Span(service, tc.input.Snapshot()); !reflect.DeepEqual(got, tc.expect) {
			t.Errorf("%s: %#v != %#v", tc.testname, got, tc.expect)
		}
	}
}
