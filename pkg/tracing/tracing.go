/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package tracing provides OpenTelemetry tracing infrastructure for the gateway-api-inference-extension
package tracing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	ServiceName = "gateway-api-inference-extension"

	envOTELTracingEnabled   = "OTEL_TRACING_ENABLED"
	envOTELExporterEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envOTELCaptureEvent     = "OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT"
	envOTELServiceName      = "OTEL_SERVICE_NAME"
	envOTELSamplingRate     = "OTEL_SAMPLING_RATE"

	OperationGatewayRequest       = "gateway.ext_proc.request"
	OperationRequestOrchestration = "gateway.request_orchestration"
	OperationScheduling           = "gateway.scheduling"
	OperationPostResponse         = "gateway.post_response"

	AttrGenAIRequestModel         = "gen_ai.request.model"
	AttrOperationOutcome          = "operation.outcome"
	AttrGatewayTargetModel        = "target_model"
	AttrGatewayRequestCriticality = "criticality"
	AttrGatewayTargetEndpoint     = "target_endpoint"

	// Span Events
	GenAISystemMessageEvent    = "gen_ai.system.message"
	GenAIUserMessageEvent      = "gen_ai.user.message"
	GenAIAssistantMessageEvent = "gen_ai.assistant.message"

	GenAiMessageContentKey = attribute.Key("content")

	OutcomeSuccess = "success"
	OutcomeError   = "error"
)

type Config struct {
	Enabled          bool
	CaptureEvent     bool
	ExporterEndpoint string
	SamplingRate     float64
	ServiceName      string
}

func NewConfigFromEnv() *Config {
	config := &Config{
		Enabled:          false,
		ExporterEndpoint: "http://localhost:4317",
		SamplingRate:     0.1,
		ServiceName:      ServiceName,
	}

	if enabled := os.Getenv(envOTELTracingEnabled); enabled != "" {
		if enabledBool, err := strconv.ParseBool(enabled); err == nil {
			config.Enabled = enabledBool
		}
	}

	if captureEvent := os.Getenv(envOTELCaptureEvent); captureEvent != "" {
		if captureEventBool, err := strconv.ParseBool(captureEvent); err == nil {
			config.CaptureEvent = captureEventBool
		}
	}

	if endpoint := os.Getenv(envOTELExporterEndpoint); endpoint != "" {
		config.ExporterEndpoint = endpoint
	}

	if serviceName := os.Getenv(envOTELServiceName); serviceName != "" {
		config.ServiceName = serviceName
	}

	if samplingRate := os.Getenv(envOTELSamplingRate); samplingRate != "" {
		if rate, err := strconv.ParseFloat(samplingRate, 64); err == nil {
			config.SamplingRate = rate
		}
	}

	return config
}

// Initialize sets up OpenTelemetry tracing with the given configuration.
// It always sets up context propagation, even if tracing is disabled.
func Initialize(ctx context.Context, config *Config) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if !config.Enabled {
		// Return a no-op shutdown function if tracing is disabled
		return shutdown, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpointURL(config.ExporterEndpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		// sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRate)),
	)

	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)

	// Set up logger provider.
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithInsecure(),
		otlploggrpc.WithEndpointURL(config.ExporterEndpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
	)
	shutdownFuncs = append(shutdownFuncs, lp.Shutdown)
	global.SetLoggerProvider(lp)

	otel.SetTracerProvider(tp)

	return shutdown, nil
}

func StartSpan(ctx context.Context, name, operation string) (context.Context, trace.Span) {
	tracer := otel.Tracer(ServiceName)
	return tracer.Start(ctx, name)
}

func StartGatewaySpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	ctx, span := StartSpan(ctx, operation, operation)
	// TODO: Add common gateway attributes here
	return ctx, span
}

func SetSpanError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func SetSpanSuccess(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}
