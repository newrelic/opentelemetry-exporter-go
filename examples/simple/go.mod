module github.com/newrelic/opentelemetry-exporter-go/examples/simple

go 1.15

replace github.com/newrelic/opentelemetry-exporter-go => ../../

require (
	github.com/newrelic/newrelic-telemetry-sdk-go v0.5.2
	github.com/newrelic/opentelemetry-exporter-go v0.17.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/metric v0.19.0
	go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/sdk/metric v0.19.0
	go.opentelemetry.io/otel/trace v0.19.0
)
