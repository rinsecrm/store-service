# Store Service

A microservice that provides gRPC-based data storage and retrieval for the RinseCRM platform.

## Overview

The Store service handles all data persistence operations via gRPC, providing a centralized data layer for other services in the platform.

## Architecture

- **Language**: Go 1.25
- **Protocol**: gRPC
- **Data Layer**: In-memory storage (can be extended to databases)
- **Canary Support**: Built-in canary routing via `X-Canary` metadata

## Development

### Prerequisites

- Go 1.25 or later
- Docker (for containerization)
- Protocol Buffer compiler (for proto generation)

### Local Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/rinsecrm/store-service.git
   cd store-service
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Generate protobuf code**:
   ```bash
   make proto
   ```

4. **Run tests**:
   ```bash
   make test
   ```

5. **Build the service**:
   ```bash
   make build
   ```

6. **Run locally**:
   ```bash
   ./bin/store-service
   ```

### Docker Development

```bash
# Build Docker image
docker build -f Dockerfile.ci -t store-service:dev .

# Run with Docker Compose (includes API service)
docker-compose up
```

## Developer Workflows

### Pull Request Canaries

When you create or update a Pull Request, the CI/CD system automatically:

1. **Builds a canary Docker image** tagged with `pr-{PR_NUMBER}`
2. **Deploys to integration environment** with canary routing
3. **Creates isolated test environment** for your changes

#### Testing Your PR Canary

Once your PR is deployed, you can test it by adding the `X-Canary` metadata to your gRPC requests:

```bash
# Test your PR canary (replace 123 with your PR number)
grpcurl -H "X-Canary: 123" -plaintext localhost:8080 store.Store/Get
```

#### PR Canary Lifecycle

- **Created**: When PR is opened or updated
- **Updated**: When you push new commits to the PR
- **Cleaned up**: Automatically when PR is closed
- **Image cleanup**: Old PR images are cleaned up after 4 weeks

### Creating a Release

To create a new release:

1. **Create a Git Tag**:
   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```

2. **Automated Process**:
   The release workflow automatically:
   - Builds and pushes Docker images (`v1.2.3` and `latest`)
   - Creates GitHub Release with details
   - **Integration environment** gets `latest` immediately
   - **Staging environment** gets a PR for `v1.2.3`
   - **Production environment** gets a PR for `v1.2.3`

3. **Deployment Process**:
   - **Integration**: Automatically updated to latest release
   - **Staging**: Review and merge the staging PR
   - **Production**: Review and merge the production PR (after staging is tested)

#### Release PR Titles

- `Staging: Release Store Service v1.2.3`
- `Production: Release Store Service v1.2.3`

### Environment Strategy

- **Integration**: Always runs `latest` (latest production release)
- **Staging**: Runs specific version (review before production)
- **Production**: Runs specific version (review before deployment)
- **PR Canaries**: Run `pr-{NUMBER}` (isolated from releases)

## Configuration

### Environment Variables

- `PORT`: gRPC server port (default: `8080`)

### Canary Metadata

- `X-Canary`: PR number for canary routing (e.g., `123`)

## gRPC API

### Service Definition

```protobuf
service Store {
  rpc Get(GetRequest) returns (GetResponse);
  rpc Set(SetRequest) returns (SetResponse);
  rpc Delete(DeleteRequest) returns (DeleteResponse);
  rpc List(ListRequest) returns (ListResponse);
}
```

### Methods

- **Get**: Retrieve a value by key
- **Set**: Store a key-value pair
- **Delete**: Remove a key-value pair
- **List**: List all stored keys

### Example Usage

```bash
# Get a value
grpcurl -plaintext -d '{"key": "example"}' localhost:8080 store.Store/Get

# Set a value
grpcurl -plaintext -d '{"key": "example", "value": "test"}' localhost:8080 store.Store/Set

# Delete a value
grpcurl -plaintext -d '{"key": "example"}' localhost:8080 store.Store/Delete

# List all keys
grpcurl -plaintext localhost:8080 store.Store/List
```

## Monitoring

The service includes:
- Health check endpoint for monitoring
- Structured logging
- gRPC server metrics
- Canary request tracking

## Troubleshooting

### Common Issues

1. **Canary not working**: Ensure you're using the correct `X-Canary` metadata format
2. **gRPC connection issues**: Check that the service is running and accessible
3. **PR canary not deployed**: Check the GitHub Actions workflow logs

### Debugging

```bash
# Check service logs
kubectl logs -f deployment/store -n apps

# Check canary routing
kubectl logs -f deployment/store-canary-pr-123 -n apps

# Test gRPC connectivity
grpcurl -plaintext localhost:8080 list
```

## Data Storage

Currently uses in-memory storage. For production use, consider:
- Redis for caching
- PostgreSQL for persistent storage
- MongoDB for document storage

## Contributing

1. Create a feature branch
2. Make your changes
3. Add tests
4. Update protobuf definitions if needed
5. Create a Pull Request
6. Test your canary deployment
7. Request review and merge

## License

[Add your license information here]
