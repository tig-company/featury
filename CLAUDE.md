# Claude Development Guide for Featury

This document provides context and guidelines for Claude when working on the Featury project.

## Project Overview

Featury is a feature flag management system built with Go and Gin framework. It consists of three main components:

1. **API Server** (`cmd/api/`): REST API for managing feature flags
2. **CLI Tool** (`cmd/cli/`): Command-line interface for feature management
3. **SDK** (`pkg/sdk/`): Go client library for consuming feature flags

## Architecture

### Directory Structure
```
featury/
├── cmd/                    # Application entry points
│   ├── api/               # API server main
│   └── cli/               # CLI tool main
├── pkg/                   # Public packages
│   └── sdk/               # SDK for external use
├── internal/              # Private application code
│   ├── api/               # HTTP handlers and routes
│   ├── config/            # Configuration management
│   ├── models/            # Data models and structs
│   ├── repository/        # Data access layer
│   └── service/           # Business logic layer
├── web/                   # Static assets
├── docs/                  # Documentation
├── scripts/               # Build/deployment scripts
└── test/                  # Integration tests
```

### Key Dependencies
- **Gin** (`github.com/gin-gonic/gin`): HTTP web framework
- **Cobra** (`github.com/spf13/cobra`): CLI framework
- **Viper** (`github.com/spf13/viper`): Configuration management

## Development Guidelines

### Code Style
- Follow standard Go conventions and idioms
- Use meaningful variable and function names
- Add appropriate error handling
- Keep functions small and focused
- Use interfaces for better testability

### API Design
- Follow RESTful principles
- Use proper HTTP status codes
- Include appropriate error responses
- Validate input data
- Support pagination for list endpoints

### CLI Design
- Use clear, descriptive command names
- Provide helpful error messages
- Support common flags (--help, --verbose, etc.)
- Follow UNIX command-line conventions

### Testing
- Write unit tests for business logic
- Include integration tests for API endpoints
- Test error scenarios
- Aim for good test coverage

### Configuration
The application supports configuration through:
- Environment variables (highest priority)
- YAML config files
- Default values (lowest priority)

Key configuration options:
- `PORT`: API server port (default: 8080)
- `DATABASE`: Database connection string
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

## Common Tasks

### Building the Project
```bash
# Build all components
make build

# Build specific components
make build-api    # API server
make build-cli    # CLI tool
```

### Running Components
```bash
# Run API server
make run-api
go run ./cmd/api

# Run CLI
make run-cli
go run ./cmd/cli [command]
```

### Testing
```bash
# Run all tests
make test
go test ./...

# Run tests with coverage
make test-coverage
```

### Adding New Features

#### API Endpoints
1. Add route in `internal/api/routes.go`
2. Implement handler function
3. Add request/response models in `internal/models/`
4. Add business logic in `internal/service/`
5. Add data access in `internal/repository/` if needed

#### CLI Commands
1. Add command definition in `cmd/cli/main.go`
2. Implement command logic
3. Add appropriate flags and validation
4. Update help text

#### SDK Methods
1. Add client method in `pkg/sdk/client.go`
2. Define request/response structures
3. Add proper error handling
4. Update SDK documentation

### Database Considerations
- Currently designed to support SQLite for development
- Structure allows for easy database abstraction
- Consider using database/sql with appropriate drivers
- Repository pattern isolates data access logic

### Error Handling
- Use structured error responses in API
- Provide meaningful error messages in CLI
- Log errors appropriately based on severity
- Return proper HTTP status codes

### Security Considerations
- Validate all input data
- Use proper authentication/authorization
- Sanitize data before database operations
- Never log sensitive information
- Use HTTPS in production

## Useful Commands

### Development
```bash
# Install development tools
go install github.com/cosmtrek/air@latest           # Hot reload
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest  # Linting

# Hot reload during development
make dev  # Uses air for hot reload

# Linting
make lint
```

### Build and Deployment
```bash
# Clean build artifacts
make clean

# Update dependencies
make deps-update

# Install binaries to $GOPATH/bin
make install
```

## Troubleshooting

### Common Issues
1. **Port already in use**: Change the PORT environment variable or config
2. **Module not found**: Run `go mod download` to fetch dependencies
3. **Permission denied**: Ensure binary has execute permissions
4. **Config not found**: Check config file path and format

### Debugging
- Use `-v` flag for verbose output in CLI
- Check logs for API server issues
- Use Go's built-in debugging tools
- Add temporary log statements for complex issues

## Future Enhancements

### Planned Features
- Database persistence (currently using in-memory storage)
- User authentication and authorization
- Feature flag rules and targeting
- Metrics and analytics
- Web UI dashboard
- Multi-environment support

### Technical Debt
- Add comprehensive unit tests
- Implement proper database layer
- Add API documentation (OpenAPI/Swagger)
- Set up CI/CD pipeline
- Add monitoring and logging infrastructure

## Notes for Claude

- Always run tests before committing changes
- Follow the existing code patterns and conventions
- Update documentation when adding new features
- Consider backwards compatibility when making changes
- Use the Makefile commands for common tasks
- Check that the API server starts correctly after changes
- Verify CLI commands work as expected
- Test SDK integration in sample applications