package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cybozu-go/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.GetTracerProvider().Tracer("oteltest")

func newResource() *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("server"),
		semconv.ServiceVersion("0.0.1"),
	)
}

func myJob(ctx context.Context) {
	_, span := tracer.Start(ctx, "myjob")
	defer span.End()

	time.Sleep(10 * time.Millisecond)
}

func handler(w http.ResponseWriter, _ *http.Request) {
	ctx, span := tracer.Start(context.Background(), "handler")
	defer span.End()
	traceId := trace.SpanContextFromContext(ctx).TraceID().String()

	myJob(ctx)
	io.WriteString(w, "Hello")

	log.Info("Hello", map[string]interface{}{"trace_id": traceId})
}

func main() {
	log.DefaultLogger().SetFormatter(log.JSONFormat{})

	client := otlptracehttp.NewClient()
	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		log.ErrorExit(fmt.Errorf("creating OTLP trace exporter: %w", err))
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			log.ErrorExit(err)
		}
	}()

	log.Info("start", nil)
	http.HandleFunc("/", handler)
	log.ErrorExit(http.ListenAndServe(":8080", nil))

}
