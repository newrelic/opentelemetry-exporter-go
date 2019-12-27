// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/export/trace"
)

func TestServiceNameMissing(t *testing.T) {
	e, err := NewExporter("", "apiKey")
	if e != nil {
		t.Error(e)
	}
	if err != errServiceNameEmpty {
		t.Error(err)
	}
}

func TestNilExporter(t *testing.T) {
	span := &trace.SpanData{}
	var e *Exporter

	e.ExportSpan(context.Background(), span)
	e.ExportSpans(context.Background(), []*trace.SpanData{span})
}
