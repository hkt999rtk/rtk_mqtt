# MCP Host CES2026 - Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying MCP Host CES2026 in various environments, from local development to production systems.

## Quick Start Deployment

### 1. Extract Release Package

```bash
# Download and extract release
wget https://github.com/your-repo/releases/mcphost_ces2026_linux_amd64_v20250816.tar.gz
tar -xzf mcphost_ces2026_linux_amd64_v20250816.tar.gz
cd mcphost_ces2026_linux_amd64_v20250816/
```

### 2. Configure Services

Edit configuration files:

**MCP Server Configuration** (`mcp_server_config.json`):
```json
{
  "server": {
    "name": "Production MCP Server",
    "version": "1.0.0"
  },
  "weatherapi": {
    "api_key": "your_weatherapi_key_here"
  },
  "http_server": {
    "port": 8081,
    "host": "0.0.0.0"
  }
}
```

**MCP Client Configuration** (`mcp_client_config.json`):
```json
{
  "server": {
    "name": "Production MCP Client",
    "version": "1.0.0"
  },
  "llm_providers": [
    {
      "name": "lmstudio",
      "type": "lmstudio",
      "base_url": "http://localhost:1234",
      "enabled": true
    }
  ],
  "mcp_servers": [
    {
      "name": "local_server",
      "url": "http://localhost:8081",
      "enabled": true
    }
  ],
  "http_server": {
    "port": 8080,
    "host": "0.0.0.0"
  }
}
```

### 3. Start Services

```bash
# Start MCP Server
./start_server.sh

# Start MCP Client (in another terminal)
./start_client.sh
```

### 4. Verify Deployment

```bash
# Test health endpoints
curl http://localhost:8080/health  # MCP Client
curl http://localhost:8081/health  # MCP Server

# Test API functionality
curl http://localhost:8080/v1/models

# Run comprehensive tests
./test_script/test_http_communication.sh
```

## Environment-Specific Deployments

### Development Environment

**Requirements**:
- Go 1.21+ for building from source
- LM Studio or similar LLM backend
- WeatherAPI.com account (free tier)

**Setup**:
```bash
# Clone and build from source
git clone <repository-url>
cd mcphost_ces2026

# Build components
cd build
make all

# Start development services
cd ../mcp_server && ./mcp_server &
cd ../mcp_client && ./mcp_client &

# Test with interactive mode
./mcp_client/mcp_client -interactive
```

### Staging Environment

**Requirements**:
- Linux server (Ubuntu 20.04+ recommended)
- Reverse proxy (nginx/Apache)
- SSL certificate
- External LLM backend

**Configuration**:

`/etc/nginx/sites-available/mcphost`:
```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location /v1/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    location /health {
        proxy_pass http://localhost:8080;
    }
}
```

Systemd service files:

`/etc/systemd/system/mcp-server.service`:
```ini
[Unit]
Description=MCP Server
After=network.target

[Service]
Type=simple
User=mcphost
WorkingDirectory=/opt/mcphost
ExecStart=/opt/mcphost/mcp_server -config /opt/mcphost/mcp_server_config.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

`/etc/systemd/system/mcp-client.service`:
```ini
[Unit]
Description=MCP Client
After=network.target mcp-server.service
Requires=mcp-server.service

[Service]
Type=simple
User=mcphost
WorkingDirectory=/opt/mcphost
ExecStart=/opt/mcphost/mcp_client -config /opt/mcphost/mcp_client_config.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**Deployment Commands**:
```bash
# Create user and directories
sudo useradd -r -s /bin/false mcphost
sudo mkdir -p /opt/mcphost
sudo chown mcphost:mcphost /opt/mcphost

# Deploy binaries and configs
sudo cp mcp_server mcp_client /opt/mcphost/
sudo cp *_config.json /opt/mcphost/
sudo chown mcphost:mcphost /opt/mcphost/*

# Enable and start services
sudo systemctl enable mcp-server mcp-client
sudo systemctl start mcp-server mcp-client

# Check status
sudo systemctl status mcp-server mcp-client
```

