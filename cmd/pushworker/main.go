package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/vpereira/trivy_runner/internal/airbrake"
	"github.com/vpereira/trivy_runner/internal/error_handler"
	"github.com/vpereira/trivy_runner/internal/metrics"
	"github.com/vpereira/trivy_runner/internal/pushworker"
	"github.com/vpereira/trivy_runner/internal/redisutil"
	"github.com/vpereira/trivy_runner/internal/sentry"
	"go.uber.org/zap"
)

var (
	ctx               = context.Background()
	rdb               *redis.Client
	logger            *zap.Logger
	airbrakeNotifier  *airbrake.AirbrakeNotifier
	sentryNotifier    sentry.Notifier
	errorHandler      *error_handler.ErrorHandler
	prometheusMetrics *metrics.Metrics
)

func init() {
	var err error
	logger, err = zap.NewProduction()

	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	airbrakeNotifier = airbrake.NewAirbrakeNotifier()

	if airbrakeNotifier == nil {
		logger.Error("Failed to create airbrake notifier")
	}

	sentryNotifier = sentry.NewSentryNotifier()

	if sentryNotifier == nil {
		logger.Error("Failed to create sentry notifier")
	}

	prometheusMetrics = metrics.NewMetrics(
		prometheus.CounterOpts{
			Name: "pushworker_processed_ops_total",
			Help: "Total number of processed operations by the pushworker.",
		},
		prometheus.CounterOpts{
			Name: "pushworker_processed_errors_total",
			Help: "Total number of processed errors by the pushworker.",
		},
	)
}

func main() {

	defer logger.Sync()

	prometheusMetrics.Register()

	errorHandler = error_handler.NewErrorHandler(logger, prometheusMetrics.ProcessedErrorsCounter, airbrakeNotifier, sentryNotifier)

	webhookURL := os.Getenv("WEBHOOK_URL")

	if webhookURL == "" {
		logger.Error("WEBHOOK_URL environment variable is not set")
	}

	rdb = redisutil.InitializeClient()

	go metrics.StartMetricsServer("8083")

	for {
		processQueue(webhookURL)
	}
}

func processQueue(webhookURL string) {
	answer, err := rdb.BRPop(ctx, 0, "topush").Result()

	if err != nil {
		errorHandler.Handle(err)
		return
	}

	item := answer[1]

	var dto pushworker.DTO
	err = json.Unmarshal([]byte(item), &dto)

	if err != nil {
		logger.Info("Could not Unmarshal dto", zap.String("item", item), zap.Error(err))
		errorHandler.Handle(err)
		return
	}

	payload := pushworker.NewPayload()
	payload.Operation = dto.Operation
	payload.Image = dto.Image
	payload.Sizes = dto.Sizes

	if dto.ResultFilePath != "" {
		scanResults, err := extractResults(dto.ResultFilePath)
		if err != nil {
			logger.Info("Could not extract scan results", zap.String("item", item), zap.Error(err))
			errorHandler.Handle(err)
			return
		}

		payload.Results = scanResults
		payload.RanAt = time.Now().Format(time.RFC3339)
	}

	go sendToWebhook(webhookURL, payload)
}

func extractResults(filePath string) (json.RawMessage, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		errorHandler.Handle(err)
		return nil, err
	}

	// unmarshal the data
	var result pushworker.ScanPayload
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result.Results, nil
}

func sendToWebhook(webhookURL string, result interface{}) {
	jsonData, err := json.Marshal(result)
	imageName := reflect.TypeOf(result).Name()

	if err != nil {
		errorHandler.Handle(err)
		return
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))

	if err != nil {
		errorHandler.Handle(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		errorHandler.Handle(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorHandler.Handle(err)
		return
	}
	logger.Info("Report sent successfully for image:", zap.String("image", imageName))
	prometheusMetrics.IncOpsProcessed()
}
