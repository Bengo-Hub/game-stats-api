# Contributing to Game Stats

Thank you for your interest in contributing to Game Stats! This document provides guidelines for contributing to the project.

## Code of Conduct

This project adheres to a code of conduct. By participating, you agree to uphold this code. See [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md).

## Development Workflow

### 1. Fork and Clone
```bash
git clone https://github.com/yourusername/game-stats.git
cd game-stats
```

### 2. Create a Branch
```bash
git checkout -b feature/your-feature-name
```

Branch naming conventions:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation
- `refactor/` - Code refactoring
- `test/` - Test additions

### 3. Make Changes
- Write clean, readable code
- Follow existing code style
- Add tests for new features
- Update documentation

### 4. Commit
Follow [Conventional Commits](https://www.conventionalcommits.org/):
```bash
git commit -m "feat(api): add game timer endpoint"
git commit -m "fix(ui): resolve score display bug"
git commit -m "docs: update API documentation"
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

### 5. Push and Create PR
```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Code Style

### Go (Backend)
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` and `golangci-lint`
- Write tests for all new code
- Target 80%+ coverage

### TypeScript/React (Frontend)
- Follow [Airbnb Style Guide](https://github.com/airbnb/javascript)
- Use ESLint and Prettier
- Write TypeScript strict mode
- Add JSDoc for complex functions

## Testing

### Backend
```bash
cd games-stats-api
make test              # Unit tests
make test-integration  # Integration tests
make test-coverage     # Coverage report
```

### Frontend
```bash
cd game-stats-ui
npm test              # Unit tests
npm run test:e2e      # E2E tests
```

## Pull Request Process

1. **Update Documentation** - If you change APIs or add features
2. **Add Tests** - Ensure tests pass and coverage doesn't decrease
3. **Update CHANGELOG** - Add your changes under "Unreleased"
4. **Request Review** - Tag maintainers for review
5. **Address Feedback** - Make requested changes
6. **Squash Commits** - Before merging (if requested)

## Pull Request Template

Your PR should include:
- Clear description of changes
- Link to related issues
- Screenshots (for UI changes)
- Test results
- Checklist confirmation

## Development Setup

See [README.md](./README.md) for full setup instructions.

## Questions?

- Open an issue for bugs or feature requests
- Join our Discord for discussions
- Email: support@gamestats.com

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
