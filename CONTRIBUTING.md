# Contributing to Switchboard Gateway

Thank you for your interest in contributing! 🎉

## Development Workflow

### 1. Fork & Clone
```bash
git clone https://github.com/YOUR_USERNAME/switchboard-gateway.git
cd switchboard-gateway
```

### 2. Create Feature Branch
```bash
git checkout -b feature/your-feature-name
# OR
git checkout -b fix/bug-description
```

### 3. Make Changes
- Write code following our style guide
- Add tests for new functionality
- Update documentation

### 4. Run Tests
```bash
make test
make lint
```

### 5. Commit
```bash
git add .
git commit -m "feat: add new feature"
```

**Commit Message Format:**
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding tests
- `refactor:` Code refactoring
- `perf:` Performance improvements

### 6. Push & Create PR
```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Code Style

- Follow Go best practices
- Use `gofmt` for formatting
- Add comments to exported functions
- Write tests for new code

## Questions?

Open an issue or discussion on GitHub!