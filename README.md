# VPS Monitor

A lightweight, **Go-based VPS monitoring solution**.  
Run a tiny **agent** on your VPS and a **home server** locally or remotely to view real-time system metrics.

---

##  Table of Contents
1. [Features](#features)
2. [Architecture](#architecture)
3. [Deployment](#deployment)
   - [Option 1: All-in-One Deployment](#option-1-all-in-one-deployment)
   - [Option 2: Remote Agent Deployment with Pangolin](#option-2-remote-agent-deployment-with-pangolin)
4. [Prerequisites](#prerequisites)
5. [Development](#development)
6. [License](#license)

---

##  Features
- **Real-time Monitoring** – Live metrics from your VPS.
- **Lightweight** – Agent <10MB, Home server <25MB.
- **Easy Deployment** – Docker Compose ready.
- **Multi-arch Support** – Works on `amd64` and `arm64`.

---

##  Architecture


```
      +-----------------------+
      |      Home Server      |
      |   (Local or Remote)   |
      |  Web UI on port 8085  |
      +-----------+-----------+
                  ^
                  |
 Secure Link (HTTP/HTTPS / Pangolin)
                  |
                  v
      +-----------------------+
      |        Agent          |
      | Runs on each VPS node |
      +-----------------------+
```


**Modes:**
1. **All-in-One** – Agent + Server on same VPS  
2. **Remote** – Agent on remote VPS, Server at home (reverse proxy required)  

---

##  Deployment

### **Option 1: All-in-One Deployment**
Run both the **Agent** and **Home Server** on the same VPS.

```yaml
services:
  home-server:
    image: hhftechnology/vps-monitor-home:latest
    ports:
      - "8085:8085"
    container_name: vps_monitor_home
    restart: unless-stopped

  agent:
    image: hhftechnology/vps-monitor-agent:latest
    environment:
      - HOME_SERVER_URL=http://home-server:8085
    container_name: vps_monitor_agent
    depends_on:
      - home-server
    restart: unless-stopped
````

Run:

```bash
docker compose up -d
```

Access:

```
http://<your-vps-ip>:8085
```

---

### **Option 2: Remote Agent Deployment with Pangolin**

Monitor remote VPS from your home network.

#### **Home Machine (Server)**

```yaml

services:
  home-server:
    image: hhftechnology/vps-monitor-home:latest
    expose:
      - "8085"
    container_name: vps_monitor_home
    restart: unless-stopped
```

```bash
docker compose up -d
```

Expose with **Pangolin**:
Create a resource pointing to port `8085` of `home-server`, giving you:

```
https://monitor.your-domain.com
```

#### **Remote VPS (Agent)**

```yaml

services:
  agent:
    image: hhftechnology/vps-monitor-agent:latest
    environment:
      - HOME_SERVER_URL=https://monitor.your-domain.com
    container_name: vps_monitor_agent
    restart: unless-stopped
```

```bash
docker compose up -d
```

---

##  Prerequisites

* Docker & Docker Compose installed.
* For remote deployment, a reverse proxy (Pangolin, Traefik, Nginx) is recommended.

---

##  Development

Clone:

```bash
git clone https://github.com/hhftechnology/vps-monitor.git
cd vps-monitor
```

Build & Run:

```bash
docker compose up --build -d
```

---

## License

[MIT License](LICENSE) © 2025 HHF Technology