### Production Environment

**Requirements**:
- High-availability Linux servers
- Load balancer
- SSL/TLS termination
- Monitoring and logging
- Backup strategy

**Architecture**:
```
Internet → Load Balancer → MCP Client (multiple) → MCP Server (multiple)
```

**Load Balancer Configuration** (HAProxy example):

`/etc/haproxy/haproxy.cfg`:
```
frontend mcp_frontend
    bind *:80
    bind *:443 ssl crt /etc/ssl/certs/your-cert.pem
    redirect scheme https if !{ ssl_fc }
    default_backend mcp_client_backend

backend mcp_client_backend
    balance roundrobin
    option httpchk GET /health
    server client1 10.0.1.10:8080 check
    server client2 10.0.1.11:8080 check
    server client3 10.0.1.12:8080 check
```

**Production Configuration** (`mcp_client_config.json`):
```json
{
  "server": {
    "name": "Production MCP Client",
    "version": "1.0.0"
  },
  "llm_providers": [
    {
      "name": "openai",
      "type": "openai",
      "base_url": "https://api.openai.com/v1",
      "api_key": "${OPENAI_API_KEY}",
      "models": ["gpt-4", "gpt-3.5-turbo"],
      "enabled": true
    }
  ],
  "mcp_servers": [
    {
      "name": "server_cluster",
      "url": "http://internal-lb.example.com:8081",
      "enabled": true
    }
  ],
  "http_server": {
    "port": 8080,
    "host": "0.0.0.0",
    "read_timeout": "30s",
    "write_timeout": "30s"
  }
}
```

## Platform-Specific Deployments

### Linux (Ubuntu/Debian)

**Package Installation**:
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y wget curl nginx

# Create deployment directory
sudo mkdir -p /opt/mcphost
cd /opt/mcphost

# Download and extract
wget https://github.com/your-repo/releases/mcphost_ces2026_linux_amd64_v20250816.tar.gz
sudo tar -xzf mcphost_ces2026_linux_amd64_v20250816.tar.gz --strip-components=1

# Set permissions
sudo chmod +x mcp_server mcp_client
```

### Linux (ARM64 - AWS Graviton, Raspberry Pi)

```bash
# Download ARM64 release
wget https://github.com/your-repo/releases/mcphost_ces2026_linux_arm64_v20250816.tar.gz
tar -xzf mcphost_ces2026_linux_arm64_v20250816.tar.gz
cd mcphost_ces2026_linux_arm64_v20250816/

# Run deployment
./start_server.sh &
./start_client.sh &
```

### macOS (Development)

```bash
# Download macOS release
wget https://github.com/your-repo/releases/mcphost_ces2026_darwin_arm64_v20250816.tar.gz
tar -xzf mcphost_ces2026_darwin_arm64_v20250816.tar.gz
cd mcphost_ces2026_darwin_arm64_v20250816/

# Make executable (macOS security)
chmod +x mcp_server mcp_client

# Run services
./start_server.sh &
./start_client.sh &
```

## Configuration Management

### Environment Variables

Both services support environment variable overrides:

```bash
# Server configuration
export MCP_SERVER_PORT=8081
export MCP_SERVER_HOST=0.0.0.0
export WEATHER_API_KEY=your_weather_key

# Client configuration
export MCP_CLIENT_PORT=8080
export MCP_CLIENT_HOST=0.0.0.0
export OPENAI_API_KEY=your_openai_key
export LM_STUDIO_URL=http://localhost:1234

