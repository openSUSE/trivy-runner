.PHONY: all webapi pullworker scanworker pushworker sbomworker getsizeworker test format lint  redisexporter

all: webapi pullworker scanworker pushworker getsizeworker sbomworker redisexporter

test:
	go test ./...

format:
	go fmt ./...

redisexporter:
	go build -o ./bin/redis_exporter ./cmd/redisexporter

webapi:
	go build -o ./bin/webapi ./cmd/webapi

pullworker:
	go build -o ./bin/pull_worker ./cmd/pullworker

scanworker:
	go build -o ./bin/scan_worker ./cmd/scanworker

pushworker:
	go build -o ./bin/push_worker ./cmd/pushworker

getsizeworker:
	go build -o ./bin/getsize_worker ./cmd/getsizeworker

sbomworker:
	go build -o ./bin/sbom_worker ./cmd/sbomworker

integration-server:
	-docker network create shared_network
	docker-compose -f docker-compose.yml -f docker-compose-integration.yml up

lint:
	docker run --rm -v "$(CURDIR):/app" -w /app golangci/golangci-lint:v1.55.2 golangci-lint run -v

helm-template:
	@helm template $(HELM_ARGS) "trivy-runner" helm-chart/
