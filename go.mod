module github.com/rinsecrm/store-service

go 1.25

require (
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.12.12
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.26.6
	github.com/google/uuid v1.5.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/prometheus/client_golang v1.23.0
	github.com/sirupsen/logrus v1.9.3
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.62.0
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.37.0
	go.opentelemetry.io/otel/sdk v1.37.0
	go.opentelemetry.io/otel/trace v1.37.0
	google.golang.org/grpc v1.60.1
	google.golang.org/protobuf v1.32.0
)
