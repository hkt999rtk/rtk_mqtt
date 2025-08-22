# RTK MQTT 部署指南

## 概述

本指南提供RTK MQTT系統在不同環境中的完整部署方案，包含開發、測試、預生產和生產環境的配置、監控和維護策略。

## 架構概覽

### 系統組件

```
┌─────────────────────────────────────────────────────────┐
│                 RTK MQTT System                         │
├─────────────────────────────────────────────────────────┤
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  RTK Devices  │  │ MQTT Broker  │  │ RTK Controller│  │
│  │  (IoT/Network)│  │ (Messaging)  │  │ (Management) │  │
│  └───────────────┘  └──────────────┘  └──────────────┘  │
├─────────────────────────────────────────────────────────┤
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Database    │  │  Monitoring  │  │   Security   │  │
│  │   (BuntDB)    │  │ (Prometheus) │  │    (TLS)     │  │
│  └───────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### 部署模式

#### 單機部署 (All-in-One)
- 適用於開發和小型環境
- 所有組件運行在同一伺服器
- 簡化配置和維護

#### 分散式部署 (Distributed)
- 適用於生產環境
- 組件分離，提高可用性
- 支持水平擴展

#### 容器化部署 (Containerized)
- 使用Docker/Kubernetes
- 標準化部署流程
- 簡化運維管理

## 環境配置

### 開發環境

#### 系統需求
```bash
# 最低配置
CPU: 2 cores
Memory: 4GB RAM
Storage: 20GB SSD
OS: Ubuntu 20.04 LTS / macOS / Windows 10

# 網路
MQTT Port: 1883
WebSocket Port: 8083
Controller API: 8080
```

#### 快速設置
```bash
#!/bin/bash
# dev_setup.sh

# 創建工作目錄
mkdir -p ~/rtk_mqtt_dev
cd ~/rtk_mqtt_dev

# 下載代碼
git clone https://github.com/your-org/rtk_mqtt.git
cd rtk_mqtt

# 設置環境變數
export RTK_ENV=development
export RTK_MQTT_BROKER=localhost:1883
export RTK_DATA_DIR=./data
export RTK_LOG_LEVEL=debug

# 安裝依賴
sudo apt-get update
sudo apt-get install -y mosquitto mosquitto-clients
pip3 install -r requirements.txt

# 啟動Mosquitto
sudo systemctl start mosquitto
sudo systemctl enable mosquitto

# 編譯RTK Controller
cd rtk_controller
make build

# 創建配置文件
cp configs/dev.yaml.example configs/dev.yaml

# 啟動Controller
./build_dir/rtk_controller --config configs/dev.yaml --cli
```

#### 開發配置檔案
```yaml
# configs/dev.yaml
environment: development

mqtt:
  broker_host: "localhost"
  broker_port: 1883
  client_id: "rtk_controller_dev"
  keepalive: 60
  clean_session: true
  
database:
  type: "buntdb"
  path: "./data/dev"
  
logging:
  level: "debug"
  format: "text"
  output: "stdout"
  
api:
  enabled: true
  host: "0.0.0.0"
  port: 8080
  
diagnostics:
  enabled: true
  interval_s: 60
  
qos:
  enabled: true
  analysis_interval_s: 30
```

### 測試環境

#### Docker Compose設置
```yaml
# docker-compose.test.yml
version: '3.8'

services:
  mosquitto:
    image: eclipse-mosquitto:2.0
    container_name: rtk-mosquitto-test
    ports:
      - "1883:1883"
      - "8083:8083"
    volumes:
      - ./mosquitto/config:/mosquitto/config
      - ./mosquitto/data:/mosquitto/data
      - ./mosquitto/log:/mosquitto/log
    restart: unless-stopped
    
  rtk-controller:
    build:
      context: ./rtk_controller
      dockerfile: Dockerfile
    container_name: rtk-controller-test
    ports:
      - "8080:8080"
    environment:
      - RTK_ENV=testing
      - RTK_MQTT_BROKER=mosquitto:1883
      - RTK_LOG_LEVEL=info
    volumes:
      - ./data:/app/data
      - ./configs:/app/configs
    depends_on:
      - mosquitto
    restart: unless-stopped
    
  prometheus:
    image: prom/prometheus:latest
    container_name: rtk-prometheus-test
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    restart: unless-stopped
    
  grafana:
    image: grafana/grafana:latest
    container_name: rtk-grafana-test
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-storage:/var/lib/grafana
      - ./monitoring/grafana:/etc/grafana/provisioning
    restart: unless-stopped

