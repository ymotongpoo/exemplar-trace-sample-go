// Copyright 2021 Yoshi Yamaguchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

const (
	MeasureLatency = "task_latency_go"
	LatencyUnit    = stats.UnitMilliseconds
)

var (
	ProjectID   string
	MLatency    *stats.Float64Measure
	LatencyView *view.View
)

func initExporter() {
	var err error
	ProjectID, err = metadata.ProjectID()
	if err != nil {
		ProjectID = os.Getenv("GCP_PROJECT_ID")
	}
	if ProjectID == "" {
		log.Fatalf("Specify GCP Project ID in $GCP_PROJECT_ID")
	}

	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: ProjectID,
	})
	if err != nil {
		log.Fatalf("failed to create Cloud Ops exporter: %v", err)
	}
	if err = exporter.StartMetricsExporter(); err != nil {
		log.Fatalf("Faield to start Cloud Monitoring exporter: %v", err)
	}

	MLatency = stats.Float64(MeasureLatency, "query latency", "ms")
	LatencyView = &view.View{
		Name:        MeasureLatency,
		Measure:     MLatency,
		Description: "the latency of root function per query",
		Aggregation: view.Distribution(
			[]float64{100.0, 200.0, 400.0, 1000.0, 2000.0, 4000.0}...,
		),
	}

	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	trace.RegisterExporter(exporter)
	view.RegisterExporter(exporter)
	view.Register(LatencyView)
}

func main() {
	initExporter()
	log.Println("starting loop")
	interval := 10 * time.Second
	t := time.NewTicker(interval)
	go func() {
		for range t.C {
			log.Println("loop start")
			Root()
			log.Println("loop end")
		}
	}()

	// dummy handler for Cloud Run
	port := os.Getenv("PORT")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

func Root() {
	ctx := context.Background()
	ctx, span := trace.StartSpan(ctx, "root")
	defer span.End()

	start := time.Now().UnixNano()
	Foo(ctx)
	end := time.Now().UnixNano()
	ms := (end - start) / (1000 * 1000)

	measurements := stats.WithMeasurements(MLatency.M(float64(ms)))
	attachments := stats.WithAttachments(metricdata.Attachments{
		metricdata.AttachmentKeySpanContext: span.SpanContext(),
	})
	stats.RecordWithOptions(ctx, measurements, attachments)
	return
}

func Foo(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "child_foo")
	defer span.End()
	ms := rand.Int63n(2000)
	log.Printf("task foo blocked: %vms", ms)
	span.AddAttributes(trace.Int64Attribute("foo_wait", ms))
	Bar(ctx)
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func Bar(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "child_bar")
	defer span.End()
	ms := rand.Int63n(1000)
	log.Printf("task bar blocked: %vms", ms)
	span.AddAttributes(trace.Int64Attribute("bar_wait", ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