# Start with environment variables
./mcp_server -config mcp_server_config.json
./mcp_client -config mcp_client_config.json
```

### Configuration Templates

**Minimal Production Config** (`mcp_client_config.json`):
```json
{
  "llm_providers": [
    {
      "name": "production_llm",
      "type": "openai",
      "base_url": "${LLM_BACKEND_URL}",
      "api_key": "${LLM_API_KEY}",
      "enabled": true
    }
  ],
  "mcp_servers": [
    {
      "name": "production_server",
      "url": "${MCP_SERVER_URL}",
      "enabled": true
    }
  ],
  "http_server": {
    "port": 8080,
    "host": "0.0.0.0"
  }
}
```

### Secret Management

**Using External Secret Management**:
```bash
# AWS Secrets Manager
export WEATHER_API_KEY=$(aws secretsmanager get-secret-value --secret-id weather-api-key --query SecretString --output text)

# HashiCorp Vault
export OPENAI_API_KEY=$(vault kv get -field=api_key secret/openai)

# Kubernetes Secrets
kubectl create secret generic mcp-secrets \
  --from-literal=weather-api-key=your_key \
  --from-literal=openai-api-key=your_key
```

## Monitoring and Logging

### Health Check Endpoints

Both services provide health check endpoints:

```bash
# Basic health check
curl http://localhost:8080/health
curl http://localhost:8081/health

# Detailed status (returns JSON with service info)
curl -H "Accept: application/json" http://localhost:8080/health
```

### Logging Configuration

**Structured Logging** (JSON format):
```json
{
  "timestamp": "2025-08-16T12:00:00Z",
  "level": "INFO",
  "component": "mcp_client",
  "message": "Tool call completed successfully",
  "tool_name": "get_weather",
  "execution_time": "234ms",
  "request_id": "req_123456"
}
```

**Log Aggregation** (using rsyslog):

`/etc/rsyslog.d/50-mcphost.conf`:
```
# MCP Host logs
if $programname == 'mcp_server' then /var/log/mcphost/server.log
if $programname == 'mcp_client' then /var/log/mcphost/client.log
& stop
```

### Monitoring with Prometheus

**Metrics Endpoints** (future feature):
```bash
# Server metrics
curl http://localhost:8081/metrics

