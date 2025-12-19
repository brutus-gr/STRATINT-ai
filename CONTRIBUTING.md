# Contributing to STRATINT

Thank you for your interest in contributing to STRATINT! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Submitting Changes](#submitting-changes)
- [Code Style](#code-style)
- [Testing](#testing)
- [Documentation](#documentation)

## Code of Conduct

Please be respectful and constructive in all interactions. We aim to foster an inclusive and welcoming community.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Create a feature branch from `main`
4. Make your changes
5. Submit a pull request

## Development Setup

### Prerequisites

- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Playwright (for scraping)

### Initial Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/STRATINT.git
cd STRATINT

# Install dependencies
go mod download
cd web && npm install && cd ..

# Set up environment
cp .env.example .env
# Edit .env with your configuration

# Run database migrations
psql $DATABASE_URL -f migrations/*.sql

# Start development server
make run

# In another terminal, start frontend
cd web && npm run dev
```

## Making Changes

### Branch Naming

Use descriptive branch names:
- `feature/add-twitter-connector`
- `fix/rss-parsing-error`
- `docs/improve-readme`
- `refactor/database-layer`

### Commit Messages

Write clear, concise commit messages:

```
Add RSS feed export feature

- Implement RSS 2.0 handler
- Add dynamic URL generation from request
- Update API documentation
```

Guidelines:
- Use imperative mood ("Add feature" not "Added feature")
- First line should be 50 chars or less
- Provide detailed description after blank line if needed
- Reference issues: "Fixes #123"

## Submitting Changes

1. **Update your fork**: Sync with upstream before creating PR
   ```bash
   git remote add upstream https://github.com/brutus-gr/STRATINT.git
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run tests**: Ensure all tests pass
   ```bash
   make test
   cd web && npm run build
   ```

3. **Format code**:
   ```bash
   make fmt
   ```

4. **Create Pull Request**:
   - Use descriptive title
   - Reference related issues
   - Describe what changed and why
   - Include test plan
   - Add screenshots for UI changes

### Pull Request Template

```markdown
## Description
Brief description of changes

## Motivation
Why is this change needed?

## Changes Made
- Change 1
- Change 2

## Test Plan
How to verify the changes work

## Checklist
- [ ] Tests pass locally
- [ ] Code formatted with `make fmt`
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
```

## Code Style

### Go

- Follow standard Go conventions
- Run `gofmt` and `golangci-lint`
- Keep functions focused and testable
- Add comments for exported functions
- Use meaningful variable names

```go
// Good
func (c *RSSConnector) FetchArticles(ctx context.Context, feedURL string) ([]models.Source, error) {
    // Implementation
}

// Avoid
func (c *RSSConnector) fa(ctx context.Context, u string) ([]models.Source, error) {
    // Implementation
}
```

### TypeScript/React

- Use TypeScript strict mode
- Functional components with hooks
- Follow existing component patterns
- Use meaningful prop and variable names
- Keep components focused

```tsx
// Good
interface EventCardProps {
  event: Event;
  onSelect: (id: string) => void;
}

export function EventCard({ event, onSelect }: EventCardProps) {
  // Implementation
}
```

### General Guidelines

- **DRY**: Don't Repeat Yourself
- **KISS**: Keep It Simple, Stupid
- **YAGNI**: You Aren't Gonna Need It
- Write self-documenting code
- Add comments for complex logic
- Keep functions under 50 lines when possible

## Testing

### Go Tests

```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/enrichment/

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...
```

### Writing Tests

```go
func TestRSSConnector_FetchArticles(t *testing.T) {
    tests := []struct {
        name    string
        feedURL string
        want    int
        wantErr bool
    }{
        {
            name:    "valid feed",
            feedURL: "https://example.com/feed.rss",
            want:    10,
            wantErr: false,
        },
        // More test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Frontend Tests

```bash
cd web
npm run test  # If configured
npm run build # Verify TypeScript compilation
```

## Documentation

- Update README.md for user-facing changes
- Add inline comments for complex logic
- Update API documentation for endpoint changes
- Create design docs for major features
- Update CHANGELOG.md (if exists)

### Documentation Structure

- **README.md**: Overview, quick start, features
- **ARCHITECTURE.md**: System design, data flow
- **API_DOCS.md**: Endpoint specifications
- **DEPLOYMENT.md**: Production deployment guides
- **Module READMEs**: Package-specific documentation

## Areas for Contribution

### High Priority

- Additional data source connectors (Reddit, Telegram, etc.)
- Enhanced AI prompts and enrichment
- Performance optimizations
- Test coverage improvements
- Documentation improvements

### Good First Issues

Look for issues labeled:
- `good-first-issue`
- `documentation`
- `help-wanted`

### Feature Ideas

- Advanced filtering and search
- Export formats (JSON, CSV, etc.)
- Webhook notifications
- Additional AI providers
- Mobile-responsive improvements
- Multi-language support

## Questions?

- Open an issue for bugs or feature requests
- Tag maintainers for questions
- Check existing issues and PRs first

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing to STRATINT! 🚀