volumes:
  grafana-storage:
```

#### 測試環境啟動腳本
```bash
#!/bin/bash
# test_deploy.sh

set -e

echo "Deploying RTK MQTT Test Environment..."

# 創建必要目錄
mkdir -p mosquitto/{config,data,log}
mkdir -p data monitoring

# 生成Mosquitto配置
cat > mosquitto/config/mosquitto.conf << EOF
# Mosquitto Test Configuration
listener 1883
listener 8083
protocol websockets

allow_anonymous true
log_dest file /mosquitto/log/mosquitto.log
log_type all
connection_messages true
EOF

# 生成Prometheus配置
cat > monitoring/prometheus.yml << EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'rtk-controller'
    static_configs:
      - targets: ['rtk-controller:8080']
    metrics_path: '/metrics'
    
  - job_name: 'mosquitto'
    static_configs:
      - targets: ['mosquitto:1883']
EOF

# 啟動服務
docker-compose -f docker-compose.test.yml up -d

echo "Waiting for services to start..."
sleep 30

# 驗證服務
echo "Verifying services..."
curl -f http://localhost:8080/health || echo "RTK Controller not ready"
curl -f http://localhost:9090 || echo "Prometheus not ready"
curl -f http://localhost:3000 || echo "Grafana not ready"

echo "Test environment deployed successfully!"
echo "Access points:"
echo "  RTK Controller: http://localhost:8080"
echo "  Prometheus: http://localhost:9090"
echo "  Grafana: http://localhost:3000 (admin/admin)"
```

### 生產環境

#### 系統需求

```bash
# 生產環境最低配置
# MQTT Broker節點
CPU: 4 cores
Memory: 8GB RAM
Storage: 100GB SSD
Network: 1Gbps

# RTK Controller節點
CPU: 2 cores
Memory: 4GB RAM
Storage: 50GB SSD

# 監控節點
CPU: 2 cores
Memory: 8GB RAM
Storage: 200GB SSD
```

#### 高可用性配置

```yaml
# kubernetes/namespace.yml
apiVersion: v1
kind: Namespace
metadata:
  name: rtk-mqtt
---
# kubernetes/mosquitto-cluster.yml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mosquitto-cluster
  namespace: rtk-mqtt
spec:
  serviceName: mosquitto-cluster
  replicas: 3
  selector:
    matchLabels:
      app: mosquitto
  template:
    metadata:
      labels:
        app: mosquitto
    spec:
      containers:
      - name: mosquitto
        image: eclipse-mosquitto:2.0
        ports:
        - containerPort: 1883
          name: mqtt
        - containerPort: 8083
          name: websocket
        volumeMounts:
        - name: mosquitto-config
          mountPath: /mosquitto/config
        - name: mosquitto-data
          mountPath: /mosquitto/data
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
      volumes:
      - name: mosquitto-config
        configMap:
          name: mosquitto-config
  volumeClaimTemplates:
  - metadata:
      name: mosquitto-data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi

---
apiVersion: v1
kind: Service
metadata:
  name: mosquitto-service
  namespace: rtk-mqtt
spec:
  selector:
    app: mosquitto
  ports:
  - name: mqtt
    port: 1883
    targetPort: 1883
  - name: websocket
    port: 8083
    targetPort: 8083
  type: LoadBalancer

---
# kubernetes/rtk-controller.yml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rtk-controller
  namespace: rtk-mqtt
