package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/vpereira/trivy_runner/internal/metrics"
	"github.com/vpereira/trivy_runner/internal/redisutil"
	"go.uber.org/zap"
)

type fakeNotifier struct {
	Enabled bool
	Tags    map[string]string
}

func (fn *fakeNotifier) NotifySentry(err error) {}

func (fn *fakeNotifier) AddTag(name string, value string) {
	fn.Tags[name] = value
}

func TestProcessQueue(t *testing.T) {
	logger, _ = zap.NewProduction()
	// Mock Redis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	os.Setenv("REDIS_HOST", mr.Host())
	os.Setenv("REDIS_PORT", mr.Port())
	os.Setenv("IMAGES_APP_DIR", "/tmp")

	// Initialize Redis client and push a mock entry
	rdb = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	gun := "registry.suse.com/bci/bci-busybox:latest"

	_, err = rdb.RPush(ctx, "topull", gun).Result()
	if err != nil {
		t.Fatal(err)
	}

	imagesAppDir = redisutil.GetEnv("IMAGES_APP_DIR", "/app/images")

	prometheusMetrics = metrics.NewMetrics(
		prometheus.CounterOpts{
			Name: "pullworker_processed_ops_total",
			Help: "Total number of processed operations by the pullworker.",
		},
		prometheus.CounterOpts{
			Name: "pullworker_processed_errors_total",
			Help: "Total number of processed errors by the pullworker.",
		},
		commandExecutionHistogram,
	)

	prometheusMetrics.Register()
	mockNotifier := &fakeNotifier{
		Tags: make(map[string]string),
	}
	sentryNotifier = mockNotifier

	// Ensure to unregister metrics to avoid pollution across tests
	defer prometheus.Unregister(prometheusMetrics.ProcessedOpsCounter)
	defer prometheus.Unregister(prometheusMetrics.ProcessedErrorsCounter)
	defer prometheus.Unregister(commandExecutionHistogram)

	processQueue()

	value, ok := mockNotifier.Tags["image.name"]

	if !ok {
		t.Errorf("Sentry tag %s not set", "gun")
	}

	if value != gun {
		t.Errorf("Sentry tag %s does not match. Want %s got %s", "gun", "registry.suse.com/bci/bci-busybox:latest", value)
	}
}

func TestGenerateSkopeoCmdArgs(t *testing.T) {
	// Define test cases
	tests := []struct {
		name           string
		imageName      string
		targetDir      string
		envUsername    string
		envPassword    string
		expectedResult []string
	}{
		{
			name:      "without credentials",
			imageName: "registry.example.com/myimage:latest",
			targetDir: "/tmp/targetdir",
			expectedResult: []string{
				"copy", "--remove-signatures",
				"docker://registry.example.com/myimage:latest",
				"docker-archive:///tmp/targetdir",
			},
		},
		{
			name:        "with credentials",
			imageName:   "registry.example.com/myimage:latest",
			targetDir:   "/tmp/targetdir",
			envUsername: "testuser",
			envPassword: "testpass",
			expectedResult: []string{
				"copy", "--remove-signatures",
				"--src-username", "testuser", "--src-password", "testpass",
				"docker://registry.example.com/myimage:latest",
				"docker-archive:///tmp/targetdir",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment variables if needed
			if tc.envUsername != "" && tc.envPassword != "" {
				os.Setenv("REGISTRY_USERNAME", tc.envUsername)
				os.Setenv("REGISTRY_PASSWORD", tc.envPassword)
				defer os.Unsetenv("REGISTRY_USERNAME")
				defer os.Unsetenv("REGISTRY_PASSWORD")
			}

			// Call the method under test
			result := GenerateSkopeoCmdArgs(tc.imageName, tc.targetDir)

			// Verify the result
			if !reflect.DeepEqual(result, tc.expectedResult) {
				t.Errorf("GenerateSkopeoCmdArgs(%s, %s) got %v, want %v", tc.imageName, tc.targetDir, result, tc.expectedResult)
			}
		})
	}
}