# Client metrics  
curl http://localhost:8080/metrics
```

**Sample Metrics**:
- `mcp_requests_total` - Total API requests
- `mcp_request_duration_seconds` - Request latency
- `mcp_tool_calls_total` - Tool execution count
- `mcp_errors_total` - Error count by type

## Security Configuration

### Network Security

**Firewall Rules** (iptables):
```bash
# Allow only necessary ports
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT  # MCP Client
sudo iptables -A INPUT -p tcp --dport 8081 -s 127.0.0.1 -j ACCEPT  # MCP Server (localhost only)
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT   # SSH
sudo iptables -A INPUT -j DROP  # Drop all other traffic
```

**TLS Configuration** (nginx):
```nginx
server {
    listen 443 ssl http2;
    ssl_certificate /etc/ssl/certs/your-cert.pem;
    ssl_certificate_key /etc/ssl/private/your-key.pem;
    
    # Security headers
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";
    
    location /v1/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Access Control

**API Key Authentication** (future feature):
```json
{
  "auth": {
    "enabled": true,
    "type": "api_key",
    "header": "X-API-Key",
    "keys": [
      {
        "key": "mcp_key_123456",
        "name": "production_client",
        "permissions": ["read", "execute"]
      }
    ]
  }
}
```

## Backup and Recovery

### Configuration Backup

```bash
#!/bin/bash
# backup-configs.sh

BACKUP_DIR="/backup/mcphost/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Backup configuration files
cp /opt/mcphost/*_config.json "$BACKUP_DIR/"

# Backup systemd service files
cp /etc/systemd/system/mcp-*.service "$BACKUP_DIR/"

# Create archive
tar -czf "$BACKUP_DIR.tar.gz" -C "$BACKUP_DIR" .
rm -rf "$BACKUP_DIR"

echo "Backup created: $BACKUP_DIR.tar.gz"
```

### Service Recovery

```bash
#!/bin/bash
# recover-services.sh

# Stop services
sudo systemctl stop mcp-client mcp-server

# Restore from backup
BACKUP_FILE="$1"
if [[ -f "$BACKUP_FILE" ]]; then
    tar -xzf "$BACKUP_FILE" -C /opt/mcphost/
    sudo chown mcphost:mcphost /opt/mcphost/*
    
    # Restart services
    sudo systemctl start mcp-server
    sleep 5
    sudo systemctl start mcp-client
    
    echo "Services recovered successfully"
else
    echo "Backup file not found: $BACKUP_FILE"
    exit 1
fi
```

## Performance Tuning

### System Optimization

**Linux Kernel Parameters** (`/etc/sysctl.conf`):
```
# Network optimization
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 65536 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216

# File descriptor limits
fs.file-max = 65536
```

**Service Limits** (`/etc/security/limits.conf`):
```
mcphost soft nofile 65536
mcphost hard nofile 65536
mcphost soft nproc 4096
mcphost hard nproc 4096
```

### Application Tuning

**Go Runtime Optimization**:
```bash
# Increase Go garbage collector target
export GOGC=200

# Set maximum number of OS threads
export GOMAXPROCS=4

# Start services with optimizations
./mcp_server -config mcp_server_config.json
./mcp_client -config mcp_client_config.json
```

## Troubleshooting

### Common Deployment Issues

**Port Already in Use**:
```bash
# Check what's using the port
sudo netstat -tulpn | grep :8080
sudo lsof -i :8080

# Kill process using port
sudo kill -9 $(sudo lsof -t -i:8080)
```

**Permission Denied**:
```bash
# Check file permissions
ls -la mcp_server mcp_client

# Fix permissions
chmod +x mcp_server mcp_client
sudo chown mcphost:mcphost /opt/mcphost/*
```

**Service Won't Start**:
```bash
# Check systemd status
sudo systemctl status mcp-server
sudo systemctl status mcp-client

# View logs
sudo journalctl -u mcp-server -f
sudo journalctl -u mcp-client -f
```

### Performance Issues

**High Memory Usage**:
```bash
# Monitor memory usage
top -p $(pgrep mcp_server)
top -p $(pgrep mcp_client)

# Check for memory leaks
valgrind --tool=memcheck --leak-check=full ./mcp_server
```

**Slow Response Times**:
```bash
# Test response times
time curl http://localhost:8080/v1/models
time curl http://localhost:8081/tools/list

# Profile application
go tool pprof http://localhost:8080/debug/pprof/profile
```

### Connectivity Issues

**Network Connectivity**:
```bash
# Test internal communication
curl -v http://localhost:8081/health
curl -v http://localhost:8080/health

# Test external access
curl -v http://your-server:8080/v1/models

# Check network configuration
ip addr show
ss -tulpn | grep 808
```

## Maintenance

### Regular Maintenance Tasks

**Daily**:
- Monitor service health
- Check error logs
- Verify backup completion

**Weekly**:
- Review performance metrics
- Update security patches
- Test disaster recovery

**Monthly**:
- Update dependencies
- Review configuration
- Capacity planning

### Update Procedure

```bash
#!/bin/bash
# update-mcphost.sh

# Download new release
NEW_VERSION="v20250901"
wget "https://github.com/your-repo/releases/mcphost_ces2026_linux_amd64_${NEW_VERSION}.tar.gz"

# Backup current installation
./backup-configs.sh

# Stop services
sudo systemctl stop mcp-client mcp-server

# Update binaries
tar -xzf "mcphost_ces2026_linux_amd64_${NEW_VERSION}.tar.gz"
sudo cp mcphost_ces2026_linux_amd64_${NEW_VERSION}/mcp_* /opt/mcphost/
sudo chown mcphost:mcphost /opt/mcphost/mcp_*

# Start services
sudo systemctl start mcp-server
sleep 5
sudo systemctl start mcp-client

# Verify update
curl http://localhost:8080/health
curl http://localhost:8081/health

echo "Update completed successfully"
```

This deployment guide provides comprehensive instructions for deploying MCP Host CES2026 across different environments and platforms, with proper security, monitoring, and maintenance procedures.