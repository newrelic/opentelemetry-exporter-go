package transform

import (
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"reflect"
	"testing"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/label"
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
	attrs := attributes("", nil, nil, nil)
	if got, want := len(attrs), len(defaultAttrs); got != want {
		t.Errorf("incorrect number of default attributes: got %d, want %d", got, want)
	}
}

func TestServiceNameAttributes(t *testing.T) {
	want := "test-service-name"
	attrs := attributes(want, nil, nil, nil)
	if got, ok := attrs[serviceNameAttrKey]; !ok || got != want {
		t.Errorf("service.name attribute wrong: got %q, want %q", got, want)
	}
}

func TestAttributes(t *testing.T) {
	for i, test := range []struct {
		res    *resource.Resource
		opts   []metricapi.InstrumentOption
		labels []kv.KeyValue
		want   map[string]interface{}
	}{
		{}, // test defaults
		{
			res:    resource.New(kv.String("A", "a")),
			opts:   nil,
			labels: nil,
			want: map[string]interface{}{
				"A": "a",
			},
		},
		{
			res:    resource.New(kv.String("A", "a"), kv.Int64("1", 1)),
			opts:   nil,
			labels: nil,
			want: map[string]interface{}{
				"A": "a",
				"1": int64(1),
			},
		},
		{
			res:    nil,
			opts:   []metricapi.InstrumentOption{metricapi.WithUnit(unit.Bytes)},
			labels: nil,
			want: map[string]interface{}{
				"unit": "By",
			},
		},
		{
			res:    nil,
			opts:   []metricapi.InstrumentOption{metricapi.WithDescription("test description")},
			labels: nil,
			want: map[string]interface{}{
				"description": "test description",
			},
		},
		{
			res:    nil,
			opts:   nil,
			labels: []kv.KeyValue{kv.String("A", "a")},
			want: map[string]interface{}{
				"A": "a",
			},
		},
		{
			res:    nil,
			opts:   nil,
			labels: []kv.KeyValue{kv.String("A", "a"), kv.Int64("1", 1)},
			want: map[string]interface{}{
				"A": "a",
				"1": int64(1),
			},
		},
		{
			res: resource.New(kv.String("K1", "V1"), kv.String("K2", "V2")),
			opts: []metricapi.InstrumentOption{
				metricapi.WithUnit(unit.Milliseconds),
				metricapi.WithDescription("d3"),
			},
			labels: []kv.KeyValue{kv.String("K2", "V3")},
			want: map[string]interface{}{
				"K1":          "V1",
				"K2":          "V3",
				"unit":        "ms",
				"description": "d3",
			},
		},
	} {
		name := fmt.Sprintf("descriptor test %d", i)
		desc := metricapi.NewDescriptor(name, metricapi.CounterKind, metric.Int64NumberKind, test.opts...)
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

var numKinds = []metric.NumberKind{metric.Int64NumberKind, metric.Float64NumberKind}

func TestMinMaxSumCountRecord(t *testing.T) {
	name := "test-mmsc"
	l := label.NewSet()
	for _, iKind := range []metric.Kind{metric.ValueRecorderKind, metric.ValueObserverKind} {
		for _, nKind := range numKinds {
			desc := metric.NewDescriptor(name, iKind, nKind)
			alloc := minmaxsumcount.New(2, &desc)
			mmsc, ckpt := &alloc[0], &alloc[1]

			var n metric.Number
			switch nKind {
			case metric.Int64NumberKind:
				n = metric.NewInt64Number(1)
			case metric.Float64NumberKind:
				n = metric.NewFloat64Number(1)
			}
			if err := mmsc.Update(context.Background(), n, &desc); err != nil {
				t.Fatal(err)
			}
			switch nKind {
			case metric.Int64NumberKind:
				n = metric.NewInt64Number(10)
			case metric.Float64NumberKind:
				n = metric.NewFloat64Number(10)
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
		desc := metric.NewDescriptor(name, metric.CounterKind, nKind)
		s := sumAgg.New(1)[0]

		var n metric.Number
		switch nKind {
		case metric.Int64NumberKind:
			n = metric.NewInt64Number(2)
		case metric.Float64NumberKind:
			n = metric.NewFloat64Number(2)
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
func (a fakeAgg) Update(context.Context, metric.Number, *metric.Descriptor) error { return nil }
func (a fakeAgg) Checkpoint(context.Context, *metric.Descriptor)                  {}
func (a fakeAgg) Merge(metricsdk.Aggregator, *metric.Descriptor) error            { return nil }

func TestErrUnimplementedAgg(t *testing.T) {
	fa := fakeAgg{}
	desc := metric.NewDescriptor("", metric.CounterKind, metric.Int64NumberKind)
	l := label.NewSet()
	_, err := Record("", metricsdk.NewRecord(&desc, &l, nil, fa, time.Now(), time.Now()))
	if !errors.Is(err, ErrUnimplementedAgg) {
		t.Errorf("unexpected error: %v", err)
	}
	if err == nil {
		t.Error("did not get ErrUnimplementedAgg error response")
	}
}