spec:
  replicas: 2
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
        image: rtk/controller:latest
        ports:
        - containerPort: 8080
        env:
        - name: RTK_ENV
          value: "production"
        - name: RTK_MQTT_BROKER
          value: "mosquitto-service:1883"
        - name: RTK_LOG_LEVEL
          value: "info"
        volumeMounts:
        - name: controller-data
          mountPath: /app/data
        - name: controller-config
          mountPath: /app/configs
        resources:
          requests:
            memory: "2Gi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: controller-config
        configMap:
          name: rtk-controller-config
      - name: controller-data
        persistentVolumeClaim:
          claimName: controller-data-pvc

---
apiVersion: v1
kind: Service
metadata:
  name: rtk-controller-service
  namespace: rtk-mqtt
spec:
  selector:
    app: rtk-controller
  ports:
  - port: 8080
    targetPort: 8080
  type: LoadBalancer
```

#### 生產環境配置

```yaml
# configs/production.yaml
environment: production

mqtt:
  broker_host: "mosquitto-service"
  broker_port: 1883
  client_id: "rtk_controller_prod"
  username: "${RTK_MQTT_USERNAME}"
  password: "${RTK_MQTT_PASSWORD}"
  tls:
    enabled: true
    ca_cert: "/certs/ca.crt"
    cert_file: "/certs/client.crt"
    key_file: "/certs/client.key"
    insecure_skip_verify: false
  keepalive: 60
  clean_session: false
  
database:
  type: "buntdb"
  path: "/app/data/production"
  backup:
    enabled: true
    interval_hours: 6
    retention_days: 30
  
logging:
  level: "info"
  format: "json"
  output: "file"
  file: "/app/logs/rtk_controller.log"
  rotation:
    max_size_mb: 100
    max_files: 10
  
api:
  enabled: true
  host: "0.0.0.0"
  port: 8080
  auth:
    enabled: true
    jwt_secret: "${RTK_JWT_SECRET}"
  rate_limit:
    enabled: true
    requests_per_minute: 100
  
metrics:
  enabled: true
  path: "/metrics"
  
diagnostics:
  enabled: true
  interval_s: 300
  concurrent_limit: 10
  
qos:
  enabled: true
  analysis_interval_s: 60
  alert_thresholds:
    latency_ms: 100
    packet_loss_percent: 5
    throughput_mbps: 10
```

## 安全配置

### TLS/SSL設置

#### 憑證生成腳本
```bash
#!/bin/bash
# generate_certs.sh

set -e

CERT_DIR="./certs"
CA_KEY="$CERT_DIR/ca.key"
CA_CERT="$CERT_DIR/ca.crt"
SERVER_KEY="$CERT_DIR/server.key"
SERVER_CERT="$CERT_DIR/server.crt"
CLIENT_KEY="$CERT_DIR/client.key"
CLIENT_CERT="$CERT_DIR/client.crt"

mkdir -p $CERT_DIR

echo "Generating CA certificate..."
openssl genrsa -out $CA_KEY 2048
openssl req -new -x509 -days 365 -key $CA_KEY -out $CA_CERT \
  -subj "/C=TW/ST=Taiwan/L=Taipei/O=RTK MQTT/CN=RTK CA"

echo "Generating server certificate..."
openssl genrsa -out $SERVER_KEY 2048
openssl req -new -key $SERVER_KEY -out $CERT_DIR/server.csr \
  -subj "/C=TW/ST=Taiwan/L=Taipei/O=RTK MQTT/CN=rtk-mqtt-broker"
openssl x509 -req -in $CERT_DIR/server.csr -CA $CA_CERT -CAkey $CA_KEY \
  -CAcreateserial -out $SERVER_CERT -days 365

echo "Generating client certificate..."
openssl genrsa -out $CLIENT_KEY 2048
openssl req -new -key $CLIENT_KEY -out $CERT_DIR/client.csr \
  -subj "/C=TW/ST=Taiwan/L=Taipei/O=RTK MQTT/CN=rtk-controller"
