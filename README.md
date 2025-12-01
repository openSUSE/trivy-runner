# Trivy Runner

## Description
Trivy Runner is a Go-based web application designed to scan Docker images using Trivy and provide vulnerability reports. This tool simplifies the process of scanning container images for security vulnerabilities by exposing a web API.

## Getting Started

### Prerequisites
- Go (version 1.19 or later)
- Docker

### Installation

To set up the Trivy Runner on your local machine, follow these steps:

1. Clone the repository:
   ```
   git clone git@github.com:openSUSE/trivy-runner.git
   ```
2. Navigate to the project directory:
   ```
   cd trivy-runner
   ```
3. Build the application:
   ```
    make
   ```

## Usage

`docker-compose build` # to build it

`docker-compose up` # to start it

Then if you want to scan a new image:

```
curl "http://localhost:8080/scan?image=registry.suse.com/bci/bci-busybox:latest"
```
## Integration with registry catalog

If you want to test integration together with registry catalog

- Check/adapt the `docker-compose-integration.yml` file
- run `make integration-server`


## Features

- Scan Docker images for vulnerabilities using Trivy.
- Real-time logging of scanning process.
