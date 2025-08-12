# VPS Monitor

A lightweight, **Go-based VPS monitoring solution** with real-time web dashboard supporting **multiple agents**.  
Monitor unlimited servers from a single dashboard with individual agent views and overview analytics.


---

##  Features

- ** Real-time Multi-Agent Monitoring** – Monitor unlimited servers from a single dashboard
- ** Individual Agent Views** – Detailed metrics for each server with tabbed interface  
- ** Overview Dashboard** – Aggregate statistics across all monitored agents
- ** Lightweight** – Agent <10MB, Home server <25MB  
- ** Easy Deployment** – Docker Compose ready with health checks
- ** Multi-arch Support** – Works on `amd64` and `arm64`
- ** Rich Dashboard** – Modern React UI with color-coded alerts and status indicators
- ** Auto-reconnection** – Robust connection handling with retry logic
- ** System Health** – CPU, Memory, Disk, Network, and Process monitoring per agent
- ** Performance Optimized** – Efficient data collection and transmission
- ** Agent Identification** – Custom agent names and IDs for easy management

---

##  Architecture

```
                    +--------------------------+
                    |     Home Server          |
                    |   (React + Go/Gin)       |
                    |   Multi-Agent Dashboard  |
                    |   Web UI on port 8085    |
                    |   WebSocket updates      |
                    +-----------+--------------+
                                ^
                                |
                 HTTP/HTTPS/WebSocket (Secure)
                                |
                                v
    +---------------------------+---------------------------+
    |                           |                           |
    v                           v                           v
+----------+              +----------+              +----------+
|  Agent 1 |              |  Agent 2 |              |  Agent N |
|Production|              |Database  |              |Staging   |
|Web Server|              |Server    |              |Server    |
+----------+              +----------+              +----------+
    ^                           ^                           ^
    |                           |                           |
    +---------------------------+---------------------------+
              Reports every 10 seconds with unique Agent ID
```

**Multi-Agent Deployment Modes:**
1. **All-in-One** – Agent + Server on same machine (single server monitoring)
2. **Centralized** – Home Server centrally located, Agents on multiple remote servers  
3. **Hybrid** – Mix of local and remote agents

---

##  Quick Start

### **Option 1: Single Server Monitoring (All-in-One)**

Monitor the local server where you deploy both components:

```bash
# Download and start
curl -o docker-compose.yml https://raw.githubusercontent.com/hhftechnology/vps-monitor/main/docker-compose.yml
docker compose up -d

# Access dashboard
http://your-server-ip:8085
```

### **Option 2: Multi-Server Monitoring (Recommended)**

#### ** Step 1: Deploy Central Monitoring Server**

```bash
# Create docker-compose.home-only.yml
curl -o docker-compose.yml https://raw.githubusercontent.com/hhftechnology/vps-monitor/main/docker-compose.home-only.yml

# Start monitoring server
docker compose up -d

# Verify it's running
curl http://localhost:8085/api/health
```

#### ** Step 2: Deploy Agents on Target Servers**

On each server you want to monitor:

```bash
# Create docker-compose.remote-agent.yml
curl -o docker-compose.yml https://raw.githubusercontent.com/hhftechnology/vps-monitor/main/docker-compose.remote-agent.yml

# Edit the configuration
nano docker-compose.yml
# Update:
# - HOME_SERVER_URL=https://your-monitoring-server.com:8085
# - AGENT_NAME=Production-Web-Server
# - AGENT_ID=prod-web-01 (optional, defaults to hostname)

# Deploy agent
docker compose up -d
```

#### ** Step 3: Access Multi-Agent Dashboard**

Visit your monitoring server: `https://your-monitoring-server.com:8085`

You'll see:
- **Overview Tab** - Aggregate statistics across all agents
- **Individual Agent Tabs** - Detailed metrics for each server
- **Agent Status** - Online/offline indicators and last-seen timestamps

---

##  Agent Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HOME_SERVER_URL` | ✅ | - | URL of the monitoring server |
| `AGENT_NAME` | ❌ | hostname | Display name for the agent |
| `AGENT_ID` | ❌ | hostname | Unique identifier (auto-generated if not set) |

### Examples

