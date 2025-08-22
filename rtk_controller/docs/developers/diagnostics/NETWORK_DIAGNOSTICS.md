# 網路診斷工具指南

## 概述

本文檔提供RTK MQTT系統中網路診斷功能的完整說明，包含診斷工具的使用方法、診斷流程和結果解釋。

## 支援的診斷測試

### 1. 速度測試 (Speed Test)

#### 功能說明
測量設備的網路上傳和下載速度，評估網路頻寬性能。

#### 命令格式
```json
{
  "id": "cmd-1699123456789",
  "op": "speed_test",
  "schema": "cmd.speed_test/1.0",
  "args": {
    "server": "auto",
    "duration": 10,
    "direction": "both",
    "test_size": "10MB"
  }
}
```

#### 參數說明
- `server`: 測試伺服器選擇
  - `"auto"`: 自動選擇最佳伺服器
  - `"ookla"`: 使用Ookla Speedtest伺服器
  - `"custom_url"`: 自定義測試URL
- `duration`: 測試持續時間(秒)，範圍: 5-60
- `direction`: 測試方向
  - `"download"`: 僅下載測試
  - `"upload"`: 僅上傳測試
  - `"both"`: 雙向測試
- `test_size`: 測試檔案大小，可選: "1MB", "10MB", "100MB"

#### 結果格式
```json
{
  "id": "cmd-1699123456789",
  "schema": "cmd.speed_test.result/1.0",
  "status": "completed",
  "result": {
    "download_mbps": 95.42,
    "upload_mbps": 48.73,
    "latency_ms": 15.6,
    "jitter_ms": 2.3,
    "server_info": {
      "name": "Speedtest Server",
      "location": "Taipei, TW",
      "distance_km": 5.2
    },
    "test_duration_s": 10.5,
    "timestamp": 1699123456789
  }
}
```

#### 實作範例
```python
import subprocess
import json
import time

def perform_speed_test(args):
    """執行網路速度測試"""
    server = args.get("server", "auto")
    duration = args.get("duration", 10)
    direction = args.get("direction", "both")
    
    try:
        if server == "ookla":
            # 使用Ookla Speedtest CLI
            cmd = ["speedtest", "--format=json"]
            if duration:
                cmd.extend(["--progress=no"])
                
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
            
            if result.returncode == 0:
                data = json.loads(result.stdout)
                return {
                    "download_mbps": data["download"]["bandwidth"] * 8 / 1000000,
                    "upload_mbps": data["upload"]["bandwidth"] * 8 / 1000000,
                    "latency_ms": data["ping"]["latency"],
                    "jitter_ms": data["ping"]["jitter"],
                    "server_info": {
                        "name": data["server"]["name"],
                        "location": f"{data['server']['location']}, {data['server']['country']}",
                        "distance_km": data["server"]["distance"]
                    }
                }
        else:
            # 使用自定義測試邏輯
            return perform_custom_speed_test(args)
            
    except subprocess.TimeoutExpired:
        return {"error": "Speed test timeout"}
    except Exception as e:
        return {"error": f"Speed test failed: {str(e)}"}

def perform_custom_speed_test(args):
    """自定義速度測試實作"""
    import urllib.request
    import time
    
    # 測試URL
    test_urls = {
        "1MB": "http://speedtest.ftp.otenet.gr/files/test1Mb.db",
        "10MB": "http://speedtest.ftp.otenet.gr/files/test10Mb.db",
        "100MB": "http://speedtest.ftp.otenet.gr/files/test100Mb.db"
    }
    
    test_size = args.get("test_size", "10MB")
    test_url = test_urls.get(test_size)
    
    if not test_url:
        return {"error": "Invalid test size"}
    
    # 下載測試
    start_time = time.time()
    try:
        with urllib.request.urlopen(test_url) as response:
            data = response.read()
            download_time = time.time() - start_time
            
        # 計算速度
        file_size_mb = len(data) / (1024 * 1024)
        download_mbps = (file_size_mb * 8) / download_time
        
        return {
            "download_mbps": round(download_mbps, 2),
            "upload_mbps": 0,  # 簡化實作，僅測試下載
            "latency_ms": 0,
            "test_duration_s": download_time,
            "file_size_mb": file_size_mb
        }
        
    except Exception as e:
        return {"error": f"Download test failed: {str(e)}"}
```

