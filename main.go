package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	"github.com/rinsecrm/store-service/core/logging"
	"github.com/rinsecrm/store-service/internal/canaryctx"
	"github.com/rinsecrm/store-service/internal/data"
	"github.com/rinsecrm/store-service/internal/metrics"
	"github.com/rinsecrm/store-service/internal/server"
	"github.com/rinsecrm/store-service/internal/tracing"
	pb "github.com/rinsecrm/store-service/proto/go"
)

var (
	name    = "store-service"
	version = "dev"
)

// Config holds the application configuration
type Config struct {
	Port            int    `envconfig:"PORT" default:"8080"`
	MetricsPort     int    `envconfig:"METRICS_PORT" default:"9090"`
	DynamoTableName string `envconfig:"DYNAMODB_TABLE_NAME" required:"true"`
	Region          string `envconfig:"AWS_REGION" default:"us-east-1"`
	LocalDebug      bool   `envconfig:"LOCAL_DEBUG" default:"false"`
	DynamoEndpoint  string `envconfig:"DYNAMODB_ENDPOINT" default:""`
	TempoHost       string `envconfig:"TEMPO_HOST" default:""`
}

func main() {
	// Initialize logging
	logging.SetStandardFields(name, version)

	// Load environment configuration
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		logging.WithError(err).Fatal("Failed to process config")
	}

	// Validate required configuration
	if cfg.DynamoTableName == "" {
		logging.Fatal("DYNAMODB_TABLE_NAME environment variable is required")
	}

	// Set logging level based on LocalDebug
	if cfg.LocalDebug {
		logging.SetLevel(logrus.DebugLevel)
		logging.Info("Debug logging enabled")
	}

	// Initialize tracing
	if err := tracing.Start(tracing.Config{
		ServiceName: name,
		TempoHost:   cfg.TempoHost,
		Version:     version,
	}); err != nil {
		logging.WithError(err).Error("Failed to initialize tracing")
	}

	logging.WithFields(logrus.Fields{
		"port":            cfg.Port,
		"metrics_port":    cfg.MetricsPort,
		"dynamo_table":    cfg.DynamoTableName,
		"region":          cfg.Region,
		"local_debug":     cfg.LocalDebug,
		"dynamo_endpoint": cfg.DynamoEndpoint,
	}).Info("Starting store service with configuration")

	// Initialize AWS DynamoDB client
	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logging.WithError(err).Fatal("Failed to load AWS config")
	}

	// Override endpoint for local development
	var dynamoClient *dynamodb.Client
	if cfg.DynamoEndpoint != "" {
		dynamoClient = dynamodb.NewFromConfig(awsConfig, func(o *dynamodb.Options) {
			o.BaseEndpoint = &cfg.DynamoEndpoint
		})
		logging.WithField("endpoint", cfg.DynamoEndpoint).Info("Using custom DynamoDB endpoint")
	} else {
		dynamoClient = dynamodb.NewFromConfig(awsConfig)
	}

	// Start metrics server
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.MetricsPort),
		Handler: metrics.MetricsHandler(),
	}
	go func() {
		logging.WithField("port", cfg.MetricsPort).Info("Starting metrics server")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.WithError(err).Error("Metrics server error")
		}
	}()

	// Initialize store
	storeService := data.NewDynamoStore(dynamoClient, cfg.DynamoTableName)

	// Create gRPC server with canary, metrics, and tracing interceptors
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			canaryctx.UnaryServerInterceptor(),
			metrics.UnaryServerInterceptor(),
		),
	)

	// Register the store service
	pb.RegisterStoreServiceServer(grpcServer, server.NewStoreServiceServer(storeService))

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		logging.WithError(err).WithField("port", cfg.Port).Fatal("Failed to listen")
	}

	logging.WithFields(logrus.Fields{
		"port":         cfg.Port,
		"dynamo_table": cfg.DynamoTableName,
	}).Info("Store service listening")

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logging.Info("Shutting down store service...")

		// Shutdown metrics server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsServer.Shutdown(ctx); err != nil {
			logging.WithError(err).Error("Failed to shutdown metrics server")
		}

		// Shutdown tracing
		if err := tracing.Stop(ctx); err != nil {
			logging.WithError(err).Error("Failed to shutdown tracing")
		}

		grpcServer.GracefulStop()
	}()

	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		logging.WithError(err).Fatal("Failed to serve")
	}
}
