# Multi-Host Docker Configuration

## Overview

VPS-Monitor includes native support for monitoring and managing multiple Docker hosts from a single central instance. This allows you to aggregate container status, logs, and statistics from distributed infrastructure without deploying separate monitoring agents on every machine.

## Configuration Mechanism

Multi-host support is configured exclusively through the `DOCKER_HOSTS` environment variable. This variable accepts a comma-separated list of key-value pairs defining your environments.

### Format Specification

The configuration string follows this pattern:

```text
DOCKER_HOSTS=name1=connection_string1,name2=connection_string2,name3=connection_string3
```

- **Friendly Name**: An arbitrary alphanumeric identifier for the host (e.g., `remote-server`). This label appears in the UI drop-down menu.
- **Connection String**: The standard Docker connection URI (e.g., `ssh://user@host` or `tcp://host:port`).

## Supported Connection Protocols

### 1. Unix Socket (Local)
Standard connection for the host where VPS-Monitor is running.
- **URI Scheme**: `unix:///path/to/socket`

### 2. SSH (Secure Shell)
The recommended method for connecting to remote hosts. It provides encryption and authentication without exposing the Docker daemon port publicly.
- **URI Scheme**: `ssh://user@hostname` or `ssh://user@ip-address`
- **Requirements**: Public/Private key pair authentication configured.

### 3. TCP (Direct Network)
Direct connection to a Docker daemon listening on a network port.
- **URI Scheme**: `tcp://hostname:port`
- **Note**: Ensure the target Docker daemon is configured to listen on the specified TCP port (traditionally 2375 for unencrypted, 2376 for TLS).

## Configuration Examples

### Hybrid Local and Remote Setup
This configuration connects to the local machine and a remote satellite server over SSH.

```bash
DOCKER_HOSTS=hq-server=unix:///var/run/docker.sock,outpost-alpha=ssh://ops@10.50.12.5
```

### Distributed Infrastructure
A setup managing three distinct environments using different protocols.

```bash
DOCKER_HOSTS=mars-base=ssh://admin@mars.internal,jupiter-station=ssh://root@192.168.42.100,saturn-ring=tcp://saturn.ring.local:2375
```

### SSH Key Management

For SSH connections to work, the VPS-Monitor container must have access to a valid private key that is authorized on the target remote hosts.

#### 1. Generate Identity File
Create a dedicated SSH key pair for the monitor service:

```bash
ssh-keygen -t ed25519 -C "monitor-access-key" -f ./monitor_key
```

#### 2. Authorize Key on Remote Hosts
Copy the public key (`monitor_key.pub`) to the `~/.ssh/authorized_keys` file of the user you intend to connect as on each remote host.

```bash
ssh-copy-id -i ./monitor_key.pub ops@10.50.12.5
```

#### 3. Mount Keys in Docker Compose
Mount the directory containing your keys into the container. The container looks for keys in `/root/.ssh` by default.

```yaml
services:
  vps-monitor:
    image: ghcr.io/hhftechnology/vps-monitor:latest
    environment:
      - DOCKER_HOSTS=hq-server=unix:///var/run/docker.sock,outpost-alpha=ssh://ops@10.50.12.5
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      # Mount the folder containing your SSH keys
      - ./ssh-keys:/root/.ssh:ro
```

## Troubleshooting Connections

### Common Issues

**Host key verification failed**
The container does not know the fingerprint of the remote host.
- **Fix**: Manually connect once from the host machine to populate `known_hosts`, or mount your host's `known_hosts` file into the container.

**Permission denied (publickey)**
The private key is not readable or not authorized.
- **Fix**: Ensure the private key file has `600` permissions and is owned by the user running the process inside the container.

**Cannot connect to the Docker daemon**
The user on the remote host may not be in the `docker` group.
- **Fix**: Run `sudo usermod -aG docker <username>` on the remote machine.
