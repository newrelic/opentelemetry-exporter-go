// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import "testing"

func TestServiceNameMissing(t *testing.T) {
	e, err := NewExporter("", "apiKey")
	if e != nil {
		t.Error(e)
	}
	if err != errServiceNameEmpty {
		t.Error(err)
	}
}