### 2. WAN連線診斷 (WAN Connectivity)

#### 功能說明
測試設備對外網路的連通性，包含DNS解析和閘道連線。

#### 命令格式
```json
{
  "id": "cmd-1699123456790",
  "op": "wan_connectivity",
  "schema": "cmd.wan_connectivity/1.0",
  "args": {
    "test_hosts": ["8.8.8.8", "google.com", "cloudflare.com"],
    "timeout": 5,
    "ping_count": 3,
    "trace_route": false
  }
}
```

#### 參數說明
- `test_hosts`: 測試目標主機列表
- `timeout`: 每個測試的超時時間(秒)
- `ping_count`: ping測試次數
- `trace_route`: 是否執行路由追蹤

#### 結果格式
```json
{
  "result": {
    "overall_status": "connected",
    "gateway_reachable": true,
    "dns_resolution": true,
    "internet_connectivity": true,
    "test_results": [
      {
        "host": "8.8.8.8",
        "ip_address": "8.8.8.8",
        "reachable": true,
        "avg_latency_ms": 12.5,
        "packet_loss_percent": 0
      },
      {
        "host": "google.com",
        "ip_address": "142.250.191.14",
        "reachable": true,
        "avg_latency_ms": 15.2,
        "packet_loss_percent": 0,
        "dns_resolution_time_ms": 45.2
      }
    ],
    "gateway_info": {
      "ip_address": "192.168.1.1",
      "latency_ms": 2.1,
      "reachable": true
    }
  }
}
```

#### 實作範例
```python
import subprocess
import socket
import time

def wan_connectivity_test(args):
    """執行WAN連線診斷"""
    test_hosts = args.get("test_hosts", ["8.8.8.8", "google.com"])
    timeout = args.get("timeout", 5)
    ping_count = args.get("ping_count", 3)
    
    results = {
        "overall_status": "unknown",
        "gateway_reachable": False,
        "dns_resolution": False,
        "internet_connectivity": False,
        "test_results": [],
        "gateway_info": {}
    }
    
    # 測試閘道連通性
    gateway_ip = get_default_gateway()
    if gateway_ip:
        gateway_result = ping_host(gateway_ip, ping_count, timeout)
        results["gateway_info"] = {
            "ip_address": gateway_ip,
            "latency_ms": gateway_result.get("avg_latency_ms", 0),
            "reachable": gateway_result.get("reachable", False)
        }
        results["gateway_reachable"] = gateway_result.get("reachable", False)
    
    # 測試各個目標主機
    dns_working = False
    internet_working = False
    
    for host in test_hosts:
        host_result = test_host_connectivity(host, ping_count, timeout)
        results["test_results"].append(host_result)
        
        if host_result.get("reachable"):
            internet_working = True
            
        if "dns_resolution_time_ms" in host_result:
            dns_working = True
    
    results["dns_resolution"] = dns_working
    results["internet_connectivity"] = internet_working
    
    # 判斷整體狀態
    if results["gateway_reachable"] and internet_working:
        results["overall_status"] = "connected"
    elif results["gateway_reachable"]:
        results["overall_status"] = "gateway_only"
    else:
        results["overall_status"] = "disconnected"
    
    return results

def get_default_gateway():
    """獲取預設閘道IP"""
    try:
        result = subprocess.run(
            ["ip", "route", "show", "default"],
            capture_output=True, text=True, timeout=5
        )
        
        for line in result.stdout.split('\n'):
            if "default via" in line:
                return line.split()[2]
                
    except:
        pass
    
    return None

def test_host_connectivity(host, ping_count, timeout):
    """測試單一主機連通性"""
    result = {
        "host": host,
        "reachable": False,
        "avg_latency_ms": 0,
        "packet_loss_percent": 100
    }
    
    # DNS解析測試
    if not is_ip_address(host):
        dns_start = time.time()
        try:
            ip_address = socket.gethostbyname(host)
            dns_time = (time.time() - dns_start) * 1000
            result["ip_address"] = ip_address
            result["dns_resolution_time_ms"] = round(dns_time, 2)
        except socket.gaierror:
            result["error"] = "DNS resolution failed"
            return result
    else:
        result["ip_address"] = host
    
    # Ping測試
    ping_result = ping_host(result["ip_address"], ping_count, timeout)
    result.update(ping_result)
    
    return result

def ping_host(host, count, timeout):
    """執行ping測試"""
    try:
        cmd = ["ping", "-c", str(count), "-W", str(timeout), host]
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout + 5)
        
        if result.returncode == 0:
            # 解析ping結果
            output = result.stdout
            
            # 提取平均延遲
            for line in output.split('\n'):
                if "rtt min/avg/max/mdev" in line:
                    avg_latency = float(line.split('/')[5])
                    return {
                        "reachable": True,
                        "avg_latency_ms": round(avg_latency, 2),
                        "packet_loss_percent": 0
                    }
            
            return {"reachable": True, "avg_latency_ms": 0, "packet_loss_percent": 0}
        else:
            return {"reachable": False, "packet_loss_percent": 100}
            
    except subprocess.TimeoutExpired:
        return {"reachable": False, "packet_loss_percent": 100, "error": "Timeout"}
    except Exception as e:
        return {"reachable": False, "packet_loss_percent": 100, "error": str(e)}

def is_ip_address(host):
    """檢查是否為IP地址"""
    try:
        socket.inet_aton(host)
        return True
    except socket.error:
        return False
```

