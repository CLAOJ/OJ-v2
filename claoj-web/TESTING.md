# CLAOJ Web - Test Setup and Configuration

## Running Tests

```bash
# Install dependencies
npm install

# Run tests
npm test

# Run tests with coverage
npm test -- --coverage

# Run tests in watch mode
npm test -- --watch
```

## Test File Organization

- `__tests__/components/` - Component unit tests
- `__tests__/hooks/` - Custom hook tests
- `__tests__/contexts/` - Context provider tests
- `__tests__/utils/` - Utility function tests
- `__tests__/__mocks__/` - Mock implementations
