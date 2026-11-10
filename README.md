# Terminus

Terminus is a high-performance, modular rate limiting service written in Go. It provides both standalone in-memory rate limiting and distributed rate limiting powered by Redis and atomic Lua scripts. Terminus is designed to sit in the request pipeline as an HTTP middleware, protecting upstream services against denial-of-service, abusive traffic patterns, and sudden load spikes while maintaining low latency and high availability through an integrated fallback mechanism.

## Architecture

Terminus follows a layered, modular architecture that separates HTTP transport concerns, configuration management, rate limiting logic, and data storage.

### System Diagram

```
                 +-----------------------+
                 |    Incoming Request   |
                 +-----------------------+
                             |
                             v
                 +-----------------------+
                 | Rate Limit Middleware |
                 | (Client ID Extraction)|
                 +-----------------------+
                             |
                             v
                 +-----------------------+
                 | RateLimiter Interface |
                 +-----------------------+
                             |
             +---------------+---------------+
             |                               |
       [Local Mode]                 [Distributed Mode]
             |                               |
             v                               v
    +-----------------+            +-------------------+
    | In-Memory Store |            |  Primary: Redis   |
    |  (Go Mutexes)   |            |  (Lua Execution)  |
    +-----------------+            +-------------------+
                                             |
                                    (on Redis failure)
                                             v
                                   +-------------------+
                                   | Fallback: Local   |
                                   | In-Memory Limiter |
                                   +-------------------+
```

### Core Components

- HTTP Server and Routing (`cmd/server`): Entrypoint that bootstraps environment configurations, establishes Redis connections when running in distributed mode, mounts middleware, and registers HTTP routes.
- Middleware (`middleware`): An HTTP middleware that intercepts requests, extracts client identifiers (such as remote IP addresses), queries the rate limiter interface, and either passes traffic to downstream handlers or halts the request with an HTTP 429 Too Many Requests response.
- Rate Limiter Engine (`ratelimit`): Implements the unified `RateLimiter` interface. A factory creates limiter instances based on configuration.
- Storage and Scripting (`scripts`): Contains Redis Lua scripts embedded into the Go binary using the standard `embed` package. These scripts execute atomically within Redis to eliminate race conditions without requiring distributed locking.
- Configuration (`internal/config`): Centralized environment variable loader that handles application modes, connection strings, and algorithm-specific parameters.

### Distributed Resilience with Fallback Limiter

When configured in distributed mode, Terminus wraps the Redis-backed limiter inside a `FallbackLimiter`. If Redis becomes unreachable, experiences network partitioning, or times out:
1. The primary distributed call fails gracefully.
2. The error is logged as a warning.
3. The request is immediately evaluated against a local in-memory limiter instance.

This prevents rate-limiting infrastructure from becoming a single point of failure for upstream applications.

## Supported Algorithms

Terminus implements five distinct rate limiting algorithms for both local in-memory and distributed Redis modes:

1. Token Bucket
   - Accumulates tokens over time at a constant refill rate up to a configured bucket capacity.
   - Each incoming request consumes a token. Requests are allowed if a token is available; otherwise, rejected.
   - Supports bursts of traffic up to the bucket capacity while maintaining an average rate.

2. Leaky Bucket
   - Models incoming requests into a bucket that leaks at a constant rate.
   - Smooths out bursts and processes requests at an even pace.
   - Requests that exceed bucket capacity are immediately rejected.

3. Fixed Window Counter
   - Divides time into fixed discrete intervals (for example, 1 minute).
   - Tracks request counts per client within the current window.
   - Resets the counter at each window boundary. Simple and lightweight, though susceptible to traffic spikes at window boundaries.

4. Sliding Window Log
   - Retains exact timestamps of each request within a rolling window.
   - Provides strict enforcement by removing timestamps older than the window length and counting remaining entries.
   - Redis implementation utilizes Sorted Sets (ZSET) with score-based range queries. Highly accurate but consumes more memory.

5. Sliding Window Counter
   - Blends the low memory overhead of fixed window counters with the precision of sliding window logs.
   - Computes an estimated count by weighting the previous window counter with the elapsed percentage of the current window.

## How It Works

### 1. Request Lifecycle

