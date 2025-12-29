# VPS-Monitor

VPS-Monitor is an open-source, high-performance Docker container monitoring and management tool. Built for speed and ease of use, it provides real-time log streaming, container stats, image management, network visualization, alerting, and multi-host support through a clean, modern interface.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Architecture](#architecture)
- [Development](#development)
- [Troubleshooting](#troubleshooting)

## Features

### Container Management

- Start, stop, restart, and remove containers
- Real-time container state synchronization
- Filter by state (running, exited, paused, restarting, dead)
- Search by container name, ID, or image
- Group containers by Docker Compose project
- Sort by creation date with date range filtering
- Read-only mode for monitoring-only deployments

### Real-Time Log Streaming

- Live log streaming with play/pause controls
- Historical log viewing with configurable line counts
- Auto-scroll toggle during streaming
- Toggleable timestamps and text wrapping
- Full-text search with highlighting and navigation
- Filter by log level (TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC)
- Log download in JSON or TXT format

### Container Stats and Metrics

- Real-time CPU and memory usage via WebSocket
- Network I/O monitoring (RX/TX bytes)
- Block I/O statistics (read/write)
- Process count (PIDs) tracking
- Threshold-based alerting

### Image Management

- List images across all Docker hosts
- View image details including size, tags, and creation date
- Pull images with real-time progress streaming
- Remove images with force option
- Multi-host image operations

### Network Management

- View Docker networks across all hosts
- Network details including IPAM configuration
- Connected containers with IP and MAC addresses
- Internal/external network indicators
- IPv6 support status

### Alerting and Notifications

- CPU and memory threshold monitoring
- Container stopped detection
- Webhook notifications (Slack, Discord, custom endpoints)
- In-memory alert history with acknowledge function
- Configurable check intervals

### Multi-Host Docker Support

- Connect to local Unix sockets, remote SSH, or TCP endpoints
- Parallel queries across all hosts for performance
- Host-aware filtering and operations
- Secure SSH-based connections with key authentication

See the [Multi-Host Setup Guide](./multi-host.md) for detailed configuration.

### Interactive Terminal

- WebSocket-based container terminal access
- Full terminal emulation with XTerm.js
- 10,000 line scrollback history
- Copy-to-clipboard support

### Environment Variables Management

- View and edit container environment variables
- Bulk import from .env files
- Container recreation with updated variables

### User Interface

- Clean dashboard with summary cards
- Light, dark, and system theme support
- Responsive design for mobile, tablet, and desktop
- URL state persistence for shareable views
- Accessible UI components with Radix UI

### Authentication and Security

- Optional JWT-based authentication
- Bcrypt password hashing
- Read-only mode support
- Per-request authorization

## Quick Start

### Using Docker Compose

```yaml
services:
  vps-monitor:
    image: ghcr.io/hhftechnology/vps-monitor:latest
    ports:
      - "6789:6789"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - READONLY_MODE=false
```

```bash
docker compose up -d
```

Access the dashboard at `http://localhost:6789`

### With Authentication

```yaml
services:
  vps-monitor:
    image: ghcr.io/hhftechnology/vps-monitor:latest
    ports:
      - "6789:6789"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - JWT_SECRET=your-secret-key-minimum-32-characters
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=$2a$10$YourBcryptHashHere
```

Generate password hash:

```bash
htpasswd -nbBC 10 "" yourpassword | tr -d ':\n'
```

## Installation

### Prerequisites

- Docker 20.10 or higher
- Go 1.23 or higher (for building from source)
- Node.js 20 or higher with Bun (for frontend development)

### From Source

```bash
# Clone repository
git clone https://github.com/hhftechnology/vps-monitor.git
cd vps-monitor

# Build backend
cd home
go build -o vps-monitor ./cmd/server

# Build frontend
cd ../frontend
bun install
bun run build

# Run
./vps-monitor
```

### Using Docker

```bash
docker pull ghcr.io/hhftechnology/vps-monitor:latest
docker run -d -p 6789:6789 -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/hhftechnology/vps-monitor:latest
```

## Configuration

### Environment Variables

#### Authentication (Optional)

| Variable | Description | Default |
|----------|-------------|---------|
| `JWT_SECRET` | Secret key for JWT tokens (min 32 chars) | None (auth disabled) |
| `ADMIN_USERNAME` | Admin username | None |
| `ADMIN_PASSWORD` | Bcrypt-hashed admin password | None |

Authentication is disabled when these variables are not set.

#### Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `READONLY_MODE` | Disable mutating operations | `false` |
| `BACKEND_PORT` | Backend server port | `6789` |
| `FRONTEND_PORT` | Frontend dev server port | `2345` |

#### Docker Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `DOCKER_HOSTS` | Multi-host configuration | Local socket |

Format: `name1=host1,name2=host2`

Examples:
```bash
# Local only
DOCKER_HOSTS=local=unix:///var/run/docker.sock

# Local and remote
DOCKER_HOSTS=local=unix:///var/run/docker.sock,prod=ssh://deploy@prod.example.com

# Multiple remotes
DOCKER_HOSTS=us=ssh://root@us.example.com,eu=ssh://root@eu.example.com
```

#### Alert Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `ALERTS_ENABLED` | Enable alerting system | `false` |
| `ALERTS_WEBHOOK_URL` | Webhook URL for notifications | None |
| `ALERTS_CPU_THRESHOLD` | CPU usage alert threshold (0-100) | `80` |
| `ALERTS_MEMORY_THRESHOLD` | Memory usage alert threshold (0-100) | `90` |
| `ALERTS_CHECK_INTERVAL` | Check interval (Go duration) | `30s` |

Example:
```bash
ALERTS_ENABLED=true
ALERTS_WEBHOOK_URL=https://hooks.slack.com/services/XXX/YYY/ZZZ
ALERTS_CPU_THRESHOLD=85
ALERTS_MEMORY_THRESHOLD=90
ALERTS_CHECK_INTERVAL=1m
```

## API Reference

### Authentication

```
POST /api/v1/auth/login
```

Request body:
```json
{
  "username": "admin",
  "password": "password"
}
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

Use the token in subsequent requests:
```
Authorization: Bearer <token>
```

### Containers

```
GET    /api/v1/containers                    # List all containers
GET    /api/v1/containers/{id}?host={host}   # Get container details
POST   /api/v1/containers/{id}/start         # Start container
POST   /api/v1/containers/{id}/stop          # Stop container
POST   /api/v1/containers/{id}/restart       # Restart container
DELETE /api/v1/containers/{id}/remove        # Remove container
GET    /api/v1/containers/{id}/logs          # Get container logs
GET    /api/v1/containers/{id}/logs/stream   # Stream logs (SSE)
GET    /api/v1/containers/{id}/stats         # Stream stats (WebSocket)
GET    /api/v1/containers/{id}/terminal      # Terminal access (WebSocket)
GET    /api/v1/containers/{id}/env           # Get environment variables
PUT    /api/v1/containers/{id}/env           # Update environment variables
```

### Images

```
GET    /api/v1/images                              # List all images
GET    /api/v1/images/{id}?host={host}             # Get image details
DELETE /api/v1/images/{id}?host={host}&force=bool  # Remove image
POST   /api/v1/images/pull?host={host}&image=name  # Pull image (streams progress)
```

### Networks

```
GET /api/v1/networks                     # List all networks
GET /api/v1/networks/{id}?host={host}    # Get network details
```

### Alerts

```
GET  /api/v1/alerts                      # List all alerts
GET  /api/v1/alerts/config               # Get alert configuration
POST /api/v1/alerts/{id}/acknowledge     # Acknowledge an alert
```

### System

```
GET /api/v1/system/stats    # Get system statistics
```

## Architecture

### Backend (Go)

```
home/
  cmd/server/main.go       # Application entry point
  internal/
    api/                   # HTTP handlers and routing
      router.go            # Chi router setup
      handlers.go          # Container handlers
      image_handlers.go    # Image handlers
      network_handlers.go  # Network handlers
      alert_handlers.go    # Alert handlers
      stats_ws.go          # WebSocket stats streaming
      terminal.go          # WebSocket terminal
    docker/                # Docker client layer
      client.go            # Multi-host client
      container.go         # Container operations
      image.go             # Image operations
      network.go           # Network operations
      stats.go             # Stats streaming
    models/                # Data structures
    config/                # Configuration parsing
    auth/                  # JWT authentication
    alerts/                # Alert monitoring system
      monitor.go           # Background monitoring
      webhook.go           # Webhook notifications
      history.go           # Alert storage
```

### Frontend (React + TypeScript)

```
frontend/src/
  components/              # Shared UI components
    ui/                    # Shadcn UI components
    header.tsx             # Navigation header
    footer.tsx             # Page footer
    theme-toggle.tsx       # Theme switcher
  contexts/                # React contexts
    auth-context.tsx       # Authentication state
    theme-context.tsx      # Theme state
  features/                # Feature modules
    containers/            # Container management
      api/                 # API functions
      hooks/               # React Query hooks
      components/          # UI components
      types.ts             # TypeScript types
    images/                # Image management
    networks/              # Network management
    alerts/                # Alert management
  routes/                  # TanStack Router pages
    __root.tsx             # Root layout
    index.tsx              # Dashboard
    images/                # Images page
    networks/              # Networks page
    alerts/                # Alerts page
  lib/                     # Utilities
    api-client.ts          # Authenticated fetch
    utils.ts               # Helper functions
```

### Key Technologies

**Backend:**
- Go 1.23+
- Chi v5 router
- Docker SDK v28
- gorilla/websocket
- JWT authentication

**Frontend:**
- React 19
- TypeScript 5.7
- TanStack Router and Query
- Tailwind CSS 4
- Shadcn UI with Radix
- Vite 7

## Development

### Backend Development

```bash
cd home
go run ./cmd/server
```

The server runs on port 6789.

### Frontend Development

```bash
cd frontend
bun install
bun run dev
```

The dev server runs on port 2345 with API proxy to localhost:6789.

### Running Tests

```bash
# Backend
cd home
go test ./...

# Frontend
cd frontend
bun run test
```

### Building for Production

```bash
# Backend
cd home
go build -o vps-monitor ./cmd/server

# Frontend
cd frontend
bun run build
```

The frontend build output is served by the backend from the embedded filesystem.

## Troubleshooting

### Cannot connect to Docker

Verify Docker socket access:
```bash
ls -l /var/run/docker.sock
docker ps
```

Ensure the user has Docker permissions:
```bash
sudo usermod -aG docker $USER
```

### Authentication not working

1. Verify all three environment variables are set:
   - `JWT_SECRET`
   - `ADMIN_USERNAME`
   - `ADMIN_PASSWORD`

2. Ensure password is bcrypt-hashed, not plaintext

3. Check JWT_SECRET is at least 32 characters

### WebSocket connections failing

1. Check firewall allows WebSocket upgrades
2. Verify reverse proxy configuration (if applicable)
3. Check browser console for connection errors

### Alerts not triggering

1. Verify `ALERTS_ENABLED=true`
2. Check container stats are streaming correctly
3. Verify webhook URL is accessible
4. Check server logs for alert errors

### Multi-host SSH connection issues

See the [Multi-Host Setup Guide](./multi-host.md) for detailed SSH configuration and troubleshooting.

## License

GPL-3.0 License. See [LICENSE](./LICENSE) for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request

## Support

- GitHub Issues: [https://github.com/hhftechnology/vps-monitor/issues](https://github.com/hhftechnology/vps-monitor/issues)
- Documentation: [https://github.com/hhftechnology/vps-monitor](https://github.com/hhftechnology/vps-monitor)
