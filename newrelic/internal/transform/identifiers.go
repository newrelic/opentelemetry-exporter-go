// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package transform

const (
	errorCodeAttrKey    = "error.code"
	errorMessageAttrKey = "error.message"

	serviceNameAttrKey = "service.name"

	instrumentationProviderAttrKey   = "instrumentation.provider"
	instrumentationProviderAttrValue = "opentelemetry"

	collectorNameAttrKey   = "collector.name"
	collectorNameAttrValue = "newrelic-opentelemetry-exporter"
)