```yaml
# Basic configuration
environment:
  - HOME_SERVER_URL=http://monitor.company.com:8085
  - AGENT_NAME=Production-Database

# Advanced configuration  
environment:
  - HOME_SERVER_URL=https://monitor.company.com
  - AGENT_NAME=Load-Balancer-Primary
  - AGENT_ID=prod-lb-01
```

---

##  Dashboard Features

###  Overview Dashboard
- **Agent Summary Cards** - Quick status overview for all agents
- **Aggregate Statistics** - Average CPU, Memory, Disk usage across agents
- **Online/Offline Status** - Real-time connection status for each agent
- **Quick Navigation** - Click any agent card to view detailed metrics

###  Individual Agent Views
- **Detailed System Metrics** - CPU, Memory, Disk, Network usage
- **Process Monitoring** - Top 30 processes with CPU and memory usage
- **Agent Information** - Version, Go runtime stats, uptime
- **Historical Context** - Last seen timestamps and connection status
- **Color-coded Alerts** - Visual indicators for critical, warning, and normal states

###  Real-time Features
- **Live Updates** - Metrics refresh every 10 seconds via WebSocket
- **Auto-reconnection** - Handles network interruptions gracefully
- **Connection Health** - Ping/pong mechanism ensures reliable connections
- **Graceful Degradation** - Shows last known state for offline agents

---

##  Deployment Scenarios

### **Scenario 1: Development/Testing (Multi-Agent Simulation)**

Test the multi-agent UI locally:

```bash
# Deploy 5 simulated agents
curl -o docker-compose.yml https://raw.githubusercontent.com/hhftechnology/vps-monitor/main/docker-compose.multi-agent-dev.yml
docker compose up -d

# Access: http://localhost:8085
# See: Production-Web, Database-Server, API-Server, Staging-Server, Load-Balancer
```

### **Scenario 2: Small Team (3-5 Servers)**

```bash
# Home Server
docker run -d --name monitor-server \
  -p 8085:8085 \
  hhftechnology/vps-monitor-home:latest

# Each Server  
docker run -d --name vps-agent \
  -e HOME_SERVER_URL=http://monitor-server-ip:8085 \
  -e AGENT_NAME="Server-$(hostname)" \
  hhftechnology/vps-monitor-agent:latest
```

### **Scenario 3: (10+ Servers)**

Use Docker Swarm or Kubernetes for orchestration:

```yaml
# kubernetes-deployment.yml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: vps-monitor-agent
spec:
  selector:
    matchLabels:
      app: vps-monitor-agent
  template:
    metadata:
      labels:
        app: vps-monitor-agent
    spec:
      containers:
      - name: agent
        image: hhftechnology/vps-monitor-agent:latest
        env:
        - name: HOME_SERVER_URL
          value: "http://vps-monitor-home.monitoring.svc.cluster.local:8085"
        - name: AGENT_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
```

---

##  Security Best Practices

### Production Deployment
- **HTTPS Only** - Always use SSL/TLS in production
- **Firewall Rules** - Restrict access to monitoring port
- **Agent Authentication** - Consider implementing API keys (future feature)
- **Network Policies** - Use VPNs or private networks for agent communication

### Example Nginx Configuration
```nginx
server {
    listen 443 ssl;
    server_name monitor.company.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8085;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

##  Scaling & Performance

### Server Resources (Home Server)
| Agents | CPU | Memory | Storage |
|--------|-----|--------|---------|
| 1-10   | 0.5 | 128MB  | 1GB     |
| 10-50  | 1.0 | 256MB  | 2GB     |
| 50-100 | 2.0 | 512MB  | 5GB     |
| 100+   | 4.0 | 1GB    | 10GB    |

### Agent Resources
- **CPU**: ~0.1-0.2 cores
- **Memory**: ~32-64MB  
- **Network**: ~1KB/10sec per agent
- **Disk**: Minimal (logs only)

### Performance Optimizations
- **WebSocket Compression** - Efficient real-time data transmission
- **Agent Deduplication** - Prevents duplicate agent registrations
- **Memory Pooling** - Optimized memory usage for large deployments
- **Graceful Cleanup** - Automatic removal of stale agents (10min timeout)

---

##  Development

### Local Development Setup

```bash
# Clone repository
git clone https://github.com/hhftechnology/vps-monitor.git
cd vps-monitor

