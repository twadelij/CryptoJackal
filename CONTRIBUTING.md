# Contributing to CryptoJackal

## Security Guidelines

**NO hardcoded secrets.** All secrets, passwords, API keys, and private keys must be loaded from environment variables or secure storage. Never commit:
- Private keys
- API keys
- Passwords
- JWT secrets
- Database credentials

**NO backdoors.** Any code that bypasses authentication, allows unauthorized access, or exfiltrates data will be rejected immediately.

**Input validation.** All API endpoints must validate input:
- Use Gin's `binding` tags for required fields
- Validate numeric ranges (e.g., amount > 0, slippage 0-100)
- Whitelist string parameters (e.g., chain names)
- Validate Ethereum addresses (0x + 40 hex chars)

**Error handling.** Never ignore errors:
- Always check `if err != nil`
- Log errors with context
- Don't leak internal error details to the client

## Code Review Checklist

Before submitting a PR, verify:
- [ ] `go vet ./...` passes
- [ ] `gosec ./...` passes with zero issues
- [ ] `go test ./internal/...` passes
- [ ] No hardcoded secrets in code
- [ ] No `fmt.Println` or `log.Fatal` in production paths
- [ ] All API endpoints validate input
- [ ] Errors are handled, not ignored
- [ ] New code has tests

## Pull Request Process

1. Create a feature branch: `git checkout -b feat/your-feature`
2. Make your changes with tests
3. Run the security checks locally:
   ```bash
   go vet ./...
   gosec ./...
   go test ./internal/...
   ```
4. Commit with clear messages: `feat: add input validation to trade endpoint`
5. Push and open a PR
6. CI must pass before merge

## Reporting Security Issues

If you find a security vulnerability, please email the maintainer directly instead of opening a public issue.
