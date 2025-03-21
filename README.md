# API Gateway Service

This is an API Gateway service implemented in Go that serves as a single entry point for client applications to access various backend microservices.

## Features
 
- **Request Routing**: Receive client request from Nginx & Routes requests to appropriate backend services
- **Authentication**: JWT-based authentication middleware
- **Rate Limiting**: Redis-based rate limiting to prevent abuse
- **Logging**: Comprehensive request/response logging
- **Health Checks**: Monitors the health of the API Gateway and its dependencies
- **Error Handling**: Consistent error handling across services

## Architecture

The API Gateway follows a modular architecture with the following components:

- **Config**: Configuration management using environment variables
- **Middleware**: Authentication, rate limiting, and logging middleware
- **Proxy**: Service proxy for routing requests to backend services
- **Handlers**: Request handlers for specific endpoints

## Prerequisites

- Go 1.23 or higher
- Redis (for rate limiting)
- Backend services to proxy requests

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/api-gateway-service-ms.git
   cd api-gateway-service-ms
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Create a `config.yaml` file based on the `sample.config` file:
   ```bash
   cp sample.config config.yaml
   ```

4. Update the `config.yaml` file with your configuration.

## Running the API Gateway

### Development Mode

```bash
go run main.go
```

### Production Mode

```bash
ENV=production go run main.go
```

### Building and Running

```bash
go build -o api-gateway
./api-gateway
```

## API Endpoints

- **Health Check**: `GET /health`
  - Returns the health status of the API Gateway and its dependencies

- **API Routes**: `GET|POST|PUT|DELETE /api/{service}/{path}`
  - Routes requests to the appropriate backend service
  - Requires JWT authentication

## Docker Support

A Dockerfile is provided to build and run the API Gateway in a container:

```bash
docker build -t api-gateway .
docker run -p 8080:8080 --env-file /app/config/config.yaml api-gateway
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit your changes: `git commit -am 'Add my feature'`
4. Push to the branch: `git push origin feature/my-feature`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details. 