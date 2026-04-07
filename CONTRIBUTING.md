# Contributing to Autonomous Polymorphic Agent

Thank you for your interest in contributing to the Autonomous Polymorphic Agent (APA) project! We welcome contributions from the community and are excited to work with you.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Process](#development-process)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Pull Request Process](#pull-request-process)
- [Community](#community)

## 🤝 Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](docs/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [founder@nbr.company](mailto:founder@nbr.company).

## 🚀 Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- Docker (optional, for container testing)

### Setting Up Your Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/APA.git
   cd APA
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/naviNBRuas/APA.git
   ```
4. Install dependencies:
   ```bash
   go mod tidy
   ```
5. Run tests to verify setup:
   ```bash
   go test ./...
   ```

## 💡 How to Contribute

### Types of Contributions

We welcome various types of contributions:

- **Bug Reports**: Help us identify and fix issues
- **Feature Requests**: Suggest new capabilities
- **Code Contributions**: Implement new features or fixes
- **Documentation**: Improve guides, examples, and comments
- **Testing**: Add test cases and improve coverage
- **Security Reviews**: Identify potential vulnerabilities

### Good First Issues

Look for issues tagged with [`good first issue`](https://github.com/naviNBRuas/APA/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) for beginner-friendly tasks.

## 🛠️ Development Process

### Branch Strategy

- `main`: Production-ready code
- Feature branches: For new features and bug fixes
- Release branches: For preparing releases

### Development Workflow

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```
2. Make your changes
3. Write tests for new functionality
4. Update documentation as needed
5. Run all tests:
   ```bash
   go test -v -race ./...
   ```
6. Ensure code quality:
   ```bash
   go fmt ./...
   go vet ./...
   ```
7. Commit your changes with descriptive messages
8. Push to your fork
9. Create a pull request

## 📝 Coding Standards

### Go Standards

- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting
- Run `go vet` for static analysis
- Maintain >80% test coverage

### Naming Conventions

- Use descriptive names for variables, functions, and types
- Follow Go naming conventions (CamelCase for exported identifiers)
- Use meaningful package names

### Code Organization

- Keep functions focused and small
- Use interfaces for abstraction
- Separate concerns into different packages
- Document exported functions and types

### Error Handling

- Always handle errors appropriately
- Use descriptive error messages
- Consider using error wrapping with `%w`
- Log errors with appropriate context

## 🧪 Testing

### Test Requirements

- All new code must include unit tests
- Integration tests for complex functionality
- Race condition detection enabled
- Performance benchmarks for critical paths

### Running Tests

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package
go test ./pkg/agent/...
```

### Writing Good Tests

- Test one thing at a time
- Use table-driven tests for multiple cases
- Include edge cases and error conditions
- Make tests independent and reproducible
- Use meaningful test names

## 📚 Documentation

### Code Documentation

- Document all exported functions, types, and variables
- Use Godoc-style comments
- Include examples for complex functionality
- Keep documentation up to date with code changes

### Project Documentation

- Update README.md for user-facing changes
- Add examples to the examples/ directory
- Update architectural documentation in docs/
- Keep CHANGELOG.md updated with notable changes

## 🔄 Pull Request Process

### Before Submitting

1. Ensure all tests pass
2. Update documentation
3. Squash commits into logical units
4. Write a clear, descriptive PR title
5. Include a detailed description of changes

### PR Description Template

```markdown
## What does this PR do?

Brief description of the changes.

## Why is this change needed?

Motivation and context.

## How was this tested?

- [ ] Unit tests
- [ ] Integration tests
- [ ] Manual testing
- [ ] Other: _____

## Checklist

- [ ] Code follows project standards
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] No new security vulnerabilities
- [ ] All CI checks pass
```

### Review Process

1. Automated checks run on all PRs
2. Maintainers review code quality and functionality
3. Security review for sensitive changes
4. Performance review for critical paths
5. Documentation review for user-facing changes

## 👥 Community

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General discussion and Q&A
- **Email**: [founder@nbr.company](mailto:founder@nbr.company)

### Recognition

Contributors will be recognized in:
- README.md contributors section
- Release notes
- Project documentation

## 🛡️ Security

### Reporting Security Issues

Please report security vulnerabilities to [founder@nbr.company](mailto:founder@nbr.company) rather than public GitHub issues.

### Security Best Practices

- Follow secure coding practices
- Keep dependencies updated
- Run security scanners regularly
- Review code for common vulnerabilities

## 📎 Additional Resources

- [Project Documentation](docs/)
- [API Reference](https://navinbruas.github.io/APA/)
- [Code of Conduct](docs/CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)

Thank you for contributing to the Autonomous Polymorphic Agent project!