### 3. 延遲測試 (Latency Test)

#### 功能說明
測量網路延遲、抖動和封包遺失率。

#### 命令格式
```json
{
  "id": "cmd-1699123456791",
  "op": "latency_test",
  "schema": "cmd.latency_test/1.0",
  "args": {
    "target_hosts": ["8.8.8.8", "1.1.1.1"],
    "packet_count": 20,
    "packet_size": 64,
    "interval_ms": 1000
  }
}
```

#### 結果格式
```json
{
  "result": {
    "test_summary": {
      "total_hosts": 2,
      "avg_latency_ms": 14.5,
      "avg_jitter_ms": 2.1,
      "overall_packet_loss": 0.5
    },
    "host_results": [
      {
        "host": "8.8.8.8",
        "packets_sent": 20,
        "packets_received": 20,
        "packet_loss_percent": 0,
        "min_latency_ms": 10.2,
        "max_latency_ms": 18.9,
        "avg_latency_ms": 13.5,
        "std_dev_ms": 2.1,
        "jitter_ms": 1.8
      }
    ]
  }
}
```

### 4. DNS解析測試 (DNS Resolution)

#### 功能說明
測試DNS伺服器的解析能力和響應時間。

#### 命令格式
```json
{
  "id": "cmd-1699123456792",
  "op": "dns_resolution",
  "schema": "cmd.dns_resolution/1.0",
  "args": {
    "test_domains": ["google.com", "github.com", "stackoverflow.com"],
    "dns_servers": ["8.8.8.8", "1.1.1.1", "system"],
    "query_types": ["A", "AAAA", "MX"]
  }
}
```

#### 結果格式
```json
{
  "result": {
    "dns_servers_tested": 3,
    "domains_tested": 3,
    "overall_success_rate": 95.5,
    "avg_resolution_time_ms": 42.3,
    "results": [
      {
        "dns_server": "8.8.8.8",
        "domain": "google.com",
        "query_type": "A",
        "success": true,
        "resolution_time_ms": 38.2,
        "resolved_ips": ["142.250.191.14"],
        "ttl": 300
      }
    ]
  }
}
```

## 診斷工具整合

### RTK Controller整合