openssl x509 -req -in $CERT_DIR/client.csr -CA $CA_CERT -CAkey $CA_KEY \
  -CAcreateserial -out $CLIENT_CERT -days 365

echo "Certificates generated in $CERT_DIR/"
ls -la $CERT_DIR/
```

#### Mosquitto TLS配置
```conf
# mosquitto_tls.conf
port 8883
listener 8884
protocol websockets

# TLS設定
cafile /mosquitto/certs/ca.crt
certfile /mosquitto/certs/server.crt
keyfile /mosquitto/certs/server.key
tls_version tlsv1.2

# 客戶端憑證驗證
require_certificate true
use_identity_as_username true

# 日誌
log_dest file /mosquitto/log/mosquitto.log
log_type all
connection_messages true

# 安全設定
allow_anonymous false
password_file /mosquitto/config/passwd
acl_file /mosquitto/config/acl

# 性能調優
max_connections 1000
max_queued_messages 1000
message_size_limit 1048576
```

### 存取控制清單 (ACL)

```conf
# mosquitto/config/acl
# RTK MQTT Access Control List

# Controller權限 - 可讀取所有設備數據，可發送命令
user rtk_controller
topic readwrite rtk/v1/+/+/+/cmd/req
topic read rtk/v1/+/+/+/state
topic read rtk/v1/+/+/+/telemetry/+
topic read rtk/v1/+/+/+/evt/+
topic read rtk/v1/+/+/+/attr
topic read rtk/v1/+/+/+/cmd/ack
topic read rtk/v1/+/+/+/cmd/res
topic read rtk/v1/+/+/+/lwt
topic read rtk/v1/+/+/+/topology/+

# 設備權限模式 - 設備只能存取自己的主題
pattern read rtk/v1/%c/+/%u/cmd/req
pattern write rtk/v1/%c/+/%u/state
pattern write rtk/v1/%c/+/%u/telemetry/+
pattern write rtk/v1/%c/+/%u/evt/+
pattern write rtk/v1/%c/+/%u/attr
pattern write rtk/v1/%c/+/%u/cmd/ack
pattern write rtk/v1/%c/+/%u/cmd/res
pattern write rtk/v1/%c/+/%u/lwt
pattern write rtk/v1/%c/+/%u/topology/+

# 監控用戶 - 只讀權限
user rtk_monitor
topic read rtk/v1/+/+/+/#

# 管理員 - 全部權限
user rtk_admin
topic readwrite rtk/v1/+/+/+/#
topic readwrite $SYS/#
```

### 使用者管理

```bash
#!/bin/bash
# manage_users.sh

PASSWD_FILE="/mosquitto/config/passwd"

case "$1" in
  add)
    if [ -z "$2" ] || [ -z "$3" ]; then
      echo "Usage: $0 add <username> <password>"
      exit 1
    fi
    mosquitto_passwd -b $PASSWD_FILE "$2" "$3"
    echo "User $2 added"
    ;;
    
  delete)
    if [ -z "$2" ]; then
      echo "Usage: $0 delete <username>"
      exit 1
    fi
    mosquitto_passwd -D $PASSWD_FILE "$2"
    echo "User $2 deleted"
    ;;
    
  list)
    echo "Current users:"
    cut -d: -f1 $PASSWD_FILE
    ;;
    
  *)
    echo "Usage: $0 {add|delete|list}"
    echo "  add <username> <password> - Add new user"
    echo "  delete <username>         - Delete user"
    echo "  list                      - List all users"
    exit 1
    ;;
