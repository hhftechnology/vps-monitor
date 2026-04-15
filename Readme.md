<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="/.github/logotype-dark.png">
    <source media="(prefers-color-scheme: light)" srcset="/.github/logotype-light.png">
    <img src="/.github/logotype-dark.png" width="400" alt="VPS Monitor">
  </picture>
</div>

<div align="center">
    <h1>VPS-Monitor</h1>
    <p>VPS-Monitor is an open-source, high-performance Docker container monitoring and management tool. Built for speed and ease of use, it provides real-time log streaming, container stats, image management, network visualization, alerting, and multi-host support through a clean, modern interface.</p>

[![Docker](https://img.shields.io/docker/pulls/hhftechnology/vps-monitor?style=flat-square)](https://hub.docker.com/r/hhftechnology/vps-monitor)
![Stars](https://img.shields.io/github/stars/hhftechnology/vps-monitor?style=flat-square)
[![Discord](https://img.shields.io/discord/994247717368909884?logo=discord&style=flat-square)](https://discord.gg/HDCt9MjyMJ)
</div>

<div align="center">
<a href="https://apps.apple.com/us/app/#"><img width="135" height="39" alt="appstore" src="https://github.com/user-attachments/assets/45e31a11-cf6b-40a2-a083-6dc8d1f01291" /></a>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<a href="https://play.google.com/store/apps/details?id=com.vps.monitor.mobile"><img width="135" height="39" alt="googleplay" src="https://github.com/user-attachments/assets/acbba639-858f-4c74-85c7-92a4096efbf5" /></a>
</div>

<div align="center">

| Docker Image | linux/amd64 Size |
| --- | --- |
| `hhftechnology/vps-monitor:latest` | <!-- VPS_MONITOR_IMAGE_SIZE_START -->pending<!-- VPS_MONITOR_IMAGE_SIZE_END --> |
</div>

<div align="center">
<img width="1735" height="1058" alt="image" src="https://github.com/user-attachments/assets/35241a4e-d523-40eb-9455-ed33ab837b66" />

<img width="1735" height="1467" alt="image" src="https://github.com/user-attachments/assets/ccb23590-8ecd-4c89-89ea-68a24db69418" />

</div>

## Stats

<img width="1735" height="802" alt="image" src="https://github.com/user-attachments/assets/78cdc82e-9d9f-4734-aae6-592b0374ec61" />
<img width="1735" height="1019" alt="image" src="https://github.com/user-attachments/assets/6327ae11-c719-42d1-8e06-9ff41c398395" />

## Images Managment

<img width="1735" height="988" alt="image" src="https://github.com/user-attachments/assets/c791ef58-cd27-4c54-b9e3-f07d1ca34227" />

## Network Info

<img width="1735" height="802" alt="image" src="https://github.com/user-attachments/assets/a8f87c44-fde5-4923-8549-6cf706113678" />
<img width="672" height="885" alt="image" src="https://github.com/user-attachments/assets/0cb67ac4-2784-4bf7-bc93-9477d2601919" />

## Grouping by Compose files
<img width="1735" height="1133" alt="image" src="https://github.com/user-attachments/assets/f83f9ed6-6642-400a-8d71-6be944143aab" />




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
- Dedicated Capacitor-based mobile app in `mobile/`
- Mobile routes for dashboard, stats, containers, images, networks, and alerts
- Mobile routes for walkthrough and About Us
- URL state persistence for shareable views
- Accessible UI components with Radix UI

### Authentication and Security

- Optional JWT-based authentication
- SHA256 password+salt credential hashing
- Read-only mode support
- Per-request authorization

### Mobile App

- React + Vite + Capacitor mobile client under `mobile/`
- Authenticated login and URL-only login
- First-launch walkthrough with replay support
- About Us page with app metadata and support links
- Mobile dashboard with system stats and container cards
- Dedicated mobile stats page for running containers
- Container detail support for logs, live stats, terminal, and env variables
- Images, networks, and alerts screens adapted for mobile
- Mobile API smoke test and production checklist included in the repo

## Quick Start

### Using Docker Compose

```yaml
services:
  vps-monitor:
    image: hhftechnology/vps-monitor:latest
    ports:
      - "6789:6789"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - ./data:/data
    environment:
      - READONLY_MODE=false
      - DOCKER_HOSTS=local=unix:///var/run/docker.sock
      - HOSTNAME_OVERRIDE=Pangolin Host 
```

```bash
docker compose up -d
```

Access the dashboard at `http://localhost:6789`

### Mobile Quick Start

Run the backend:

```bash
cd home
go run ./cmd/server
```

In another terminal, run the mobile client:

```bash
cd mobile
npm install
npm run dev
```

Then:

1. Open the mobile app in a browser, emulator, or Capacitor shell
2. Connect to your backend URL, for example `http://localhost:6789`
3. Log in if authentication is enabled
4. Complete the walkthrough on first launch
5. Validate the API first with:

```bash
chmod +x home/scripts/mobile-api-smoke.sh
VPS_MONITOR_USERNAME=admin \
VPS_MONITOR_PASSWORD='your-password' \
home/scripts/mobile-api-smoke.sh http://localhost:6789
```

For the full mobile validation flow, see:

- `mobile/README.md`
- `mobile/MOBILE_PROD_CHECKLIST.md`

### With Authentication

```yaml
services:
  vps-monitor:
    image: hhftechnology/vps-monitor:latest
    ports:
      - "6789:6789"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - ./data:/data
    environment:
      - JWT_SECRET=your-secret-key-minimum-32-characters
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD_SALT=mysalt
      # Hash of "admin123mysalt"
      - ADMIN_PASSWORD=200ceb26807d6bf99fd6f4f0d1ca54d410af42fd47c58747466549a8f2762e15
```

Generate password hash:

```bash
# Format: echo -n "password+salt" | shasum -a 256
echo -n "admin123mysalt" | shasum -a 256 | awk '{print $1}'
```

## Installation

### Prerequisites

- Docker 20.10 or higher
- Go 1.24 or higher (for building from source)
- Node.js 20 or higher with Bun (for frontend development)
- Node.js 20 or higher with npm (for mobile development)

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

# Build mobile web bundle
cd ../mobile
npm install
npm run build

# Run
./vps-monitor
```

### Using Docker

```bash
docker pull hhftechnology/vps-monitor:latest
docker run -d -p 6789:6789 -v /var/run/docker.sock:/var/run/docker.sock hhftechnology/vps-monitor:latest
```

## Configuration

### Environment Variables

#### Authentication (Optional)

| Variable | Description | Default |
|----------|-------------|---------|
| `JWT_SECRET` | Secret key for JWT tokens (min 32 chars) | None (auth disabled) |
| `ADMIN_USERNAME` | Admin username | None |
| `ADMIN_PASSWORD` | SHA256 hash of (password + salt) | None |
| `ADMIN_PASSWORD_SALT` | Salt for SHA256 password hashing | None |

Authentication is disabled when these variables are not set.

#### Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `READONLY_MODE` | Disable mutating operations | `false` |
| `HOSTNAME_OVERRIDE` | Custom hostname to display in UI | System hostname |
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
POST   /api/v1/containers/{id}/remove        # Remove container
GET    /api/v1/containers/{id}/logs/parsed   # Get or stream parsed logs
GET    /api/v1/containers/{id}/stats         # Stream stats (WebSocket)
GET    /api/v1/containers/{id}/stats/once    # Get one stats snapshot
GET    /api/v1/containers/{id}/exec          # Terminal access (WebSocket)
GET    /api/v1/containers/{id}/env           # Get environment variables
PUT    /api/v1/containers/{id}/env           # Update environment variables
```

Note: container terminal access is implemented on `/api/v1/containers/{id}/exec`. Older references to `/api/v1/containers/{id}/terminal` are legacy naming and not the current documented route.

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
POST /api/v1/alerts/acknowledge-all      # Acknowledge all alerts
```

### System

```
GET /api/v1/system/stats    # Get system statistics
```

### Devices

```
POST /api/v1/devices/register   # Accept mobile device registration payload
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

### Mobile App (React + Capacitor)

```
mobile/
  src/
    App.tsx               # Mobile router and shell
    pages/                # Mobile route screens
    components/           # Mobile UI and feature components
    hooks/                # Mobile data/realtime hooks
    lib/api-client.ts     # Shared mobile HTTP/WS client helpers
    lib/onboarding.ts     # Walkthrough persistence helpers
    lib/push-notifications.ts
  MOBILE_PROD_CHECKLIST.md
  README.md
```

### Key Technologies

**Backend:**
- Go 1.24+
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

### Mobile Development

```bash
cd mobile
npm install
npm run dev
```

Typecheck, lint, and test the mobile app:

```bash
cd mobile
npx tsc -p tsconfig.app.json --noEmit
npm run lint
npm test
```

Build the mobile web bundle:

```bash
cd mobile
npm run build
```

Capacitor workflows:

```bash
cd mobile
npm run build:mobile
npm run cap:sync
npm run cap:android
npm run cap:ios
```

The mobile app now also includes:
- a first-launch walkthrough
- replay support from the header menu
- an About Us page with app/version/support info

### Running Tests

```bash
# Backend
cd home
go test ./...

# Frontend
cd frontend
bun run test

# Mobile
cd ../mobile
npm test
```

### Mobile API smoke test

```bash
chmod +x home/scripts/mobile-api-smoke.sh
VPS_MONITOR_USERNAME=admin \
VPS_MONITOR_PASSWORD='your-password' \
home/scripts/mobile-api-smoke.sh https://your-server:6789
```

For the full real-device/mobile validation flow, see:

- `mobile/MOBILE_PROD_CHECKLIST.md`

### Building for Production

```bash
# Backend
cd home
go build -o vps-monitor ./cmd/server

# Frontend
cd frontend
bun run build

# Mobile
cd ../mobile
npm run build
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

1. Verify all four environment variables are set:
   - `JWT_SECRET`
   - `ADMIN_USERNAME`
   - `ADMIN_PASSWORD`
   - `ADMIN_PASSWORD_SALT`

2. Ensure `ADMIN_PASSWORD` is the SHA256 hash of `password + salt`, not plaintext

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