```go
// internal/diagnostics/network_diagnostics.go
package diagnostics

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
)

type NetworkDiagnostics struct {
    logger Logger
}

func NewNetworkDiagnostics(logger Logger) *NetworkDiagnostics {
    return &NetworkDiagnostics{
        logger: logger,
    }
}

func (nd *NetworkDiagnostics) RunSpeedTest(ctx context.Context, deviceID string, args map[string]interface{}) (*DiagnosticResult, error) {
    nd.logger.Info("Starting speed test", "device", deviceID)
    
    // 發送命令到設備
    command := &Command{
        ID:     generateCommandID(),
        Op:     "speed_test",
        Schema: "cmd.speed_test/1.0",
        Args:   args,
    }
    
    result, err := nd.sendCommandToDevice(ctx, deviceID, command)
    if err != nil {
        return nil, fmt.Errorf("speed test failed: %w", err)
    }
    
    return &DiagnosticResult{
        TestType:  "speed_test",
        DeviceID:  deviceID,
        Status:    "completed",
        Result:    result,
        Timestamp: time.Now(),
    }, nil
}

func (nd *NetworkDiagnostics) RunWANConnectivityTest(ctx context.Context, deviceID string, args map[string]interface{}) (*DiagnosticResult, error) {
    nd.logger.Info("Starting WAN connectivity test", "device", deviceID)
    
    command := &Command{
        ID:     generateCommandID(),
        Op:     "wan_connectivity",
        Schema: "cmd.wan_connectivity/1.0",
        Args:   args,
    }
    
    result, err := nd.sendCommandToDevice(ctx, deviceID, command)
    if err != nil {
        return nil, fmt.Errorf("WAN connectivity test failed: %w", err)
    }
    
    return &DiagnosticResult{
        TestType:  "wan_connectivity",
        DeviceID:  deviceID,
        Status:    "completed",
        Result:    result,
        Timestamp: time.Now(),
    }, nil
}

func (nd *NetworkDiagnostics) sendCommandToDevice(ctx context.Context, deviceID string, command *Command) (map[string]interface{}, error) {
    // 實作MQTT命令發送邏輯
    // 等待設備響應
    // 解析結果
    
    // 模擬實作
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    case <-time.After(30 * time.Second):
        return nil, fmt.Errorf("command timeout")
    }
}

type DiagnosticResult struct {
    TestType  string                 `json:"test_type"`
    DeviceID  string                 `json:"device_id"`
    Status    string                 `json:"status"`
    Result    map[string]interface{} `json:"result"`
    Timestamp time.Time              `json:"timestamp"`
    Error     string                 `json:"error,omitempty"`
}
```

### CLI命令介面

```go
// internal/cli/diagnostic_commands.go
package cli

import (
    "encoding/json"
    "fmt"
    "github.com/spf13/cobra"
)

func (h *Handler) buildDiagnosticCommands() *cobra.Command {
    diagnosticCmd := &cobra.Command{
        Use:   "diagnostic",
        Short: "Network diagnostic commands",
        Long:  "Run various network diagnostic tests on devices",
    }
    
    // 速度測試命令
    speedTestCmd := &cobra.Command{
        Use:   "speed_test [device_id]",
        Short: "Run network speed test",
        Args:  cobra.ExactArgs(1),
        RunE:  h.handleSpeedTest,
    }
    
    speedTestCmd.Flags().String("server", "auto", "Test server selection")
    speedTestCmd.Flags().Int("duration", 10, "Test duration in seconds")
    speedTestCmd.Flags().String("direction", "both", "Test direction: download, upload, both")
    
    // WAN連線測試命令
    wanTestCmd := &cobra.Command{
        Use:   "wan_test [device_id]",
        Short: "Run WAN connectivity test",
        Args:  cobra.ExactArgs(1),
        RunE:  h.handleWANTest,
    }
    
    wanTestCmd.Flags().StringSlice("hosts", []string{"8.8.8.8", "google.com"}, "Test hosts")
    wanTestCmd.Flags().Int("timeout", 5, "Timeout in seconds")
    wanTestCmd.Flags().Int("ping_count", 3, "Number of ping packets")
    
    // 延遲測試命令
    latencyTestCmd := &cobra.Command{
        Use:   "latency_test [device_id]",
        Short: "Run latency and jitter test",
        Args:  cobra.ExactArgs(1),
        RunE:  h.handleLatencyTest,
    }
    
    latencyTestCmd.Flags().StringSlice("targets", []string{"8.8.8.8"}, "Target hosts")
    latencyTestCmd.Flags().Int("count", 20, "Number of packets")
    latencyTestCmd.Flags().Int("size", 64, "Packet size in bytes")
    
    diagnosticCmd.AddCommand(speedTestCmd, wanTestCmd, latencyTestCmd)
    
    return diagnosticCmd
}

func (h *Handler) handleSpeedTest(cmd *cobra.Command, args []string) error {
    deviceID := args[0]
    
    server, _ := cmd.Flags().GetString("server")
    duration, _ := cmd.Flags().GetInt("duration")
    direction, _ := cmd.Flags().GetString("direction")
    
    testArgs := map[string]interface{}{
        "server":    server,
        "duration":  duration,
        "direction": direction,
    }
    
    fmt.Printf("Running speed test on device %s...\n", deviceID)
    
    result, err := h.diagnostics.RunSpeedTest(cmd.Context(), deviceID, testArgs)
    if err != nil {
        return fmt.Errorf("speed test failed: %w", err)
    }
    
    // 格式化輸出結果
    return h.displaySpeedTestResult(result)
}

func (h *Handler) displaySpeedTestResult(result *DiagnosticResult) error {
    fmt.Printf("Speed Test Results for Device %s\n", result.DeviceID)
    fmt.Printf("================================\n")
    fmt.Printf("Status: %s\n", result.Status)
    fmt.Printf("Timestamp: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
    
    if result.Status == "completed" {
        resultData := result.Result
        
        if download, ok := resultData["download_mbps"].(float64); ok {
            fmt.Printf("Download Speed: %.2f Mbps\n", download)
        }
        
        if upload, ok := resultData["upload_mbps"].(float64); ok {
            fmt.Printf("Upload Speed: %.2f Mbps\n", upload)
        }
        
        if latency, ok := resultData["latency_ms"].(float64); ok {
            fmt.Printf("Latency: %.2f ms\n", latency)
        }
        
        if jitter, ok := resultData["jitter_ms"].(float64); ok {
            fmt.Printf("Jitter: %.2f ms\n", jitter)
        }
        
        // 顯示伺服器資訊
        if serverInfo, ok := resultData["server_info"].(map[string]interface{}); ok {
            fmt.Printf("\nServer Information:\n")
            if name, ok := serverInfo["name"].(string); ok {
                fmt.Printf("  Name: %s\n", name)
            }
            if location, ok := serverInfo["location"].(string); ok {
                fmt.Printf("  Location: %s\n", location)
            }
            if distance, ok := serverInfo["distance_km"].(float64); ok {
                fmt.Printf("  Distance: %.1f km\n", distance)
            }
        }
    } else {
        fmt.Printf("Error: %s\n", result.Error)
    }
    
    return nil
}
```

