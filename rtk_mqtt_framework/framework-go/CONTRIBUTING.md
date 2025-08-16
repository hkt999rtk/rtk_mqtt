# Contributing to RTK MQTT Framework Go

We welcome contributions to the RTK MQTT Framework Go implementation! This document provides guidelines for contributing to the project.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git
- Make (for build automation)
- An MQTT broker for testing (we provide a mock broker)

### Development Setup

1. **Fork the repository** on GitHub

2. **Clone your fork:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/rtk_mqtt_framework.git
   cd rtk_mqtt_framework/framework-go
   ```

3. **Set up the upstream remote:**
   ```bash
   git remote add upstream https://github.com/ORIGINAL_OWNER/rtk_mqtt_framework.git
   ```

4. **Install dependencies:**
   ```bash
   make install-deps
   make deps
   ```

5. **Verify the setup:**
   ```bash
   make test
   make build
   ```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

Branch naming conventions:
- `feature/` - New features
- `bugfix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring
- `test/` - Test improvements

### 2. Make Your Changes

- Follow the [Go Code Style](#code-style) guidelines
- Add tests for new functionality
- Update documentation as needed
- Ensure backward compatibility when possible

### 3. Test Your Changes

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run comprehensive test suite
./test.sh

# Run code quality checks
make fmt
make vet
make lint
```

### 4. Commit Your Changes

Use clear, descriptive commit messages following the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```bash
git add .
git commit -m "feat: add new device plugin for smart thermostats"
```

Commit message format:
- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

### 5. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Create a pull request on GitHub with:
- Clear title and description
- Reference to related issues
- List of changes made
- Testing performed
- Screenshots (if applicable)

## Code Style

### Go Style Guidelines

We follow the standard Go style guidelines:

- Use `gofmt` for formatting (run `make fmt`)
- Use `go vet` for static analysis (run `make vet`)
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use meaningful variable and function names
- Write clear, concise comments for public APIs

### Package Organization

```go
// Package declaration
package device

// Imports (standard library first, then third-party, then local)
import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/sirupsen/logrus"
    
    "github.com/rtk/mqtt-framework/pkg/mqtt"
)

// Constants
const (
    DefaultTimeout = 30 * time.Second
)

// Types (interfaces first, then structs)
type Plugin interface {
    Start(ctx context.Context) error
    Stop() error
}

type BasePlugin struct {
    logger *logrus.Logger
}

// Functions (constructors first, then methods)
func NewBasePlugin() *BasePlugin {
    return &BasePlugin{
        logger: logrus.New(),
    }
}
```

### Error Handling

- Use typed errors for different error conditions
- Wrap errors with context using `fmt.Errorf("action failed: %w", err)`
- Return errors as the last return value
- Handle errors at the appropriate level

```go
// Good
func (c *Client) Connect(ctx context.Context) error {
    if err := c.validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    // ...
}

// Custom error types
var (
    ErrNotConnected = &Error{Code: "NOT_CONNECTED", Message: "client not connected"}
)
```

### Interface Design

- Keep interfaces small and focused
- Define interfaces where they are used, not where they are implemented
- Use composition over inheritance

```go
// Good - small, focused interface
type MessageHandler interface {
    HandleMessage(ctx context.Context, msg *Message) error
}

// Good - composed interfaces
type Publisher interface {
    Publish(ctx context.Context, topic string, payload []byte) error
}

type Subscriber interface {
    Subscribe(ctx context.Context, topic string, handler MessageHandler) error
}

