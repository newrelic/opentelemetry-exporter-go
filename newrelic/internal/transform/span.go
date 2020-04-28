// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package transform

import (
	"encoding/hex"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.opentelemetry.io/otel/sdk/export/trace"
	"google.golang.org/grpc/codes"
)

// Span transforms an OpenTelemetry SpanData into a New Relic Span for a
// unique service.
//
// https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#Span
// https://godoc.org/go.opentelemetry.io/otel/sdk/export/trace#SpanData
func Span(service string, span *trace.SpanData) telemetry.Span {
	// Account for the instrumentation provider and collector name.
	numAttrs := len(span.Attributes) + 2

	// Consider everything other than an OK as an error.
	isError := span.StatusCode != codes.OK
	if isError {
		numAttrs += 2
	}

	// Copy attributes to new value.
	attrs := make(map[string]interface{}, numAttrs)
	for _, kv := range span.Attributes {
		attrs[string(kv.Key)] = kv.Value.AsInterface()
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
		ServiceName: service,
		Attributes:  attrs,
	}
}
