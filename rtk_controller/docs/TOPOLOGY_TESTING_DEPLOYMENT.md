# RTK Controller Topology - Testing & Deployment Guide

## Table of Contents
1. [System Requirements](#system-requirements)
2. [Build Instructions](#build-instructions)
3. [Testing Procedures](#testing-procedures)
4. [Deployment Guide](#deployment-guide)
5. [Integration Testing](#integration-testing)
6. [Performance Testing](#performance-testing)
7. [Monitoring & Maintenance](#monitoring--maintenance)

## System Requirements

### Development Environment
- Go 1.19 or higher
- Git 2.0 or higher
- 4GB RAM minimum
- 1GB free disk space

### Runtime Environment
- Linux/macOS/Windows (64-bit)
- 512MB RAM minimum
- 100MB free disk space
- Network connectivity for MQTT

## Build Instructions

### 1. Clone Repository
```bash
git clone https://github.com/your-org/rtk_controller.git
cd rtk_controller
```

### 2. Install Dependencies
```bash
go mod download
go mod verify
```

### 3. Build Binary
```bash
# Development build
go build -o rtk-controller ./cmd/controller

# Production build (optimized)
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o rtk-controller ./cmd/controller

# Cross-platform builds
make build-all  # Builds for linux/amd64, darwin/amd64, windows/amd64
```

### 4. Verify Build
```bash
./rtk-controller --version
./rtk-controller --help
```

## Testing Procedures

### Unit Tests

#### 1. Run All Tests
```bash
go test ./...
```

#### 2. Run Topology Tests
```bash
go test ./internal/topology/...
```

#### 3. Run with Coverage
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

#### 1. Load Sample Data
```bash
# Generate test data
go run test/topology_test_simple.go > test/sample_topology.json

# Load into database
go run test/load_topology.go
```

#### 2. Test CLI Commands
```bash
# Start interactive CLI
./rtk-controller --cli

# Test commands
> topology show
> topology devices
> help topology
> exit
```

#### 3. Test MQTT Processing
```bash
# Simulate MQTT messages
go run test/test_topology_simple.go

# Verify data processing
./rtk-controller --cli
> topology show
```

### Automated Test Suite

Create `test/run_tests.sh`:
```bash
#!/bin/bash

echo "Running RTK Controller Test Suite"
echo "================================="

# Unit tests
echo -e "\n1. Running unit tests..."
go test ./... -v

# Build test
echo -e "\n2. Testing build..."
go build -o test-binary ./cmd/controller
if [ $? -eq 0 ]; then
    echo "✓ Build successful"
    rm test-binary
else
    echo "✗ Build failed"
    exit 1
fi

# Data loading test
echo -e "\n3. Testing data loading..."
go run test/topology_test_simple.go > /tmp/test_topology.json
go run test/load_topology.go

# CLI test
echo -e "\n4. Testing CLI..."
echo -e "topology show\nexit" | ./rtk-controller --cli > /tmp/cli_test.log 2>&1
if grep -q "devices" /tmp/cli_test.log; then
    echo "✓ CLI test passed"
else
    echo "✗ CLI test failed"
    exit 1
fi

echo -e "\n✓ All tests passed!"
```

## Deployment Guide

### 1. Configuration Setup

Create `configs/production.yaml`:
```yaml
# Server Configuration
server:
  host: "0.0.0.0"
  port: 8080
  mode: "production"

# Storage Configuration
storage:
  type: "buntdb"
  path: "/var/lib/rtk-controller/data"
  backup:
    enabled: true
    interval: "24h"
    retention: 7

# MQTT Configuration
mqtt:
  broker: "tcp://mqtt.your-domain.com:1883"
  client_id: "rtk-controller-prod"
  username: "${MQTT_USERNAME}"
  password: "${MQTT_PASSWORD}"
  topics:
    - "rtk/v1/+/+/+/topology/#"
    - "rtk/v1/+/+/+/telemetry/#"
    - "rtk/v1/+/+/+/device/#"
    - "rtk/v1/+/+/+/diagnostics/#"

# Topology Configuration
topology:
  tenant: "production"
  site: "main"
  update_interval: "30s"
  cleanup_interval: "5m"
  device_offline_timeout: "10m"
  
# Logging
logging:
  level: "info"
  file: "/var/log/rtk-controller/controller.log"
  max_size: 100  # MB
  max_age: 30    # days
  max_backups: 10
```

### 2. Systemd Service

Create `/etc/systemd/system/rtk-controller.service`:
```ini
[Unit]
Description=RTK Controller - Network Topology Management
After=network.target

[Service]
Type=simple
User=rtk
Group=rtk
WorkingDirectory=/opt/rtk-controller
ExecStart=/opt/rtk-controller/rtk-controller --config /etc/rtk-controller/production.yaml
Restart=always
RestartSec=10
StandardOutput=append:/var/log/rtk-controller/service.log
StandardError=append:/var/log/rtk-controller/error.log

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/rtk-controller /var/log/rtk-controller

[Install]
WantedBy=multi-user.target
```

### 3. Docker Deployment

Create `Dockerfile`:
```dockerfile
# Build stage
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o rtk-controller ./cmd/controller

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/rtk-controller .
COPY configs/production.yaml /etc/rtk-controller/config.yaml
EXPOSE 8080
VOLUME ["/var/lib/rtk-controller", "/var/log/rtk-controller"]
ENTRYPOINT ["./rtk-controller"]
CMD ["--config", "/etc/rtk-controller/config.yaml"]
```

Create `docker-compose.yml`:
```yaml
version: '3.8'

services:
  rtk-controller:
    build: .
    container_name: rtk-controller
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - rtk-data:/var/lib/rtk-controller
      - rtk-logs:/var/log/rtk-controller
      - ./configs/production.yaml:/etc/rtk-controller/config.yaml:ro
    environment:
      - MQTT_USERNAME=${MQTT_USERNAME}
      - MQTT_PASSWORD=${MQTT_PASSWORD}
    networks:
      - rtk-network

  mqtt-broker:
    image: eclipse-mosquitto:latest
    container_name: rtk-mqtt
    restart: unless-stopped
    ports:
      - "1883:1883"
      - "9001:9001"
    volumes:
      - mqtt-data:/mosquitto/data
      - mqtt-logs:/mosquitto/log
      - ./configs/mosquitto.conf:/mosquitto/config/mosquitto.conf:ro
    networks:
      - rtk-network

volumes:
  rtk-data:
  rtk-logs:
  mqtt-data:
  mqtt-logs:

networks:
  rtk-network:
    driver: bridge
```

### 4. Kubernetes Deployment

Create `k8s-deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rtk-controller
  namespace: rtk-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: rtk-controller
  template:
    metadata:
      labels:
        app: rtk-controller
    spec:
      containers:
      - name: rtk-controller
        image: your-registry/rtk-controller:latest
        ports:
        - containerPort: 8080
        env:
        - name: MQTT_USERNAME
          valueFrom:
            secretKeyRef:
              name: rtk-secrets
              key: mqtt-username
        - name: MQTT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: rtk-secrets
              key: mqtt-password
        volumeMounts:
        - name: config
          mountPath: /etc/rtk-controller
        - name: data
          mountPath: /var/lib/rtk-controller
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: config
        configMap:
          name: rtk-controller-config
      - name: data
        persistentVolumeClaim:
          claimName: rtk-controller-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: rtk-controller-service
  namespace: rtk-system
spec:
  selector:
    app: rtk-controller
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: rtk-controller-pvc
  namespace: rtk-system
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

## Integration Testing

### 1. MQTT Integration Test
```bash
# Start MQTT broker
docker run -d -p 1883:1883 eclipse-mosquitto

# Start controller
./rtk-controller --config configs/test.yaml &

# Publish test messages
go run test/mqtt_topology_test.go

# Verify processing
curl http://localhost:8080/api/topology
```

### 2. Load Testing
```bash
# Generate large dataset
go run test/generate_load_test.go --devices 1000 --connections 5000

# Load and measure performance
time go run test/load_topology.go

# Check memory usage
ps aux | grep rtk-controller
```

### 3. End-to-End Test
```bash
# Start full stack
docker-compose up -d

# Wait for services
sleep 10

# Run E2E tests
go run test/e2e_test.go

# Check results
docker-compose logs rtk-controller
```

## Performance Testing

### Benchmark Tests
```go
// test/benchmark_test.go
package test

import (
    "testing"
    "rtk_controller/internal/topology"
)

func BenchmarkTopologyLoad(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // Load topology
    }
}

func BenchmarkDeviceUpdate(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // Update device
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./test/...
```

### Stress Testing
```bash
# Generate stress test script
cat > test/stress_test.sh << 'EOF'
#!/bin/bash
# Stress test with concurrent connections

for i in {1..100}; do
    (
        echo "topology show" | ./rtk-controller --cli > /dev/null 2>&1
    ) &
done
wait

echo "Stress test completed"
EOF

chmod +x test/stress_test.sh
./test/stress_test.sh
```

## Monitoring & Maintenance

### 1. Health Check Endpoint
```go
// Add to main.go
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"healthy"}`))
})
```

### 2. Metrics Collection
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'rtk-controller'
    static_configs:
    - targets: ['localhost:8080']
```

### 3. Log Rotation
```bash
# /etc/logrotate.d/rtk-controller
/var/log/rtk-controller/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0644 rtk rtk
    postrotate
        systemctl reload rtk-controller
    endscript
}
```

### 4. Backup Strategy
```bash
#!/bin/bash
# backup.sh
BACKUP_DIR="/backup/rtk-controller"
DATA_DIR="/var/lib/rtk-controller"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR
tar -czf $BACKUP_DIR/rtk-backup-$DATE.tar.gz $DATA_DIR

# Keep only last 7 days
find $BACKUP_DIR -name "rtk-backup-*.tar.gz" -mtime +7 -delete
```

### 5. Monitoring Dashboard
```yaml
# grafana-dashboard.json
{
  "dashboard": {
    "title": "RTK Controller Topology",
    "panels": [
      {
        "title": "Total Devices",
        "query": "count(rtk_topology_devices)"
      },
      {
        "title": "Online Devices",
        "query": "count(rtk_topology_devices{status='online'})"
      },
      {
        "title": "Connections",
        "query": "count(rtk_topology_connections)"
      },
      {
        "title": "MQTT Messages/sec",
        "query": "rate(rtk_mqtt_messages_total[1m])"
      }
    ]
  }
}
```

## Troubleshooting

### Common Issues

#### 1. Database Lock Error
```bash
# Fix: Remove lock file
rm /var/lib/rtk-controller/data/controller.db/LOCK
```

#### 2. MQTT Connection Failed
```bash
# Check connectivity
telnet mqtt.broker.com 1883

# Verify credentials
mosquitto_sub -h mqtt.broker.com -t "rtk/#" -u username -P password
```

#### 3. High Memory Usage
```bash
# Analyze memory profile
go tool pprof http://localhost:8080/debug/pprof/heap

# Restart service
systemctl restart rtk-controller
```

## Security Checklist

- [ ] Change default passwords
- [ ] Enable TLS for MQTT
- [ ] Restrict API access
- [ ] Regular security updates
- [ ] Log monitoring
- [ ] Backup encryption
- [ ] Network isolation
- [ ] Rate limiting

## Support

For issues and questions:
- GitHub Issues: https://github.com/your-org/rtk_controller/issues
- Documentation: https://docs.rtk-controller.io
- Community Forum: https://forum.rtk-controller.io

---
Last Updated: 2025-08-17