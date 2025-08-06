# Featury

Feature flag management system built with Go and Gin - Backend API, CLI tool, and SDK.

## Overview

Featury is a comprehensive feature flag management solution that consists of:

- **API Server**: RESTful API built with Gin framework for managing feature flags
- **CLI Tool**: Command-line interface for developers and operators to manage features
- **SDK**: Go client library for applications to consume feature flags

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Make (optional, for convenience commands)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/tig-company/featury.git
cd featury
```

2. Install dependencies:
```bash
go mod download
```

3. Build the project:
```bash
make build
```

### Running the API Server

```bash
# Using Make
make run-api

# Or directly with Go
go run ./cmd/api
```

The API server will start on `http://localhost:8080` by default.

### Using the CLI

```bash
# Using the built binary
./bin/featury --help

# Or with Go
go run ./cmd/cli --help

# Examples
./bin/featury feature list
./bin/featury feature create my-feature
```

### Using the SDK

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/tig-company/featury/pkg/sdk"
)

func main() {
    client := sdk.NewClient(sdk.ClientOptions{
        BaseURL: "http://localhost:8080",
        APIKey:  "your-api-key",
    })
    
    enabled, err := client.IsFeatureEnabled("my-feature", "user-123")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Feature enabled: %v\n", enabled)
}
```

## API Endpoints

- `GET /health` - Health check
- `GET /api/v1/features` - List all features
- `POST /api/v1/features` - Create a new feature
- `GET /api/v1/features/:id` - Get feature by ID
- `PUT /api/v1/features/:id` - Update feature
- `DELETE /api/v1/features/:id` - Delete feature

## Project Structure

```
├── cmd/
│   ├── api/           # API server application
│   └── cli/           # CLI application
├── pkg/
│   └── sdk/           # SDK package
├── internal/
│   ├── api/           # API handlers and routes
│   ├── config/        # Configuration management
│   ├── models/        # Data models
│   ├── repository/    # Data access layer
│   └── service/       # Business logic
├── web/               # Static web assets (if any)
├── docs/              # Documentation
├── scripts/           # Build and deployment scripts
└── test/              # Integration tests
```

## Development

### Available Make Commands

```bash
make help           # Show available commands
make build          # Build all binaries
make run-api        # Run the API server
make test           # Run tests
make test-coverage  # Run tests with coverage
make lint           # Run linter
make clean          # Clean build artifacts
```

### Configuration

The application can be configured using:

- Environment variables
- YAML configuration file (`config.yaml`)
- Command-line flags (for CLI)

Default configuration:
- Port: `8080`
- Database: `featury.db`
- Log Level: `info`

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support, please open an issue in the GitHub repository or contact the development team.