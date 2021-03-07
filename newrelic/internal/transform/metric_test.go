package transform

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
	sumAgg "go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/unit"
)

var defaultAttrs = map[string]string{
	instrumentationProviderAttrKey: instrumentationProviderAttrValue,
	collectorNameAttrKey:           collectorNameAttrValue,
}

func TestDefaultAttributes(t *testing.T) {
	attrs := attributes("", nil, nil, nil)
	if got, want := len(attrs), len(defaultAttrs); got != want {
		t.Errorf("incorrect number of default attributes: got %d, want %d", got, want)
	}
}

func TestServiceNameAttributes(t *testing.T) {
	wrong := "wrong"
	want := "test-service-name"
	attrs := attributes(want, nil, nil, nil)
	if got, ok := attrs[serviceNameAttrKey]; !ok || got != want {
		t.Errorf("service.name attribute wrong: got %q, want %q", got, want)
	}

	r := resource.NewWithAttributes(label.String("service.name", want))
	attrs = attributes(wrong, r, nil, nil)
	if got, ok := attrs[serviceNameAttrKey]; !ok || got != want {
		t.Errorf("service.name attribute wrong: got %q, want %q", got, want)
	}

	r = resource.NewWithAttributes(label.String("service.name", wrong))
	l := label.NewSet(label.String("service.name", want))
	attrs = attributes(wrong, r, nil, &l)
	if got, ok := attrs[serviceNameAttrKey]; !ok || got != want {
		t.Errorf("service.name attribute wrong: got %q, want %q", got, want)
	}
}

func TestAttributes(t *testing.T) {
	for i, test := range []struct {
		res    *resource.Resource
		opts   []metric.InstrumentOption
		labels []label.KeyValue
		want   map[string]interface{}
	}{
		{}, // test defaults
		{
			res:    resource.NewWithAttributes(label.String("A", "a")),
			opts:   nil,
			labels: nil,
			want: map[string]interface{}{
				"A": "a",
			},
		},
		{
			res:    resource.NewWithAttributes(label.String("A", "a"), label.Int64("1", 1)),
			opts:   nil,
			labels: nil,
			want: map[string]interface{}{
				"A": "a",
				"1": int64(1),
			},
		},
		{
			res:    nil,
			opts:   []metric.InstrumentOption{metric.WithUnit(unit.Bytes)},
			labels: nil,
			want: map[string]interface{}{
				"unit": "By",
			},
		},
		{
			res:    nil,
			opts:   []metric.InstrumentOption{metric.WithDescription("test description")},
			labels: nil,
			want: map[string]interface{}{
				"description": "test description",
			},
		},
		{
			res:    nil,
			opts:   nil,
			labels: []label.KeyValue{label.String("A", "a")},
			want: map[string]interface{}{
				"A": "a",
			},
		},
		{
			res:    nil,
			opts:   nil,
			labels: []label.KeyValue{label.String("A", "a"), label.Int64("1", 1)},
			want: map[string]interface{}{
				"A": "a",
				"1": int64(1),
			},
		},
		{
			res: resource.NewWithAttributes(label.String("K1", "V1"), label.String("K2", "V2")),
			opts: []metric.InstrumentOption{
				metric.WithUnit(unit.Milliseconds),
				metric.WithDescription("d3"),
			},
			labels: []label.KeyValue{label.String("K2", "V3")},
			want: map[string]interface{}{
				"K1":          "V1",
				"K2":          "V3",
				"unit":        "ms",
				"description": "d3",
			},
		},
	} {
		name := fmt.Sprintf("descriptor test %d", i)
		desc := metric.NewDescriptor(name, metric.CounterInstrumentKind, number.Int64Kind, test.opts...)
		l := label.NewSet(test.labels...)
		expected := make(map[string]interface{}, len(defaultAttrs)+len(test.want))
		for k, v := range defaultAttrs {
			expected[k] = v
		}
		for k, v := range test.want {
			expected[k] = v
		}
		got := attributes("", test.res, &desc, &l)
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("%s: %#v != %#v", name, got, expected)
		}
	}
}

var numKinds = []number.Kind{number.Int64Kind, number.Float64Kind}

func TestMinMaxSumCountRecord(t *testing.T) {
	name := "test-mmsc"
	l := label.NewSet()
	for _, iKind := range []metric.InstrumentKind{metric.ValueRecorderInstrumentKind, metric.ValueObserverInstrumentKind} {
		for _, nKind := range numKinds {
			desc := metric.NewDescriptor(name, iKind, nKind)
			alloc := minmaxsumcount.New(2, &desc)
			mmsc, ckpt := &alloc[0], &alloc[1]

			var n number.Number
			switch nKind {
			case number.Int64Kind:
				n = number.NewInt64Number(1)
			case number.Float64Kind:
				n = number.NewFloat64Number(1)
			}
			if err := mmsc.Update(context.Background(), n, &desc); err != nil {
				t.Fatal(err)
			}
			switch nKind {
			case number.Int64Kind:
				n = number.NewInt64Number(10)
			case number.Float64Kind:
				n = number.NewFloat64Number(10)
			}
			if err := mmsc.Update(context.Background(), n, &desc); err != nil {
				t.Fatal(err)
			}

			if err := mmsc.SynchronizedMove(ckpt, &desc); err != nil {
				t.Fatal(err)
			}

			m, err := Record("", metricsdk.NewRecord(&desc, &l, nil, ckpt, time.Now(), time.Now()))
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
	l := label.NewSet()
	for _, nKind := range numKinds {
		desc := metric.NewDescriptor(name, metric.CounterInstrumentKind, nKind)
		s := sumAgg.New(1)[0]

		var n number.Number
		switch nKind {
		case number.Int64Kind:
			n = number.NewInt64Number(2)
		case number.Float64Kind:
			n = number.NewFloat64Number(2)
		}
		if err := s.Update(context.Background(), n, &desc); err != nil {
			t.Fatal(err)
		}

		m, err := Record("", metricsdk.NewRecord(&desc, &l, nil, &s, time.Now(), time.Now()))
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

func (a fakeAgg) Kind() aggregation.Kind                                          { return aggregation.MinMaxSumCountKind }
func (a fakeAgg) Update(context.Context, number.Number, *metric.Descriptor) error { return nil }
func (a fakeAgg) Checkpoint(context.Context, *metric.Descriptor)                  {}
func (a fakeAgg) Merge(metricsdk.Aggregator, *metric.Descriptor) error            { return nil }

func TestErrUnimplementedAgg(t *testing.T) {
	fa := fakeAgg{}
	desc := metric.NewDescriptor("", metric.CounterInstrumentKind, number.Int64Kind)
	l := label.NewSet()
	_, err := Record("", metricsdk.NewRecord(&desc, &l, nil, fa, time.Now(), time.Now()))
	if !errors.Is(err, ErrUnimplementedAgg) {
		t.Errorf("unexpected error: %v", err)
	}
	if err == nil {
		t.Error("did not get ErrUnimplementedAgg error response")
	}
}
