# bug-free-umbrella

Initial Go web API setup using [Gin](https://github.com/gin-gonic/gin) with OpenTelemetry tracing, Swagger documentation, and a standard project structure established. Business logic to follow.

## Stack

- **Go / Gin** - HTTP framework
- **OpenTelemetry** - Distributed tracing (exported via gRPC to an OTel Collector)
- **Jaeger** - Trace visualization
- **Swag** - Auto-generated OpenAPI/Swagger docs from annotations

## Project Structure

```
cmd/server/          Entrypoint and dependency wiring
internal/handler/    HTTP handlers with Swagger annotations
internal/service/    Business logic
pkg/tracing/         OpenTelemetry initialization
docs/                Generated Swagger spec (do not edit manually)
```

## Running

```sh
docker compose up --build
```

| Service     | URL                          |
|-------------|------------------------------|
| API         | http://localhost:8080         |
| Swagger UI  | http://localhost:8080/swagger/index.html |
| Jaeger UI   | http://localhost:16686        |

## API Endpoints

| Method | Path         | Description                        |
|--------|--------------|------------------------------------|
| GET    | /health      | Health check                       |
| GET    | /api/hello   | Hello world greeting               |
| GET    | /api/slow    | Simulated slow response (150ms)    |

## Regenerating Swagger Docs

After adding or modifying handler annotations:

```sh
swag init -g cmd/server/main.go
```

This runs automatically during the Docker build.
