# New Relic Go OpenTelemetry exporter [![GoDoc](https://godoc.org/github.com/newrelic/opentelemetry-exporter-go/newrelic?status.svg)](https://godoc.org/github.com/newrelic/opentelemetry-exporter-go/newrelic)

The `"github.com/newrelic/opentelemetry-exporter-go/newrelic"` package
provides an exporter for sending OpenTelemetry data to New Relic.  Currently,
traces and the latest metric instruments (as of v0.12 of Open Telemetry) are
supported.

Example use:

```go
import "github.com/newrelic/opentelemetry-exporter-go/newrelic"

func main() {
	// Assumes the NEW_RELIC_API_KEY environment variable contains your New
	// Relic Event API key. This will error if it does not.
	controller, err := newrelic.InstallNewPipeline("My Service")
	if err != nil {
		panic(err)
	}
	defer controller.Stop()

	/*...*/
}
```

## Disclaimer

This exporter is built with the alpha release of OpenTelemetry Go client. Due
to the rapid development of OpenTelemetry, this exporter does not guarantee
compatibility with future releases of the OpenTelemetry APIs. Additionally,
this exporter may introduce changes that are not backwards compatible without a
major version increment. We will strive to ensure backwards compatibility when
a stable version of the OpenTelemetry Go client is released.

## Find and use your data

For tips on how to find and query your data in New Relic, see [Find trace/span data](https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/trace-api/introduction-trace-api#view-data).

For general querying information, see:
- [Query New Relic data](https://docs.newrelic.com/docs/using-new-relic/data/understand-data/query-new-relic-data)
- [Intro to NRQL](https://docs.newrelic.com/docs/query-data/nrql-new-relic-query-language/getting-started/introduction-nrql)

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
Relic Trace API requirements and
limits](https://docs.newrelic.com/docs/apm/distributed-tracing/trace-api/trace-api-general-requirements-limits)
on the specifics of the rate limits.
