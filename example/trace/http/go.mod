module github.com/GoogleCloudPlatform/opentelemetry-operations-go/example/trace/http

go 1.13

replace github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace => ../../../exporter/trace

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.13.1
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v0.1.0
	go.opencensus.io v0.22.3
	go.opentelemetry.io/otel v0.5.0
	google.golang.org/grpc v1.27.1
)
