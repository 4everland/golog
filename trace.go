package golog

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"os"
	"strings"
)

const (
	OTLPProtocol     = "OTEL_EXPORTER_OTLP_PROTOCOL"
	OTLPEndpoint     = "OTEL_EXPORTER_OTLP_ENDPOINT"
	OTLPCompression  = "OTEL_EXPORTER_OTLP_COMPRESSION"
	OTLPHeaders      = "OTEL_EXPORTER_OTLP_HEADERS"
	OTLPPath         = "OTEL_EXPORTER_OTLP_PATH"
	OTLPExportEnable = "OTEL_AGENT_ENABLED"
)

// InitOTLPTracer init export by env
// headers example: x-otel-project=,x-otel-access-id=,x-otel-access-key=
func InitOTLPTracer(serverName string, ratio float64) error {
	if os.Getenv(OTLPExportEnable) == "false" {
		return nil
	}
	protocol := os.Getenv(OTLPProtocol)
	host := os.Getenv(OTLPEndpoint)
	headersEnv := os.Getenv(OTLPHeaders)
	headers := make(map[string]string)
	for _, pairs := range strings.Split(headersEnv, ",") {
		s := strings.Split(pairs, "=")
		if len(s) != 2 {
			continue
		}
		headers[strings.TrimSpace(s[0])] = strings.TrimSpace(s[1])
	}
	var (
		exp *otlptrace.Exporter
		err error
	)
	if protocol == "grpc" {
		options := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(host),
			otlptracegrpc.WithHeaders(headers),
		}
		compression := os.Getenv(OTLPCompression)
		if compression == "gzip" {
			options = append(options, otlptracegrpc.WithCompressor(compression))
		}
		exp, err = otlptracegrpc.New(context.Background(), options...)

	} else if protocol == "http" {
		options := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(host),
			otlptracehttp.WithHeaders(headers),
		}
		compression := os.Getenv(OTLPCompression)
		if compression == "gzip" {
			options = append(options, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
		}
		path := os.Getenv(OTLPPath)
		if path != "" {
			options = append(options, otlptracehttp.WithURLPath(path))
		}
		exp, err = otlptracehttp.New(context.Background(), options...)
	}
	if err != nil {
		return err
	}

	opts := []tracesdk.TracerProviderOption{
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(ratio))),
		tracesdk.WithResource(resource.NewSchemaless(
			semconv.ServiceNameKey.String(serverName),
		)),
	}

	if exp != nil {
		opts = append(opts, tracesdk.WithBatcher(exp))
	} else {
		fmt.Println("do not initial export server")
	}

	tp := tracesdk.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	return nil
}
