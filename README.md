## API Gateway with Rate Limiter
### Design
- Token bucket rate limiting with per-minute and per-hour limits
- Response caching with Redis (60s cache expiry)
- API key authentication with PostgreSQL
- Tracked system metrics like cache hits, cache misses, and rate limits
### Flowchart of Design
Tried to use color coding to show the direction of the path/flow for:
  - Response from Backend API (Red)
  - Request from Proxy Handler (Blue)
  - Cache hit and Cache miss (Green)
##### Backend API "external REST service" is [jsonplaceholder](https://jsonplaceholder.typicode.com/) for context
<img width="585" height="452" alt="image" src="https://github.com/user-attachments/assets/39ea986c-37dd-48ac-8470-b5ebee0ac51f" />

### Tech
- Built using Go 1.21
- Redis for rate limiting and caching
- PostgreSQL for API key authentication and storing request logs
- Docker for containerization
### Start

```bash
# start docker
docker-compose up -d

# run
go run cmd/server/main.go

# create api key
curl -X POST http://localhost:8080/admin/keys \
  -H "Content-Type: application/json" \
  -d '{"name":"test-key","rate_limit_per_minute":60,"rate_limit_per_hour":1000}'

# make request
curl http://localhost:8080/users/1 \
  -H "X-API-Key: YOUR_KEY_HERE"

# check metrics
curl http://localhost:8080/metrics
```

### Performance
- Cache hit: ~10ms
- Cache miss: ~230ms
- Average improvement: 220ms


