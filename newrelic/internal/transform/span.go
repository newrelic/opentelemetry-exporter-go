// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package transform

import (
	"encoding/hex"
	"strings"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	apitrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/semconv"
	"google.golang.org/grpc/codes"
)

// Span transforms an OpenTelemetry SpanData into a New Relic Span for a
// unique service.
//
// https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#Span
// https://godoc.org/go.opentelemetry.io/otel/sdk/export/trace#SpanData
func Span(service string, span *trace.SpanData) telemetry.Span {
	// Default to exporter service name.
	serviceName := service

	// Account for the instrumentation provider and collector name.
	numAttrs := len(span.Attributes) + span.Resource.Len() + 2

	// If kind has been set, make room for it.
	if span.SpanKind != apitrace.SpanKindUnspecified {
		numAttrs++
	}

	// Consider everything other than an OK as an error.
	isError := span.StatusCode != codes.OK
	if isError {
		numAttrs += 2
	}

	// Copy attributes to new value.
	attrs := make(map[string]interface{}, numAttrs)
	for iter := span.Resource.Iter(); iter.Next(); {
		kv := iter.Label()
		// Resource service name overrides the exporter.
		if kv.Key == semconv.ServiceNameKey {
			serviceName = kv.Value.AsString()
		}
		attrs[string(kv.Key)] = kv.Value.AsInterface()
	}
	for _, kv := range span.Attributes {
		// Span service name overrides the Resource.
		if kv.Key == semconv.ServiceNameKey {
			serviceName = kv.Value.AsString()
		}
		attrs[string(kv.Key)] = kv.Value.AsInterface()
	}

	if span.SpanKind != apitrace.SpanKindUnspecified {
		attrs["span.kind"] = strings.ToUpper(span.SpanKind.String())
	}

	// New Relic registered attributes to identify where this data came from.
	attrs[instrumentationProviderAttrKey] = instrumentationProviderAttrValue
	attrs[collectorNameAttrKey] = collectorNameAttrValue

	if isError {
		attrs[errorCodeAttrKey] = uint32(span.StatusCode)
		attrs[errorMessageAttrKey] = span.StatusMessage
	}

	parentSpanID := ""
	if span.ParentSpanID.IsValid() {
		parentSpanID = hex.EncodeToString(span.ParentSpanID[:])
	}

	return telemetry.Span{
		ID:          span.SpanContext.SpanID.String(),
		TraceID:     span.SpanContext.TraceID.String(),
		Timestamp:   span.StartTime,
		Name:        span.Name,
		ParentID:    parentSpanID,
		Duration:    span.EndTime.Sub(span.StartTime),
		ServiceName: serviceName,
		Attributes:  attrs,
	}
}
