# Contributing to the New Relic Go OpenTelemetry Exporter
Thanks for your interest in contributing to the New Relic Go OpenTelemetry Exporter! We look forward to engaging with you.

## How to contribute
* Read this CONTRIBUTING file
* Read our [Code of Conduct](CODE_OF_CONDUCT.md)
* Submit a [pull request](#pull-request-guidelines) or [issue](#filing-issues--bug-reports). For pull requests, please also ensure that your work satisfies:
    * Unit tests (`go test ./...`)
    * [golint](https://github.com/golang/lint)
    * [go vet](https://golang.org/cmd/vet/)
    * [go fmt](https://golang.org/cmd/gofmt/)
* Ensure you’ve signed the CLA, otherwise you’ll be asked to do so.

## How to get help or ask questions
Do you have questions or are you experiencing unexpected behaviors after modifying this Open Source Software? Please engage with the “Build on New Relic” space in the [Explorers Hub](https://discuss.newrelic.com/c/build-on-new-relic/Open-Source-Agents-SDKs), New Relic’s Forum. Posts are publicly viewable by anyone, please do not include PII or sensitive information in your forum post.

## Contributor License Agreement ("CLA")

We'd love to get your contributions to improve the New Relic Go OpenTelemetry Exporter! Keep in mind when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.

To execute our corporate CLA, which is required if your contribution is on behalf of a company, or if you have any questions, please drop us an email at open-source@newrelic.com.

## Filing Issues & Bug Reports
We use GitHub issues to track public issues and bugs. If possible, please provide a link to an example app or gist that reproduces the issue. When filing an issue, please ensure your description is clear and includes the following information. Be aware that GitHub issues are publicly viewable by anyone, so please do not include personal information in your GitHub issue or in any of your contributions, except as minimally necessary for the purpose of supporting your issue. New Relic will process any personal data you submit through GitHub issues in compliance with the [New Relic Privacy Notice](https://newrelic.com/termsandconditions/privacy).   
- Project version (ex: 0.4.0)
- Custom configurations (ex: flag=true)
- Any modifications made to the exporter

### A note about vulnerabilities  
New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

## Setting up your environment
This Open Source Software can be used in a large number of environments, all of which have their own quirks and best practices. As such, while we are happy to provide documentation and assistance for unmodified Open Source Software, we cannot provide support for your specific environment or your modifications to the code.

## Pull Request Guidelines
Before we can accept a pull request, you must sign our [Contributor Licensing Agreement](#contributor-license-agreement-cla), if you have not already done so. This grants us the right to use your code under the same Apache 2.0 license as we use for this project in general.

If this is a notable change, please include a very short summary of your work in the "Unreleased" section of [CHANGELOG.md](./CHANGELOG.MD).

## Coding Style Guidelines
Our code base is formatted according to [gofmt](https://golang.org/cmd/gofmt/) and linted with [golint](https://github.com/golang/lint).

## License
By contributing to the New Relic Go OpenTelemetry Exporter, you agree that your contributions will be licensed under the LICENSE file in the root directory of this source tree.