esac
```

## 監控和告警

### Prometheus監控配置

```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "rtk_mqtt_rules.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  - job_name: 'rtk-controller'
    static_configs:
      - targets: ['rtk-controller:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
    
  - job_name: 'mosquitto'
    static_configs:
      - targets: ['mosquitto:1883']
    metrics_path: '/metrics'
    
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']
```

### 告警規則

```yaml
# monitoring/rtk_mqtt_rules.yml
groups:
  - name: rtk_mqtt_alerts
    rules:
    - alert: RTKControllerDown
      expr: up{job="rtk-controller"} == 0
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "RTK Controller is down"
        description: "RTK Controller has been down for more than 1 minute"
        
    - alert: MQTTBrokerDown
      expr: up{job="mosquitto"} == 0
      for: 30s
      labels:
        severity: critical
      annotations:
        summary: "MQTT Broker is down"
        description: "MQTT Broker has been down for more than 30 seconds"
        
    - alert: HighDeviceLatency
      expr: rtk_device_command_latency_seconds > 5
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "High device command latency"
        description: "Device {{ $labels.device_id }} latency is {{ $value }}s"
        
    - alert: DeviceOffline
      expr: time() - rtk_device_last_seen_timestamp > 300
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Device offline"
        description: "Device {{ $labels.device_id }} has been offline for more than 5 minutes"
        
    - alert: HighConnectionCount
      expr: rtk_mqtt_active_connections > 800
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High MQTT connection count"
        description: "MQTT broker has {{ $value }} active connections"
```

### Grafana儀表板

```json
{
  "dashboard": {
    "title": "RTK MQTT System Dashboard",
    "panels": [
      {
        "title": "System Overview",
        "type": "stat",
        "targets": [
          {
            "expr": "up{job=\"rtk-controller\"}",
            "legendFormat": "Controller Status"
          },
          {
            "expr": "up{job=\"mosquitto\"}",
            "legendFormat": "Broker Status"
          }
        ]
      },
      {
        "title": "Device Count",
        "type": "graph",
        "targets": [
          {
            "expr": "rtk_active_devices_total",
            "legendFormat": "Active Devices"
          },
          {
            "expr": "rtk_total_devices_total",
            "legendFormat": "Total Devices"
          }
        ]
      },
      {
        "title": "Message Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(rtk_messages_received_total[5m])",
            "legendFormat": "Messages/sec"
          }
        ]
      },
      {
        "title": "Command Latency",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(rtk_command_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.50, rate(rtk_command_duration_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ]
      }
    ]
  }
}
```

## 備份和恢復

### 自動備份腳本

```bash
#!/bin/bash
# backup.sh

set -e

BACKUP_DIR="/backup/rtk_mqtt"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# 創建備份目錄
mkdir -p $BACKUP_DIR

echo "Starting RTK MQTT backup at $(date)"

# 備份RTK Controller數據
echo "Backing up RTK Controller data..."
tar -czf "$BACKUP_DIR/rtk_controller_data_$DATE.tar.gz" -C /app/data .

# 備份Mosquitto數據
echo "Backing up Mosquitto data..."
tar -czf "$BACKUP_DIR/mosquitto_data_$DATE.tar.gz" -C /mosquitto/data .

# 備份配置文件
echo "Backing up configurations..."
tar -czf "$BACKUP_DIR/configs_$DATE.tar.gz" \
  /app/configs \
  /mosquitto/config \
  /certs

