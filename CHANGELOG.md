# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [0.17.0] - 2021-03-04

### Changed

- Upgrade `go.opentelemetry.io/otel*` to v0.17.0 and `github.com/newrelic/newrelic-telemetry-sdk-go` to v0.5.2.
  ([#63](https://github.com/newrelic/opentelemetry-exporter-go/pull/63))
- Added SpanKind to Getting Started guide and simple sample application in order
  to provide a better New Relic UI experience.
  ([#54](https://github.com/newrelic/opentelemetry-exporter-go/pull/54))

## [0.15.1] - 2021-01-26

### Changed

- Upgraded `go.opentelemetry.io/otel*` dependencies to v0.16.0. ([#48](https://github.com/newrelic/opentelemetry-exporter-go/pull/48))

### Added

- Added Getting Started guide with sample application. ([#44](https://github.com/newrelic/opentelemetry-exporter-go/pull/44), [#49](https://github.com/newrelic/opentelemetry-exporter-go/pull/49))

## [0.14.0] - 2020-12-04

### Changed

- Upgrade `go.opentelemetry.io/otel*` to v0.14.0. ([#40](https://github.com/newrelic/opentelemetry-exporter-go/pull/40))

## [0.13.0] - 2020-10-28

### Added

- Support for metrics (#10)
- Version number has been modified to track the version numbers of the
  go.opentelemetry.io/otel upstream library.

### Changed

- Updated to use version 0.13.0 of the go.opentelemetry.io/otel packages. (#30)
- Standardized CHANGELOG.md format. When making changes to this project, add
  human-readable summaries of what you've done to the "Unreleased" section
  above. When creating a release, move that information into a new release
  section in this document. (#35)

## [0.1.0] - 2019-12-31

First release!

[Unreleased]: https://github.com/newrelic/opentelemetry-exporter-go/compare/v0.17.0...HEAD
[0.17.0]: https://github.com/newrelic/opentelemetry-exporter-go/compare/v0.15.0...v0.17.0
[0.15.0]: https://github.com/newrelic/opentelemetry-exporter-go/compare/v0.14.0...v0.15.0
[0.14.0]: https://github.com/newrelic/opentelemetry-exporter-go/compare/v0.13.0...v0.14.0
[0.13.0]: https://github.com/newrelic/opentelemetry-exporter-go/compare/v0.1.0...v0.13.0
[0.1.0]: https://github.com/newrelic/opentelemetry-exporter-go/releases/tag/v0.1.0