1. An HTTP client issues a request to an endpoint protected by `RateLimitMiddleware`.
2. The middleware extracts the client identifier (currently parsed from `RemoteAddr`).
3. The middleware passes the client identifier and request context to `limiter.Allow(ctx, clientID)`.
4. If allowed:
   - The request proceeds to the target handler and returns HTTP 200 OK.
5. If denied:
   - The middleware terminates execution immediately and returns HTTP 429 Too Many Requests with a `rate limit exceeded` message.
6. If an unhandled error occurs during execution:
   - The middleware returns HTTP 500 Internal Server Error.

### 2. Local vs. Distributed Execution

- Local Mode:
  State is stored inside Go process memory using native data structures (maps, slices, queues) protected with synchronization primitives (`sync.Mutex`). It provides near-zero latency and requires no external dependencies, suitable for single-instance deployments.

- Distributed Mode:
  State is maintained centrally in Redis. All evaluation logic is bundled inside Lua scripts executed directly on Redis via `EVALSHA`. Because Redis executes Lua scripts as a single atomic unit, updates to counters, timestamps, and sorted sets are safe from concurrency issues without explicit distributed locks.

## Configuration

Terminus is configured through environment variables:

| Variable | Description | Default |
|---|---|---|
| `SERVER_ADDR` | Server bind address and port | `:8081` |
| `MODE` | Limiter execution mode (`local` or `distributed`) | `local` |
| `RATE_LIMIT_ALGORITHM` | Rate limiter algorithm (`token_bucket`, `leaky_bucket`, `fixed_window`, `sliding_log`, `sliding_counter`) | `token_bucket` |
| `REDIS_ADDR` | Redis host and port (distributed mode) | `localhost:6379` |
| `REDIS_PASSWORD` | Redis authentication password | `""` |
| `REDIS_DB` | Redis database index | `0` |
| `TOKEN_BUCKET_CAPACITY` | Maximum token bucket size | `5` |
| `TOKEN_BUCKET_REFILL_RATE` | Tokens added per second | `1.0` |
| `LEAKY_BUCKET_CAPACITY` | Maximum queue depth for leaky bucket | `5` |
| `LEAKY_BUCKET_LEAK_RATE` | Leaked requests per second | `1.0` |
| `FIXED_WINDOW_LIMIT` | Max allowed requests per fixed window | `10` |
| `FIXED_WINDOW_DURATION` | Duration of fixed window interval | `1m` |
| `SLIDING_LOG_LIMIT` | Max allowed requests per sliding log window | `10` |
| `SLIDING_LOG_WINDOW` | Sliding log time window duration | `1m` |
| `SLIDING_COUNTER_LIMIT` | Max allowed requests per sliding counter window | `10` |
| `SLIDING_COUNTER_WINDOW` | Sliding counter time window duration | `1m` |

## Getting Started

### Prerequisites

- Go 1.22 or higher
- Redis 7+ (optional, required only for distributed mode)
- Docker and Docker Compose (optional, for containerized execution)

### Running Locally

1. Clone the repository and install dependencies:
   ```bash
   git clone https://github.com/sushr05/terminus.git
   cd terminus
   go mod download
   ```

2. Run in local mode with default settings:
   ```bash
   go run cmd/server/main.go
   ```

3. Run in distributed mode against a local Redis instance:
   ```bash
   MODE=distributed REDIS_ADDR=localhost:6379 RATE_LIMIT_ALGORITHM=token_bucket go run cmd/server/main.go
   ```

### Running with Docker Compose

To spin up both Terminus and Redis in distributed mode:

```bash
docker compose up --build
```

### Verification and Health Check

- Health Check:
  ```bash
  curl -i http://localhost:8081/health
  ```

- Test Rate Limiter:
  ```bash
  curl -i http://localhost:8081/api/test
  ```

## Future Plans

- Standardized Rate Limit Headers: Emit RFC-compliant HTTP response headers (`X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `Retry-After`).
- Flexible Key Resolvers: Support client identification by API key, JWT claims, custom headers, and route paths rather than IP address alone.
- Redis Cluster and Sentinel Support: Extend Redis connection management to handle Redis Sentinel failover and distributed Redis Cluster setups.
- Multi-tier Rules: Allow different rate limit tiers based on user roles, endpoints, and subscription levels.
- Metrics and Observability: Expose Prometheus metrics for allowed requests, rejected requests, Redis latency, and fallback activations.