# 清理舊備份
echo "Cleaning up old backups..."
find $BACKUP_DIR -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup completed at $(date)"
echo "Backup files:"
ls -la $BACKUP_DIR/*$DATE*
```

### 恢復腳本

```bash
#!/bin/bash
# restore.sh

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <backup_date>"
  echo "Available backups:"
  ls /backup/rtk_mqtt/rtk_controller_data_*.tar.gz | sed 's/.*_data_\(.*\)\.tar\.gz/\1/'
  exit 1
fi

BACKUP_DATE="$1"
BACKUP_DIR="/backup/rtk_mqtt"

echo "Restoring RTK MQTT from backup $BACKUP_DATE"

# 停止服務
echo "Stopping services..."
docker-compose down

# 恢復RTK Controller數據
echo "Restoring RTK Controller data..."
mkdir -p /app/data
tar -xzf "$BACKUP_DIR/rtk_controller_data_$BACKUP_DATE.tar.gz" -C /app/data

# 恢復Mosquitto數據
echo "Restoring Mosquitto data..."
mkdir -p /mosquitto/data
tar -xzf "$BACKUP_DIR/mosquitto_data_$BACKUP_DATE.tar.gz" -C /mosquitto/data

# 恢復配置文件
echo "Restoring configurations..."
tar -xzf "$BACKUP_DIR/configs_$BACKUP_DATE.tar.gz" -C /

# 重新啟動服務
echo "Starting services..."
docker-compose up -d

echo "Restore completed"
```

## 性能調優

### MQTT Broker調優

```conf
# mosquitto_optimized.conf
# 連接設定
max_connections 2000
max_queued_messages 1000

# 記憶體設定
memory_limit 512MB
message_size_limit 1048576

# 持久化設定
persistence true
persistence_location /mosquitto/data/
autosave_interval 300
autosave_on_changes false

# 網路設定
keepalive_interval 60
max_keepalive 120

# 日誌優化
log_dest file /mosquitto/log/mosquitto.log
log_type error
log_type warning
connection_messages false
```

### RTK Controller調優

```yaml
# configs/optimized.yaml
performance:
  max_concurrent_commands: 100
  command_timeout_seconds: 30
  batch_size: 50
  
database:
  cache_size_mb: 256
  write_buffer_size_mb: 64
  
mqtt:
  connection_pool_size: 10
  max_reconnect_interval: 60
  message_buffer_size: 10000
  
diagnostics:
  worker_pool_size: 20
  queue_size: 1000
```

### 系統層級調優

```bash
#!/bin/bash
# system_tune.sh

echo "Applying RTK MQTT system optimizations..."

# 網路參數調優
echo "Tuning network parameters..."
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_rmem="4096 65536 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"
sysctl -w net.ipv4.tcp_window_scaling=1
sysctl -w net.core.netdev_max_backlog=5000

# 檔案描述符限制
echo "Setting file descriptor limits..."
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# Docker容器資源限制
cat > docker-compose.override.yml << EOF
version: '3.8'
services:
  mosquitto:
    deploy:
      resources:
        limits:
          memory: 2G
          cpus: '1.0'
        reservations:
          memory: 1G
          cpus: '0.5'
    sysctls:
      net.core.somaxconn: 1024
      
  rtk-controller:
    deploy:
      resources:
        limits:
          memory: 4G
          cpus: '2.0'
        reservations:
          memory: 2G
          cpus: '1.0'
EOF

echo "System tuning completed"
```

## 維護和運維

### 健康檢查

```bash
#!/bin/bash
# health_check.sh

echo "RTK MQTT System Health Check"
echo "=============================="

# 檢查服務狀態
echo "1. Service Status:"
docker-compose ps

# 檢查MQTT連接
echo -e "\n2. MQTT Connectivity:"
timeout 5 mosquitto_pub -h localhost -t test -m "health_check" && echo "✓ MQTT OK" || echo "✗ MQTT Failed"

# 檢查Controller API
echo -e "\n3. Controller API:"
curl -sf http://localhost:8080/health && echo "✓ API OK" || echo "✗ API Failed"

# 檢查磁碟空間
echo -e "\n4. Disk Usage:"
df -h | grep -E "(/$|/app|/mosquitto)"

# 檢查記憶體使用
echo -e "\n5. Memory Usage:"
free -h

# 檢查活躍設備數量
echo -e "\n6. Active Devices:"
mosquitto_sub -h localhost -t 'rtk/v1/+/+/+/state' -C 10 --quiet | wc -l | xargs echo "Active devices:"

# 檢查錯誤日誌
echo -e "\n7. Recent Errors:"
docker logs rtk-controller-prod 2>&1 | grep -i error | tail -5
```

### 日誌管理

```bash
#!/bin/bash
# log_management.sh

LOG_DIR="/app/logs"
RETENTION_DAYS=7

case "$1" in
  rotate)
    echo "Rotating logs..."
    logrotate /etc/logrotate.d/rtk_mqtt
    ;;
    
  cleanup)
    echo "Cleaning up old logs..."
    find $LOG_DIR -name "*.log.*" -mtime +$RETENTION_DAYS -delete
    find /mosquitto/log -name "*.log.*" -mtime +$RETENTION_DAYS -delete
    ;;
    
  analyze)
    echo "Analyzing error patterns..."
    grep -i error $LOG_DIR/rtk_controller.log | \
      awk '{print $1, $2, $NF}' | sort | uniq -c | sort -nr | head -10
    ;;
    
  tail)
    echo "Following live logs..."
    tail -f $LOG_DIR/rtk_controller.log
    ;;
    
  *)
    echo "Usage: $0 {rotate|cleanup|analyze|tail}"
    ;;
esac
```

### 更新腳本

```bash
#!/bin/bash
# update.sh

set -e

NEW_VERSION="$1"
if [ -z "$NEW_VERSION" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

echo "Updating RTK MQTT to version $NEW_VERSION"

# 創建備份
echo "Creating backup..."
./backup.sh

# 拉取新映像
echo "Pulling new images..."
docker pull rtk/controller:$NEW_VERSION
docker pull eclipse-mosquitto:2.0

# 更新docker-compose
sed -i "s|rtk/controller:.*|rtk/controller:$NEW_VERSION|" docker-compose.yml

# 滾動更新
echo "Performing rolling update..."
docker-compose up -d --no-deps rtk-controller

# 等待服務就緒
echo "Waiting for service to be ready..."
sleep 30

# 健康檢查
./health_check.sh

echo "Update completed successfully!"
```

## 故障排除

### 常見問題診斷

```bash
#!/bin/bash
# troubleshoot.sh

echo "RTK MQTT Troubleshooting Guide"
echo "==============================="

echo "1. Checking common issues..."

# MQTT連接問題
if ! mosquitto_pub -h localhost -t test -m test 2>/dev/null; then
  echo "✗ MQTT connection failed"
  echo "  - Check if Mosquitto is running: docker ps | grep mosquitto"
  echo "  - Check Mosquitto logs: docker logs mosquitto"
  echo "  - Verify port 1883 is open: netstat -ln | grep 1883"
else
  echo "✓ MQTT connection OK"
fi

# Controller API問題
if ! curl -sf http://localhost:8080/health >/dev/null; then
  echo "✗ Controller API failed"
  echo "  - Check if Controller is running: docker ps | grep controller"
  echo "  - Check Controller logs: docker logs rtk-controller"
  echo "  - Verify port 8080 is open: netstat -ln | grep 8080"
else
  echo "✓ Controller API OK"
fi

# 磁碟空間檢查
DISK_USAGE=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
if [ $DISK_USAGE -gt 90 ]; then
  echo "✗ Disk usage critical: ${DISK_USAGE}%"
  echo "  - Clean up logs: find /var/log -type f -mtime +7 -delete"
  echo "  - Clean up Docker: docker system prune -f"
else
  echo "✓ Disk usage OK: ${DISK_USAGE}%"
fi

# 記憶體檢查
MEMORY_USAGE=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
if [ $MEMORY_USAGE -gt 90 ]; then
  echo "✗ Memory usage critical: ${MEMORY_USAGE}%"
  echo "  - Check top processes: docker stats"
  echo "  - Consider increasing memory limits"
else
  echo "✓ Memory usage OK: ${MEMORY_USAGE}%"
fi

echo -e "\n2. Service logs (last 10 lines):"
echo "RTK Controller:"
docker logs rtk-controller --tail 10

echo -e "\nMosquitto:"
docker logs mosquitto --tail 10
```

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Quick Start Guide](QUICK_START_GUIDE.md)
- [Testing Integration Guide](TESTING_INTEGRATION.md)
- [Troubleshooting Guide](TROUBLESHOOTING_GUIDE.md)
- [Docker Documentation](https://docs.docker.com/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)