# New Relic Go OpenTelemetry Exporter [![GoDoc](https://godoc.org/github.com/newrelic/newrelic-opentelemetry-exporter-go/newrelic?status.svg)](https://godoc.org/github.com/newrelic/newrelic-opentelemetry-exporter-go/newrelic)

The `"github.com/newrelic/newrelic-opentelemetry-exporter-go/newrelic"` package
provides an exporter for sending OpenTelemetry data to New Relic.  Currently,
only traces are supported.

Example use:

```go
func initTracer() {
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
```

## Licensing
The New Relic Go OpenTelemetry exporter is licensed under the Apache 2.0 License.
The New Relic Go OpenTelemetry exporter also uses source code from third party
libraries. Full details on which libraries are used and the terms under which
they are licensed can be found in the third party notices document.


## Contributing
Full details are available in our CONTRIBUTING.md file. We'd love to get your
contributions to improve the Go OpenTelemetry exporter! Keep in mind when you
submit your pull request, you'll need to sign the CLA via the click-through
using CLA-Assistant. You only have to sign the CLA one time per project. To
execute our corporate CLA, which is required if your contribution is on
behalf of a company, or if you have any questions, please drop us an email at
open-source@newrelic.com.


## Limitations
The New Relic Telemetry APIs are rate limited. Please reference the
documentation for [New Relic Metrics
API](https://docs.newrelic.com/docs/introduction-new-relic-metric-api) and [New
Relic Trace API Requirements and
Limits](https://docs.newrelic.com/docs/apm/distributed-tracing/trace-api/trace-api-general-requirements-limits)
on the specifics of the rate limits.
