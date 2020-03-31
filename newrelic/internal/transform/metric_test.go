package transform

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/metric"
	metricapi "go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"
	metricsdk "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
	sumAgg "go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
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

var numKinds = []core.NumberKind{core.Int64NumberKind, core.Uint64NumberKind, core.Float64NumberKind}

func TestMinMaxSumCountRecord(t *testing.T) {
	name := "test-mmsc"
	l := metricsdk.NewLabels(metricsdk.LabelSlice{}, "", nil)
	for _, iKind := range []metric.Kind{metric.MeasureKind, metric.ObserverKind} {
		for _, nKind := range numKinds {
			desc := metric.NewDescriptor(name, iKind, nKind)
			mmsc := minmaxsumcount.New(&desc)

			var n core.Number
			switch nKind {
			case core.Int64NumberKind:
				n = core.NewInt64Number(1)
			case core.Uint64NumberKind:
				n = core.NewUint64Number(1)
			case core.Float64NumberKind:
				n = core.NewFloat64Number(1)
			}
			if err := mmsc.Update(context.Background(), n, &desc); err != nil {
				t.Fatal(err)
			}
			switch nKind {
			case core.Int64NumberKind:
				n = core.NewInt64Number(10)
			case core.Uint64NumberKind:
				n = core.NewUint64Number(10)
			case core.Float64NumberKind:
				n = core.NewFloat64Number(10)
			}
			if err := mmsc.Update(context.Background(), n, &desc); err != nil {
				t.Fatal(err)
			}

			mmsc.Checkpoint(context.Background(), &desc)

			m, err := Record("", metricsdk.NewRecord(&desc, l, mmsc))
			if err != nil {
				t.Fatalf("Record(MMSC,%s,%s) error: %v", nKind, iKind, err)
			}
			summary, ok := m.(telemetry.Summary)
			if !ok {
				t.Fatalf("Record(MMSC,%s,%s) did not return a Summary", nKind, iKind)
			}
			if got := summary.Name; got != name {
				t.Errorf("Record(MMSC,%s,%s) name: got %q, want %q", nKind, iKind, got, name)
			}
			if want := float64(1); summary.Min != want {
				t.Errorf("Record(MMSC,%s,%s) min: got %g, want %g", nKind, iKind, summary.Min, want)
			}
			if want := float64(10); summary.Max != want {
				t.Errorf("Record(MMSC,%s,%s) max: got %g, want %g", nKind, iKind, summary.Max, want)
			}
			if want := float64(11); summary.Sum != want {
				t.Errorf("Record(MMSC,%s,%s) sum: got %g, want %g", nKind, iKind, summary.Sum, want)
			}
			if want := float64(2); summary.Count != want {
				t.Errorf("Record(MMSC,%s,%s) count: got %g, want %g", nKind, iKind, summary.Count, want)
			}
		}
	}
}

func TestSumRecord(t *testing.T) {
	name := "test-sum"
	l := metricsdk.NewLabels(metricsdk.LabelSlice{}, "", nil)
	for _, nKind := range numKinds {
		desc := metric.NewDescriptor(name, metric.CounterKind, nKind)
		s := sumAgg.New()

		var n core.Number
		switch nKind {
		case core.Int64NumberKind:
			n = core.NewInt64Number(2)
		case core.Uint64NumberKind:
			n = core.NewUint64Number(2)
		case core.Float64NumberKind:
			n = core.NewFloat64Number(2)
		}
		if err := s.Update(context.Background(), n, &desc); err != nil {
			t.Fatal(err)
		}

		s.Checkpoint(context.Background(), &desc)
		m, err := Record("", metricsdk.NewRecord(&desc, l, s))
		if err != nil {
			t.Fatalf("Record(SUM,%s) error: %v", nKind, err)
		}
		c, ok := m.(telemetry.Count)
		if !ok {
			t.Fatalf("Record(SUM,%s) did not return a Counter", nKind)
		}
		if got := c.Name; got != name {
			t.Errorf("Record(SUM,%s) name: got %q, want %q", nKind, got, name)
		}
		if got := c.Name; got != name {
			t.Errorf("Record(SUM) name: got %q, want %q", got, name)
		}
		if want := float64(2); c.Value != want {
			t.Errorf("Record(SUM,%s) value: got %g, want %g", nKind, c.Value, want)
		}
	}
}

type fakeAgg struct{}

func (a fakeAgg) Update(context.Context, core.Number, *metric.Descriptor) error { return nil }
func (a fakeAgg) Checkpoint(context.Context, *metric.Descriptor)                {}
func (a fakeAgg) Merge(metricsdk.Aggregator, *metric.Descriptor) error          { return nil }

func TestErrUnimplementedAgg(t *testing.T) {
	fa := fakeAgg{}
	desc := metric.NewDescriptor("", metric.CounterKind, core.Int64NumberKind)
	l := metricsdk.NewLabels(metricsdk.LabelSlice{}, "", nil)
	_, err := Record("", metricsdk.NewRecord(&desc, l, fa))
	if !errors.Is(err, ErrUnimplementedAgg) {
		t.Errorf("unexpected error: %v", err)
	}
	if err == nil {
		t.Error("did not get ErrUnimplementedAgg error response")
	}
}
