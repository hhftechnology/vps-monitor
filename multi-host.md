# Multi-Host Docker Configuration

## Overview

VPS-Monitor supports managing Docker containers across multiple Docker hosts simultaneously. This feature enables monitoring and controlling containers on local, remote, and SSH-connected Docker daemons from a single interface.

## Prerequisites

### Server Requirements

- Go 1.21 or higher
- Network connectivity to target Docker hosts
- Appropriate authentication credentials for remote hosts

### SSH-Based Connections

For SSH connections to remote Docker hosts:

- SSH client installed on the server
- SSH key-based authentication configured
- Valid SSH private key with appropriate permissions (0600)
- Docker daemon running on remote host
- User account with Docker socket permissions on remote host

### TCP-Based Connections

For TCP connections:

- Docker daemon configured to listen on TCP port (typically 2375 or 2376)
- Network access to the Docker daemon port
- TLS certificates (recommended for production)

## Configuration

### Environment Variable Format

Configure Docker hosts using the `DOCKER_HOSTS` environment variable. The format is:

```
DOCKER_HOSTS=name1=host1,name2=host2,name3=host3
```

Each host entry consists of:

- **name**: A friendly identifier for the host (alphanumeric, no spaces)
- **host**: The Docker daemon connection URL

The `=` delimiter separates the name from the host URL, while `,` separates multiple host entries.

### Supported Protocols

#### Unix Socket (Local)

```bash
DOCKER_HOSTS=local=unix:///var/run/docker.sock
```

Used for connecting to the local Docker daemon via Unix socket.

#### SSH Protocol

```bash
DOCKER_HOSTS=remote=ssh://user@hostname
DOCKER_HOSTS=remote=ssh://user@192.168.1.100
```

Used for secure connections to remote Docker daemons over SSH. Requires SSH key authentication.

#### TCP Protocol

```bash
DOCKER_HOSTS=remote=tcp://192.168.1.100:2375
DOCKER_HOSTS=secure=tcp://192.168.1.100:2376
```

Used for direct TCP connections to Docker daemons. Port 2376 typically indicates TLS encryption.

## Configuration Examples

### Single Local Host

```bash
DOCKER_HOSTS=local=unix:///var/run/docker.sock
```

Default configuration when `DOCKER_HOSTS` is not set.

### Local and Remote SSH

```bash
DOCKER_HOSTS=local=unix:///var/run/docker.sock,production=ssh://deploy@prod.example.com
```

### Multiple Remote Hosts

```bash
DOCKER_HOSTS=prod=ssh://deploy@prod.example.com,staging=ssh://deploy@staging.example.com,dev=tcp://dev.example.com:2375
```

### Complex Multi-Environment Setup

```bash
DOCKER_HOSTS=local=unix:///var/run/docker.sock,prod-us=ssh://root@us-prod.example.com,prod-eu=ssh://root@eu-prod.example.com,staging=tcp://staging.example.com:2375
```

## SSH Configuration

### SSH Key Setup

1. Generate SSH key pair if not already available:

```bash
ssh-keygen -t ed25519 -C "vps-monitor-docker-access"
```

2. Copy public key to remote host:

```bash
ssh-copy-id user@remote-host
```

3. Verify SSH access:

```bash
ssh user@remote-host docker ps
```

### Docker Compose Configuration

When running VPS-Monitor via Docker Compose, mount the SSH directory:

```yaml
services:
  vps-monitor:
    volumes:
      - ~/.ssh:/root/.ssh:ro
```

This provides the container access to your SSH keys for authentication.

### SSH Agent Forwarding

For enhanced security, use SSH agent forwarding instead of mounting keys:

```yaml
services:
  vps-monitor:
    environment:
      - SSH_AUTH_SOCK=/ssh-agent
    volumes:
      - ${SSH_AUTH_SOCK}:/ssh-agent
```

### Host Key Verification

Add remote hosts to `known_hosts` to avoid verification prompts:

```bash
ssh-keyscan remote-host >> ~/.ssh/known_hosts
```

Or disable strict host key checking (not recommended for production):

```bash
# In ~/.ssh/config
Host *
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
```

## Deployment Scenarios

### Standalone Server

```bash
export DOCKER_HOSTS="local=unix:///var/run/docker.sock,remote=ssh://user@remote-host"
./vps-monitor-home
```

### Docker Compose

```yaml
services:
  vps-monitor:
    image: vps-monitor:latest
    ports:
      - "6789:6789"
    environment:
      - DOCKER_HOSTS=local=unix:///var/run/docker.sock,remote=ssh://deploy@prod.example.com
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ~/.ssh:/root/.ssh:ro
```

## Troubleshooting

### Connection Issues

**Symptom**: Cannot connect to remote Docker host

**Solutions**:

1. Verify network connectivity:

   ```bash
   ping remote-host
   telnet remote-host 22  # for SSH
   telnet remote-host 2375  # for TCP
   ```

2. Check SSH authentication:

   ```bash
   ssh -v user@remote-host docker ps
   ```

3. Verify Docker daemon is running:

   ```bash
   ssh user@remote-host systemctl status docker
   ```

4. Check Docker daemon configuration:
   ```bash
   ssh user@remote-host cat /etc/docker/daemon.json
   ```

### Permission Issues

**Symptom**: Permission denied errors when accessing Docker

**Solutions**:

1. Add user to docker group on remote host:

   ```bash
   sudo usermod -aG docker username
   ```

2. Verify Docker socket permissions:

   ```bash
   ls -l /var/run/docker.sock
   ```

3. Check SELinux/AppArmor policies if applicable

### SSH Key Issues

**Symptom**: SSH authentication failures

**Solutions**:

1. Verify key permissions:

   ```bash
   chmod 600 ~/.ssh/id_ed25519
   chmod 644 ~/.ssh/id_ed25519.pub
   chmod 700 ~/.ssh
   ```

2. Check SSH agent:

   ```bash
   ssh-add -l
   ssh-add ~/.ssh/id_ed25519
   ```

3. Test SSH connection:
   ```bash
   ssh -vvv user@remote-host
   ```

### Invalid Configuration

**Symptom**: Server fails to start with configuration error

**Solutions**:

1. Validate `DOCKER_HOSTS` format:

   - Ensure `=` separator between name and host
   - Ensure `,` separator between entries
   - Check for trailing commas or spaces

2. Verify host URLs:

   - SSH: `ssh://user@host` (not `ssh:user@host`)
   - TCP: `tcp://host:port` (not `tcp:host:port`)
   - Unix: `unix:///path/to/socket` (three slashes)

3. Check for special characters in host names:
   - Use only alphanumeric characters and hyphens
   - Avoid spaces, special characters in friendly names

## Migration from Single-Host

### Upgrading Existing Deployments

1. Current single-host setups will continue to work without changes
2. Default configuration uses local Unix socket if `DOCKER_HOSTS` is not set
3. Add `DOCKER_HOSTS` environment variable to expand to multiple hosts
4. No database migration required
5. Frontend automatically adapts to single or multi-host mode

### Backward Compatibility

The system maintains backward compatibility:

- Existing container operations work on the default host
- API responses include host information even for single-host setups
- Frontend displays host filter even with single host configured
- No breaking changes to existing API contracts
