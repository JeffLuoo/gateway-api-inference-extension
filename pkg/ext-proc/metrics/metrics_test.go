package metrics

import (
	"os"
	"testing"
	"time"

	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/component-base/metrics/testutil"
)

const RequestTotalMetric = LLMServiceModelComponent + "_request_total"

func TestMonitorRequest(t *testing.T) {
	type requests struct {
		llmserviceName  string
		modelName       string
		targetModelName string
		reqSize         int
		elapsed         time.Duration
	}
	scenarios := []struct {
		name string
		reqs []requests
	}{{
		name: "multiple requests",
		reqs: []requests{
			{
				modelName:       "m10",
				targetModelName: "t10",
				reqSize:         1200,
			},
			{
				modelName:       "m10",
				targetModelName: "t10",
				reqSize:         500,
			},
			{
				modelName:       "m10",
				targetModelName: "t11",
				reqSize:         2480,
			},
			{
				modelName:       "m20",
				targetModelName: "t20",
				reqSize:         80,
				llmserviceName:  "s10",
				modelName:       "m10",
				targetModelName: "t10",
				reqSize:         10,
				elapsed:         time.Millisecond * 10,
			},
			{
				llmserviceName:  "s10",
				modelName:       "m10",
				targetModelName: "t10",
				reqSize:         20,
				elapsed:         time.Millisecond * 20,
			},
			{
				llmserviceName:  "s10",
				modelName:       "m10",
				targetModelName: "t11",
				reqSize:         30,
				elapsed:         time.Millisecond * 30,
			},
			{
				llmserviceName:  "s20",
				modelName:       "m20",
				targetModelName: "t20",
				reqSize:         40,
				elapsed:         time.Millisecond * 40,
			},
		},
	}}
	Register()
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for _, req := range scenario.reqs {
				MonitorRequest(req.llmserviceName, req.modelName, req.targetModelName, req.reqSize, req.elapsed)
			}
			wantRequestTotal, err := os.Open("testdata/request_total_metric")
			defer func() {
				if err := wantRequestTotal.Close(); err != nil {
					t.Error(err)
				}
			}()
			if err != nil {
				t.Fatal(err)
			}
			if err := testutil.GatherAndCompare(legacyregistry.DefaultGatherer, wantRequestTotal, RequestTotalMetric); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRecordRequestLatencies(t *testing.T) {
	timeBaseline := time.Now()
	type requests struct {
		modelName       string
		targetModelName string
		receivedTime    time.Time
		completeTime    time.Time
	}
	scenarios := []struct {
		name    string
		reqs    []requests
		invalid bool
	}{{
		name: "multiple requests",
		reqs: []requests{
			{
				modelName:       "m10",
				targetModelName: "t10",
				receivedTime:    timeBaseline,
				completeTime:    timeBaseline.Add(time.Millisecond * 10),
			},
			{
				modelName:       "m10",
				targetModelName: "t10",
				receivedTime:    timeBaseline,
				completeTime:    timeBaseline.Add(time.Millisecond * 1600),
			},
			{
				modelName:       "m10",
				targetModelName: "t11",
				receivedTime:    timeBaseline,
				completeTime:    timeBaseline.Add(time.Millisecond * 60),
			},
			{
				modelName:       "m20",
				targetModelName: "t20",
				receivedTime:    timeBaseline,
				completeTime:    timeBaseline.Add(time.Millisecond * 120),
			},
		},
	},
		{
			name: "invalid elapsed time",
			reqs: []requests{
				{
					modelName:       "m10",
					targetModelName: "t10",
					receivedTime:    timeBaseline.Add(time.Millisecond * 10),
					completeTime:    timeBaseline,
				}},
			invalid: true,
		}}
	Register()
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for _, req := range scenario.reqs {
				success := RecordRequestLatencies(req.modelName, req.targetModelName, req.receivedTime, req.completeTime)
				if success == scenario.invalid {
					t.Errorf("got record success(%v), but the request expects invalid(%v)", success, scenario.invalid)
				}
			}

			wantRequestLatencies, err := os.Open("testdata/request_duration_seconds_metric")
			defer func() {
				if err := wantRequestLatencies.Close(); err != nil {
					t.Error(err)
				}
			}()
			if err != nil {
				t.Fatal(err)
			}
			if err := testutil.GatherAndCompare(legacyregistry.DefaultGatherer, wantRequestLatencies, RequestLatenciesMetric); err != nil {
				t.Error(err)
			}
		})
	}
}