type Client interface {
    Publisher
    Subscriber
    Connect(ctx context.Context) error
    Disconnect() error
}
```

## Testing Guidelines

### Test Structure

- Place tests in `*_test.go` files
- Use table-driven tests for multiple test cases
- Test both success and error conditions
- Use meaningful test names

```go
func TestClientConnect(t *testing.T) {
    tests := []struct {
        name        string
        config      *Config
        expectError bool
    }{
        {
            name: "valid config",
            config: &Config{
                BrokerHost: "localhost",
                BrokerPort: 1883,
            },
            expectError: false,
        },
        {
            name: "invalid config",
            config: &Config{
                BrokerHost: "",
            },
            expectError: true,
        },
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            client := NewClient(test.config)
            err := client.Connect(context.Background())
            
            if test.expectError && err == nil {
                t.Error("expected error but got none")
            }
            if !test.expectError && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
```

### Test Requirements

- All new code must include tests
- Aim for >80% test coverage for new packages
- Include integration tests for complex features
- Use mocks for external dependencies
- Test concurrent operations with `-race` flag

### Benchmarks

Add benchmarks for performance-critical code:

```go
func BenchmarkMessageEncode(b *testing.B) {
    codec := NewCodec()
    state := &State{Status: "online", Health: "healthy"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := codec.EncodeState(context.Background(), state)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Documentation

### Code Documentation

- Document all public APIs
- Use Go doc conventions
- Include examples for complex APIs

```go
// Client represents an MQTT client for the RTK framework.
// It provides methods for connecting to brokers, publishing messages,
// and subscribing to topics.
type Client interface {
    // Connect establishes a connection to the MQTT broker.
    // It returns an error if the connection fails.
    Connect(ctx context.Context) error
    
    // Publish sends a message to the specified topic.
    // The QoS and retention settings are specified in the options.
    Publish(ctx context.Context, topic string, payload []byte, opts *PublishOptions) error
}
```

### README Updates

- Update package READMEs for new features
- Add usage examples
- Update the main README if needed

### Changelog

- Add entries to CHANGELOG.md for significant changes
- Follow the format in the existing changelog
- Include breaking changes prominently

## Plugin Development

### Creating New Device Plugins

1. **Implement the Plugin interface:**
   ```go
   type MyPlugin struct {
       *device.BasePlugin
   }
   
   func (p *MyPlugin) GetInfo() *device.Info {
       return &device.Info{
           Name: "My Device",
           Type: "my_device",
           Version: "1.0.0",
       }
   }
   ```

2. **Add configuration support:**
   ```go
   type MyPluginConfig struct {
       Setting1 string `json:"setting1" validate:"required"`
       Setting2 int    `json:"setting2" validate:"min=1"`
   }
   ```

3. **Implement required methods:**
   - `Initialize(ctx context.Context, config json.RawMessage) error`
   - `Start(ctx context.Context) error`
   - `Stop() error`
   - `GetState(ctx context.Context) (*State, error)`
   - `HandleCommand(ctx context.Context, cmd *Command) (*CommandResponse, error)`

4. **Add tests and examples**

### Schema Definitions

When adding new message types or events:

1. **Define the schema in `pkg/codec/schema.go`**
2. **Add validation rules**
3. **Update the schema manager**
4. **Add tests for schema validation**

## Issue Guidelines

### Reporting Bugs

Include the following information:
- Go version
- Operating system
- Steps to reproduce
- Expected behavior
- Actual behavior
- Relevant logs or error messages

### Feature Requests

Provide:
- Clear description of the feature
- Use case and motivation
- Proposed API (if applicable)
- Alternative solutions considered

### Questions

For questions:
- Check existing documentation first
- Search existing issues
- Provide context about what you're trying to achieve

## Review Process

### Pull Request Requirements

- All tests must pass
- Code coverage should not decrease significantly
- Code must be formatted (`make fmt`)
- No linting errors (`make vet`)
- Documentation updated if needed
- CHANGELOG.md updated for significant changes

### Review Criteria

Reviewers will check:
- Code quality and style
- Test coverage and quality
- Documentation completeness
- Backward compatibility
- Performance implications
- Security considerations

## Community Guidelines

### Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Help others learn and grow
- Follow the project's code of conduct

### Communication

- Use clear, professional language
- Provide detailed explanations
- Ask questions when unclear
- Share knowledge and best practices

## Release Process

Maintainers handle releases, but contributors should:
- Follow semantic versioning principles
- Document breaking changes
- Update migration guides if needed
- Test against multiple Go versions

## Getting Help

- Check the documentation in `docs/`
- Look at examples in `examples/`
- Search existing issues
- Create a new issue for bugs or feature requests
- Ask questions in discussions

Thank you for contributing to the RTK MQTT Framework Go implementation!