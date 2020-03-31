package transform

import (
	"fmt"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/api/core"
	metricapi "go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"
	metricsdk "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

var defaultAttrs = map[string]string{
	instrumentationProviderAttrKey: instrumentationProviderAttrValue,
	collectorNameAttrKey:           collectorNameAttrValue,
}

func TestDefaultAttributes(t *testing.T) {
	attrs := attributes("", nil, metricsdk.NewLabels(metricsdk.LabelSlice{}, "", nil))
	if got, want := len(attrs), len(defaultAttrs); got != want {
		t.Errorf("incorrect number of default attributes: got %d, want %d", got, want)
	}
}

func TestServiceNameAttributes(t *testing.T) {
	want := "test-service-name"
	attrs := attributes(want, nil, metricsdk.NewLabels(metricsdk.LabelSlice{}, "", nil))
	if got, ok := attrs[serviceNameAttrKey]; !ok || got != want {
		t.Errorf("service.name attribute wrong: got %q, want %q", got, want)
	}
}

func TestDescriptorAttributes(t *testing.T) {
	for i, test := range []struct {
		opts []metricapi.Option
		want map[string]interface{}
	}{
		{
			[]metricapi.Option{
				metricapi.WithResource(*resource.New(core.Key("A").String("a"))),
			},
			map[string]interface{}{
				"A": "a",
			},
		},
		{
			[]metricapi.Option{
				metricapi.WithResource(*resource.New(core.Key("A").String("a"), core.Key("1").Int64(1))),
			},
			map[string]interface{}{
				"A": "a",
				"1": int64(1),
			},
		},
		{
			[]metricapi.Option{
				metricapi.WithUnit(unit.Bytes),
			},
			map[string]interface{}{
				"unit": "By",
			},
		},
		{
			[]metricapi.Option{
				metricapi.WithDescription("test description"),
			},
			map[string]interface{}{
				"description": "test description",
			},
		},
		{
			[]metricapi.Option{
				metricapi.WithResource(*resource.New(core.Key("B").String("b"), core.Key("2").Int64(2))),
				metricapi.WithUnit(unit.Milliseconds),
				metricapi.WithDescription("test description 2"),
			},
			map[string]interface{}{
				"B":           "b",
				"2":           int64(2),
				"unit":        "ms",
				"description": "test description 2",
			},
		},
	} {
		name := fmt.Sprintf("descriptor test %d", i)
		desc := metricapi.NewDescriptor(name, metricapi.CounterKind, core.Int64NumberKind, test.opts...)
		expected := make(map[string]interface{}, len(defaultAttrs)+len(test.want))
		for k, v := range defaultAttrs {
			expected[k] = v
		}
		for k, v := range test.want {
			expected[k] = v
		}
		got := attributes("", &desc, metricsdk.NewLabels(metricsdk.LabelSlice{}, "", nil))
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("%s: %#v != %#v", name, got, expected)
		}
	}
}

func TestLabelAttributes(t *testing.T) {
	for i, test := range []struct {
		labels []core.KeyValue
		want   map[string]interface{}
	}{
		{
			[]core.KeyValue{
				core.Key("A").String("a"),
			},
			map[string]interface{}{
				"A": "a",
			},
		},
		{
			[]core.KeyValue{
				core.Key("A").String("a"),
				core.Key("1").Int64(1),
			},
			map[string]interface{}{
				"A": "a",
				"1": int64(1),
			},
		},
	} {
		expected := make(map[string]interface{}, len(defaultAttrs)+len(test.want))
		for k, v := range defaultAttrs {
			expected[k] = v
		}
		for k, v := range test.want {
			expected[k] = v
		}
		l := metricsdk.NewLabels(metricsdk.LabelSlice(test.labels), "", nil)
		got := attributes("", nil, l)
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("labels test %d: %#v != %#v", i, got, expected)
		}
	}
}