## 診斷結果分析

### 性能評級標準

```go
// internal/diagnostics/analysis.go
package diagnostics

type PerformanceGrade string

const (
    GradeExcellent PerformanceGrade = "excellent"
    GradeGood      PerformanceGrade = "good"
    GradeFair      PerformanceGrade = "fair"
    GradePoor      PerformanceGrade = "poor"
)

type NetworkAnalyzer struct {
    thresholds map[string]PerformanceThresholds
}

type PerformanceThresholds struct {
    Excellent struct {
        DownloadMbps float64 `json:"download_mbps"`
        UploadMbps   float64 `json:"upload_mbps"`
        LatencyMs    float64 `json:"latency_ms"`
        JitterMs     float64 `json:"jitter_ms"`
        PacketLoss   float64 `json:"packet_loss"`
    } `json:"excellent"`
    Good struct {
        DownloadMbps float64 `json:"download_mbps"`
        UploadMbps   float64 `json:"upload_mbps"`
        LatencyMs    float64 `json:"latency_ms"`
        JitterMs     float64 `json:"jitter_ms"`
        PacketLoss   float64 `json:"packet_loss"`
    } `json:"good"`
    Fair struct {
        DownloadMbps float64 `json:"download_mbps"`
        UploadMbps   float64 `json:"upload_mbps"`
        LatencyMs    float64 `json:"latency_ms"`
        JitterMs     float64 `json:"jitter_ms"`
        PacketLoss   float64 `json:"packet_loss"`
    } `json:"fair"`
}

func NewNetworkAnalyzer() *NetworkAnalyzer {
    thresholds := map[string]PerformanceThresholds{
        "speed_test": {
            Excellent: struct {
                DownloadMbps float64 `json:"download_mbps"`
                UploadMbps   float64 `json:"upload_mbps"`
                LatencyMs    float64 `json:"latency_ms"`
                JitterMs     float64 `json:"jitter_ms"`
                PacketLoss   float64 `json:"packet_loss"`
            }{
                DownloadMbps: 100.0,
                UploadMbps:   50.0,
                LatencyMs:    20.0,
                JitterMs:     2.0,
                PacketLoss:   0.1,
            },
            Good: struct {
                DownloadMbps float64 `json:"download_mbps"`
                UploadMbps   float64 `json:"upload_mbps"`
                LatencyMs    float64 `json:"latency_ms"`
                JitterMs     float64 `json:"jitter_ms"`
                PacketLoss   float64 `json:"packet_loss"`
            }{
                DownloadMbps: 50.0,
                UploadMbps:   25.0,
                LatencyMs:    50.0,
                JitterMs:     5.0,
                PacketLoss:   1.0,
            },
            Fair: struct {
                DownloadMbps float64 `json:"download_mbps"`
                UploadMbps   float64 `json:"upload_mbps"`
                LatencyMs    float64 `json:"latency_ms"`
                JitterMs     float64 `json:"jitter_ms"`
                PacketLoss   float64 `json:"packet_loss"`
            }{
                DownloadMbps: 10.0,
                UploadMbps:   5.0,
                LatencyMs:    100.0,
                JitterMs:     10.0,
                PacketLoss:   5.0,
            },
        },
    }
    
    return &NetworkAnalyzer{
        thresholds: thresholds,
    }
}

func (na *NetworkAnalyzer) AnalyzeSpeedTest(result map[string]interface{}) *AnalysisResult {
    analysis := &AnalysisResult{
        TestType: "speed_test",
        Overall:  GradePoor,
        Metrics:  make(map[string]MetricAnalysis),
    }
    
    thresholds := na.thresholds["speed_test"]
    
    // 分析下載速度
    if download, ok := result["download_mbps"].(float64); ok {
        analysis.Metrics["download"] = na.gradeMetric(download, 
            thresholds.Excellent.DownloadMbps,
            thresholds.Good.DownloadMbps,
            thresholds.Fair.DownloadMbps,
            true) // true表示數值越大越好
    }
    
    // 分析上傳速度
    if upload, ok := result["upload_mbps"].(float64); ok {
        analysis.Metrics["upload"] = na.gradeMetric(upload,
            thresholds.Excellent.UploadMbps,
            thresholds.Good.UploadMbps,
            thresholds.Fair.UploadMbps,
            true)
    }
    
    // 分析延遲
    if latency, ok := result["latency_ms"].(float64); ok {
        analysis.Metrics["latency"] = na.gradeMetric(latency,
            thresholds.Excellent.LatencyMs,
            thresholds.Good.LatencyMs,
            thresholds.Fair.LatencyMs,
            false) // false表示數值越小越好
    }
    
    // 計算綜合評級
    analysis.Overall = na.calculateOverallGrade(analysis.Metrics)
    
    // 生成建議
    analysis.Recommendations = na.generateRecommendations(analysis)
    
    return analysis
}

type AnalysisResult struct {
    TestType        string                    `json:"test_type"`
    Overall         PerformanceGrade         `json:"overall_grade"`
    Metrics         map[string]MetricAnalysis `json:"metrics"`
    Recommendations []string                  `json:"recommendations"`
}

type MetricAnalysis struct {
    Value float64          `json:"value"`
    Grade PerformanceGrade `json:"grade"`
    Unit  string          `json:"unit"`
}

func (na *NetworkAnalyzer) gradeMetric(value, excellentThreshold, goodThreshold, fairThreshold float64, higherIsBetter bool) MetricAnalysis {
    var grade PerformanceGrade
    
    if higherIsBetter {
        if value >= excellentThreshold {
            grade = GradeExcellent
        } else if value >= goodThreshold {
            grade = GradeGood
        } else if value >= fairThreshold {
            grade = GradeFair
        } else {
            grade = GradePoor
        }
    } else {
        if value <= excellentThreshold {
            grade = GradeExcellent
        } else if value <= goodThreshold {
            grade = GradeGood
        } else if value <= fairThreshold {
            grade = GradeFair
        } else {
            grade = GradePoor
        }
    }
    
    return MetricAnalysis{
        Value: value,
        Grade: grade,
    }
}

func (na *NetworkAnalyzer) generateRecommendations(analysis *AnalysisResult) []string {
    var recommendations []string
    
    for metric, analysis := range analysis.Metrics {
        switch analysis.Grade {
        case GradePoor:
            switch metric {
            case "download":
                recommendations = append(recommendations, "下載速度較低，建議檢查網路設備或聯絡ISP")
            case "upload":
                recommendations = append(recommendations, "上傳速度較低，建議檢查上傳頻寬限制")
            case "latency":
                recommendations = append(recommendations, "延遲過高，建議檢查網路路由或DNS設定")
            }
        case GradeFair:
            switch metric {
            case "download":
                recommendations = append(recommendations, "下載速度尚可，但可能影響大檔案傳輸")
            case "latency":
                recommendations = append(recommendations, "延遲略高，可能影響即時應用效能")
            }
        }
    }
    
    if len(recommendations) == 0 {
        recommendations = append(recommendations, "網路效能良好，無需特別優化")
    }
    
    return recommendations
}
```