# Test multi-agent functionality
docker-compose -f docker-compose.multi-agent-dev.yml up -d

# Access: http://localhost:8085
# See 5 different agents in the dashboard

# Frontend development (optional)
cd home/web
npm install
npm start  # Runs on :3000 with proxy to :8085
```

### Building Custom Images

```bash
# Build multi-arch images
docker buildx build --platform linux/amd64,linux/arm64 -t your-repo/vps-monitor-home:latest ./home
docker buildx build --platform linux/amd64,linux/arm64 -t your-repo/vps-monitor-agent:latest ./agent
```

---

##  Docker Images

Official multi-architecture images:

```bash
# Latest stable versions
docker pull hhftechnology/vps-monitor-home:latest   # ~25MB
docker pull hhftechnology/vps-monitor-agent:latest  # ~10MB

# Architectures supported:
# - linux/amd64 (Intel/AMD 64-bit)
# - linux/arm64 (ARM 64-bit, Apple Silicon, Raspberry Pi 4+)
```

---

##  Troubleshooting

### Common Issues

#### No Agents Appearing
```bash
# Check agent logs
docker logs vps_monitor_agent

# Check home server logs  
docker logs vps_monitor_home

# Test connectivity
curl -X POST http://your-server:8085/api/metrics \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"test","hostname":"test"}'
```

#### WebSocket Connection Issues
- Verify no proxy is blocking WebSocket upgrades
- Check firewall settings on port 8085
- Ensure correct protocol (`ws://` vs `wss://`)

#### Agent Not Updating
```bash
# Check if agent is running
docker ps | grep agent

# Verify agent can reach home server
docker exec vps_monitor_agent curl -f http://home-server:8085/api/health

# Check environment variables
docker exec vps_monitor_agent env | grep HOME_SERVER_URL
```

#### Performance Issues
- Monitor home server resources: `docker stats vps_monitor_home`
- Check agent count: Visit `/api/health` endpoint
- Review agent timeout settings (default: 2 minutes)

---

##  What's New in v1.1.0

###  Multi-Agent Support
- **Multiple Agent Monitoring** - Monitor unlimited servers from single dashboard
- **Agent Identification** - Custom agent names and IDs for easy management
- **Overview Dashboard** - Aggregate statistics across all agents
- **Individual Agent Views** - Detailed metrics for each server
- **Agent Status Tracking** - Online/offline indicators and last-seen timestamps

###  Enhanced Agent Features
- **Custom Agent Names** - Set `AGENT_NAME` environment variable
- **Unique Agent IDs** - Optional `AGENT_ID` for custom identification
- **Improved Logging** - Better log formatting with agent identification
- **Agent Self-Monitoring** - Runtime statistics and health metrics

###  UI Improvements
- **Tabbed Interface** - Easy navigation between agents and overview
- **Agent Summary Cards** - Quick status overview with click-to-view details
- **Enhanced Status Indicators** - Color-coded online/offline status
- **Responsive Design** - Better mobile and tablet support
- **Real-time Agent Count** - Live agent statistics in header

###  Technical Enhancements
- **Improved WebSocket Protocol** - Better handling of multi-agent data streams
- **Agent Cleanup** - Automatic removal of stale agents (10min timeout)
- **Enhanced Error Handling** - Better retry logic and connection management
- **Performance Optimizations** - Efficient data structures for multi-agent scenarios

---

##  License

[MIT License](LICENSE) © 2025 HHF Technology

---

##  Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Test with multi-agent setup (`docker-compose -f docker-compose.multi-agent-dev.yml up`)
4. Commit your changes (`git commit -m 'Add amazing multi-agent feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

---

** Pro Tips:**
- Use descriptive `AGENT_NAME` values like "Production-Web-Server" or "Database-Primary"
- Set up centralized logging to correlate agent metrics with application logs
- Use Docker health checks to monitor agent connectivity
- Implement monitoring alerts based on agent offline status
- Consider using a reverse proxy with SSL for production deployments