## 自動化診斷

### 定期診斷任務

```go
// internal/diagnostics/scheduler.go
package diagnostics

import (
    "context"
    "time"
    "sync"
)

type DiagnosticScheduler struct {
    diagnostics *NetworkDiagnostics
    analyzer    *NetworkAnalyzer
    ticker      *time.Ticker
    devices     []string
    mu          sync.RWMutex
    running     bool
}

func NewDiagnosticScheduler(diagnostics *NetworkDiagnostics, analyzer *NetworkAnalyzer) *DiagnosticScheduler {
    return &DiagnosticScheduler{
        diagnostics: diagnostics,
        analyzer:    analyzer,
        devices:     make([]string, 0),
    }
}

func (ds *DiagnosticScheduler) Start(interval time.Duration) {
    ds.mu.Lock()
    defer ds.mu.Unlock()
    
    if ds.running {
        return
    }
    
    ds.ticker = time.NewTicker(interval)
    ds.running = true
    
    go ds.runScheduledDiagnostics()
}

func (ds *DiagnosticScheduler) Stop() {
    ds.mu.Lock()
    defer ds.mu.Unlock()
    
    if !ds.running {
        return
    }
    
    ds.ticker.Stop()
    ds.running = false
}

func (ds *DiagnosticScheduler) AddDevice(deviceID string) {
    ds.mu.Lock()
    defer ds.mu.Unlock()
    
    ds.devices = append(ds.devices, deviceID)
}

func (ds *DiagnosticScheduler) runScheduledDiagnostics() {
    for range ds.ticker.C {
        ds.runDiagnosticsRound()
    }
}

func (ds *DiagnosticScheduler) runDiagnosticsRound() {
    ds.mu.RLock()
    devices := make([]string, len(ds.devices))
    copy(devices, ds.devices)
    ds.mu.RUnlock()
    
    for _, deviceID := range devices {
        go ds.runDeviceDiagnostics(deviceID)
    }
}

func (ds *DiagnosticScheduler) runDeviceDiagnostics(deviceID string) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    // 執行基本網路診斷
    speedArgs := map[string]interface{}{
        "server":    "auto",
        "duration":  10,
        "direction": "both",
    }
    
    result, err := ds.diagnostics.RunSpeedTest(ctx, deviceID, speedArgs)
    if err != nil {
        // 記錄錯誤
        return
    }
    
    // 分析結果
    analysis := ds.analyzer.AnalyzeSpeedTest(result.Result)
    
    // 如果效能不佳，觸發告警
    if analysis.Overall == GradePoor {
        ds.triggerAlert(deviceID, analysis)
    }
    
    // 儲存診斷結果
    ds.storeDiagnosticResult(deviceID, result, analysis)
}

func (ds *DiagnosticScheduler) triggerAlert(deviceID string, analysis *AnalysisResult) {
    // 實作告警邏輯
    // 可以發送MQTT事件、寫入日誌、調用webhook等
}

func (ds *DiagnosticScheduler) storeDiagnosticResult(deviceID string, result *DiagnosticResult, analysis *AnalysisResult) {
    // 實作結果儲存邏輯
    // 可以儲存到資料庫、檔案或其他儲存系統
}
```

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Commands and Events Reference](../core/COMMANDS_EVENTS_REFERENCE.md)
- [Quick Start Guide](../guides/QUICK_START_GUIDE.md)
- [Troubleshooting Guide](../guides/TROUBLESHOOTING_GUIDE.md)
- [Speedtest CLI Documentation](https://www.speedtest.net/apps/cli)
- [Network Performance Testing Tools](https://github.com/sivel/speedtest-cli)