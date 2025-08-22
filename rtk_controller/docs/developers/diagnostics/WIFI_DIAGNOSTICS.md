# WiFi 診斷工具指南

## 概述

本文檔提供RTK MQTT系統中WiFi診斷功能的完整說明，包含WiFi網路分析、信號品質評估、干擾檢測和覆蓋範圍分析等工具。

## 支援的WiFi診斷測試

### 1. WiFi頻道掃描 (WiFi Channel Scan)

#### 功能說明
掃描周邊的WiFi網路，分析頻道使用情況和信號強度分佈。

#### 命令格式
```json
{
  "id": "cmd-1699123456789",
  "op": "wifi_scan_channels",
  "schema": "cmd.wifi_scan_channels/1.0",
  "args": {
    "scan_type": "active",
    "channels": [1, 6, 11, 36, 40, 44, 48],
    "duration_per_channel": 500,
    "include_hidden": true,
    "band": "both"
  }
}
```

#### 參數說明
- `scan_type`: 掃描類型
  - `"active"`: 主動掃描 (發送Probe Request)
  - `"passive"`: 被動掃描 (僅監聽Beacon)
- `channels`: 指定掃描的頻道列表 (空陣列表示掃描所有頻道)
- `duration_per_channel`: 每個頻道掃描時間(毫秒)
- `include_hidden`: 是否包含隱藏SSID
- `band`: 頻段選擇
  - `"2.4ghz"`: 僅2.4GHz
  - `"5ghz"`: 僅5GHz
  - `"both"`: 雙頻段

#### 結果格式
```json
{
  "result": {
    "scan_summary": {
      "total_networks": 25,
      "channels_scanned": 13,
      "scan_duration_ms": 6500,
      "strongest_signal": -35,
      "weakest_signal": -89
    },
    "networks": [
      {
        "ssid": "Office-WiFi-5G",
        "bssid": "aabbccddeeff",
        "channel": 36,
        "frequency": 5180,
        "signal_strength": -42,
        "signal_quality": 85,
        "security": "WPA2-PSK",
        "bandwidth": "80MHz",
        "hidden": false,
        "vendor": "Cisco Systems",
        "last_seen": 1699123456789
      }
    ],
    "channel_utilization": [
      {
        "channel": 1,
        "frequency": 2412,
        "network_count": 8,
        "utilization_percent": 75,
        "interference_level": "high"
      }
    ],
    "band_analysis": {
      "2.4ghz": {
        "network_count": 18,
        "avg_signal_strength": -65,
        "congestion_level": "high"
      },
      "5ghz": {
        "network_count": 7,
        "avg_signal_strength": -58,
        "congestion_level": "medium"
      }
    }
  }
}
```

#### 實作範例
```python
import subprocess
import json
import re
import time

def wifi_channel_scan(args):
    """執行WiFi頻道掃描"""
    scan_type = args.get("scan_type", "active")
    channels = args.get("channels", [])
    duration = args.get("duration_per_channel", 500)
    include_hidden = args.get("include_hidden", True)
    band = args.get("band", "both")
    
    try:
        # 使用iwlist進行掃描
        scan_results = perform_iwlist_scan(scan_type, channels, band)
        
        # 解析掃描結果
        networks = parse_scan_results(scan_results)
        
        # 分析頻道使用情況
        channel_analysis = analyze_channel_utilization(networks)
        
        # 生成頻段分析
        band_analysis = analyze_band_usage(networks)
        
        return {
            "scan_summary": {
                "total_networks": len(networks),
                "channels_scanned": len(set(n["channel"] for n in networks)),
                "scan_duration_ms": duration * len(channels) if channels else 13000,
                "strongest_signal": max(n["signal_strength"] for n in networks) if networks else 0,
                "weakest_signal": min(n["signal_strength"] for n in networks) if networks else 0
            },
            "networks": networks,
            "channel_utilization": channel_analysis,
            "band_analysis": band_analysis
        }
        
    except Exception as e:
        return {"error": f"WiFi scan failed: {str(e)}"}

def perform_iwlist_scan(scan_type, channels, band):
    """執行iwlist掃描"""
    interface = get_wifi_interface()
    if not interface:
        raise Exception("No WiFi interface found")
    
    # 設定掃描參數
    cmd = ["iwlist", interface, "scan"]
    
    # 如果指定了頻道，先設定頻道
    if channels:
        for channel in channels:
            subprocess.run(["iwconfig", interface, "channel", str(channel)], 
                         capture_output=True)
            time.sleep(0.1)
    
    # 執行掃描
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
    
    if result.returncode != 0:
        raise Exception(f"iwlist scan failed: {result.stderr}")
    
    return result.stdout

def parse_scan_results(scan_output):
    """解析iwlist掃描結果"""
    networks = []
    current_network = {}
    
    for line in scan_output.split('\n'):
        line = line.strip()
        
        # 新的AP
        if line.startswith('Cell'):
            if current_network:
                networks.append(current_network)
            current_network = {}
            
            # 提取BSSID
            bssid_match = re.search(r'Address: ([a-fA-F0-9:]{17})', line)
            if bssid_match:
                current_network['bssid'] = bssid_match.group(1).lower()
        
        # SSID
        elif 'ESSID:' in line:
            essid_match = re.search(r'ESSID:"([^"]*)"', line)
            if essid_match:
                current_network['ssid'] = essid_match.group(1)
            else:
                current_network['ssid'] = '<hidden>'
                current_network['hidden'] = True
        
        # 頻道和頻率
        elif 'Channel:' in line:
            channel_match = re.search(r'Channel:(\d+)', line)
            freq_match = re.search(r'Frequency:([\d.]+) GHz', line)
            
            if channel_match:
                current_network['channel'] = int(channel_match.group(1))
            if freq_match:
                freq_ghz = float(freq_match.group(1))
                current_network['frequency'] = int(freq_ghz * 1000)
        
        # 信號強度
        elif 'Signal level=' in line:
            signal_match = re.search(r'Signal level=(-?\d+)', line)
            if signal_match:
                current_network['signal_strength'] = int(signal_match.group(1))
                current_network['signal_quality'] = calculate_signal_quality(int(signal_match.group(1)))
        
        # 加密方式
        elif 'Encryption key:' in line:
            if 'off' in line:
                current_network['security'] = 'Open'
            else:
                current_network['security'] = 'Encrypted'
        
        elif 'IE: WPA' in line:
            if 'WPA2' in line:
                current_network['security'] = 'WPA2-PSK'
            else:
                current_network['security'] = 'WPA-PSK'
    
    # 添加最後一個網路
    if current_network:
        networks.append(current_network)
    
    # 填充缺失的欄位
    for network in networks:
        network.setdefault('hidden', False)
        network.setdefault('vendor', get_vendor_from_bssid(network.get('bssid', '')))
        network.setdefault('bandwidth', estimate_bandwidth(network.get('channel', 0)))
        network['last_seen'] = int(time.time() * 1000)
    
    return networks

def calculate_signal_quality(signal_strength):
    """計算信號品質百分比"""
    if signal_strength >= -30:
        return 100
    elif signal_strength >= -50:
        return 80
    elif signal_strength >= -60:
        return 60
    elif signal_strength >= -70:
        return 40
    elif signal_strength >= -80:
        return 20
    else:
        return 0

def analyze_channel_utilization(networks):
    """分析頻道使用情況"""
    channel_stats = {}
    
    for network in networks:
        channel = network.get('channel', 0)
        if channel not in channel_stats:
            channel_stats[channel] = {
                'channel': channel,
                'frequency': network.get('frequency', 0),
                'network_count': 0,
                'signal_strengths': []
            }
        
        channel_stats[channel]['network_count'] += 1
        channel_stats[channel]['signal_strengths'].append(network.get('signal_strength', -100))
    
    # 計算利用率和干擾等級
    utilization = []
    for channel, stats in channel_stats.items():
        # 簡化的利用率計算
        utilization_percent = min(stats['network_count'] * 12.5, 100)  # 假設8個網路為100%
        
        # 干擾等級
        if utilization_percent > 75:
            interference = "high"
        elif utilization_percent > 50:
            interference = "medium"
        elif utilization_percent > 25:
            interference = "low"
        else:
            interference = "minimal"
        
        utilization.append({
            'channel': channel,
            'frequency': stats['frequency'],
            'network_count': stats['network_count'],
            'utilization_percent': round(utilization_percent, 1),
            'interference_level': interference
        })
    
    return sorted(utilization, key=lambda x: x['channel'])

def get_wifi_interface():
    """獲取WiFi網路介面名稱"""
    try:
        result = subprocess.run(['iwconfig'], capture_output=True, text=True)
        for line in result.stdout.split('\n'):
            if 'IEEE 802.11' in line:
                return line.split()[0]
    except:
        pass
    
    # 嘗試常見的介面名稱
    for interface in ['wlan0', 'wlp3s0', 'wifi0']:
        try:
            result = subprocess.run(['iwconfig', interface], capture_output=True, text=True)
            if 'IEEE 802.11' in result.stdout:
                return interface
        except:
            continue
    
    return None
```

### 2. 干擾分析 (Interference Analysis)

#### 功能說明
分析WiFi頻段的干擾來源，包含相鄰頻道干擾和非WiFi干擾。

#### 命令格式
```json
{
  "id": "cmd-1699123456790",
  "op": "wifi_analyze_interference",
  "schema": "cmd.wifi_analyze_interference/1.0",
  "args": {
    "current_channel": 6,
    "scan_duration": 30,
    "include_spectrum": true,
    "threshold_dbm": -70
  }
}
```

#### 結果格式
```json
{
  "result": {
    "interference_summary": {
      "overall_level": "medium",
      "primary_sources": ["adjacent_channel", "bluetooth"],
      "recommended_channels": [1, 11],
      "current_channel_score": 65
    },
    "channel_interference": [
      {
        "channel": 6,
        "frequency": 2437,
        "interference_level": "high",
        "interference_sources": [
          {
            "type": "adjacent_channel",
            "source_channel": 7,
            "signal_strength": -45,
            "overlap_percent": 75
          },
          {
            "type": "bluetooth",
            "frequency_range": "2402-2480",
            "signal_strength": -55,
            "duty_cycle": 15
          }
        ],
        "noise_floor": -95,
        "signal_to_noise": 25
      }
    ],
    "spectrum_analysis": {
      "2.4ghz_utilization": 78,
      "5ghz_utilization": 35,
      "peak_interference_freq": 2442,
      "interference_bandwidth": 22
    }
  }
}
```

### 3. 信號強度映射 (Signal Strength Mapping)

#### 功能說明
創建WiFi信號覆蓋範圍的熱圖，分析信號強度分佈。

#### 命令格式
```json
{
  "id": "cmd-1699123456791",
  "op": "wifi_signal_strength_map",
  "schema": "cmd.wifi_signal_strength_map/1.0",
  "args": {
    "target_ssid": "Office-WiFi",
    "measurement_points": [
      {"x": 0, "y": 0, "description": "AP位置"},
      {"x": 10, "y": 0, "description": "走廊"},
      {"x": 20, "y": 10, "description": "會議室"}
    ],
    "measurement_duration": 10
  }
}
```

#### 結果格式
```json
{
  "result": {
    "coverage_summary": {
      "total_points": 3,
      "strong_signal_points": 1,
      "weak_signal_points": 1,
      "dead_zones": 0,
      "avg_signal_strength": -58
    },
    "measurement_points": [
      {
        "x": 0,
        "y": 0,
        "description": "AP位置",
        "signal_strength": -25,
        "signal_quality": 95,
        "coverage_level": "excellent"
      },
      {
        "x": 10,
        "y": 0,
        "description": "走廊",
        "signal_strength": -55,
        "signal_quality": 70,
        "coverage_level": "good"
      },
      {
        "x": 20,
        "y": 10,
        "description": "會議室",
        "signal_strength": -78,
        "signal_quality": 35,
        "coverage_level": "poor"
      }
    ],
    "coverage_recommendations": [
      "考慮在(20,10)附近增加AP或信號延伸器",
      "會議室信號較弱，建議調整AP位置或功率"
    ]
  }
}
```

### 4. 漫遊最佳化 (Roaming Optimization)

#### 功能說明
分析WiFi漫遊行為，提供漫遊參數調整建議。

#### 命令格式
```json
{
  "id": "cmd-1699123456792",
  "op": "wifi_roaming_optimization",
  "schema": "cmd.wifi_roaming_optimization/1.0",
  "args": {
    "current_bssid": "aabbccddeeff",
    "roaming_threshold": -70,
    "scan_interval": 10,
    "analyze_history": true
  }
}
```

#### 結果格式
```json
{
  "result": {
    "roaming_analysis": {
      "current_ap_score": 75,
      "available_aps": 3,
      "roaming_candidates": [
        {
          "bssid": "aabbccddeeff",
          "ssid": "Office-WiFi-5G",
          "signal_strength": -45,
          "load_estimate": 15,
          "roaming_score": 85
        }
      ],
      "roaming_recommended": true,
      "optimal_threshold": -65
    },
    "roaming_history": [
      {
        "timestamp": 1699120000000,
        "from_bssid": "aabbccddeeff",
        "to_bssid": "aabbccddeeff",
        "roaming_time_ms": 150,
        "success": true,
        "signal_improvement": 20
      }
    ],
    "optimization_recommendations": [
      "降低漫遊閾值至-65dBm以改善漫遊決策",
      "啟用11k/11v支援以加速漫遊過程"
    ]
  }
}
```

### 5. 頻譜利用率分析 (Spectrum Utilization)

#### 功能說明
分析WiFi頻譜的使用情況和效率。

#### 命令格式
```json
{
  "id": "cmd-1699123456793",
  "op": "wifi_spectrum_utilization",
  "schema": "cmd.wifi_spectrum_utilization/1.0",
  "args": {
    "band": "both",
    "analysis_duration": 60,
    "include_non_wifi": true
  }
}
```

#### 實作範例 - WiFi診斷整合

```python
import time
import subprocess
import json
from collections import defaultdict

class WiFiDiagnostics:
    def __init__(self):
        self.interface = self.get_wifi_interface()
        
    def run_comprehensive_analysis(self, args=None):
        """執行完整的WiFi診斷分析"""
        if not self.interface:
            return {"error": "No WiFi interface available"}
        
        results = {}
        
        try:
            # 1. 頻道掃描
            print("Running channel scan...")
            scan_result = self.wifi_channel_scan({})
            results['channel_scan'] = scan_result
            
            # 2. 干擾分析
            print("Analyzing interference...")
            interference_result = self.analyze_interference({})
            results['interference_analysis'] = interference_result
            
            # 3. 當前連接分析
            print("Analyzing current connection...")
            connection_result = self.analyze_current_connection()
            results['current_connection'] = connection_result
            
            # 4. 生成建議
            print("Generating recommendations...")
            recommendations = self.generate_wifi_recommendations(results)
            results['recommendations'] = recommendations
            
            return results
            
        except Exception as e:
            return {"error": f"WiFi analysis failed: {str(e)}"}
    
    def analyze_current_connection(self):
        """分析當前WiFi連接"""
        try:
            # 獲取當前連接資訊
            iwconfig_result = subprocess.run(
                ['iwconfig', self.interface],
                capture_output=True, text=True
            )
            
            connection_info = self.parse_iwconfig_output(iwconfig_result.stdout)
            
            # 獲取詳細統計
            stats_result = subprocess.run(
                ['cat', f'/proc/net/wireless'],
                capture_output=True, text=True
            )
            
            wireless_stats = self.parse_wireless_stats(stats_result.stdout, self.interface)
            
            # 合併資訊
            connection_info.update(wireless_stats)
            
            return connection_info
            
        except Exception as e:
            return {"error": f"Failed to analyze current connection: {str(e)}"}
    
    def parse_iwconfig_output(self, output):
        """解析iwconfig輸出"""
        info = {}
        
        for line in output.split('\n'):
            line = line.strip()
            
            # SSID
            if 'ESSID:' in line:
                essid_match = re.search(r'ESSID:"([^"]*)"', line)
                if essid_match:
                    info['connected_ssid'] = essid_match.group(1)
            
            # 接入點
            elif 'Access Point:' in line:
                ap_match = re.search(r'Access Point: ([a-fA-F0-9:]{17})', line)
                if ap_match:
                    info['connected_bssid'] = ap_match.group(1).lower()
            
            # 頻率
            elif 'Frequency:' in line:
                freq_match = re.search(r'Frequency:([\d.]+) GHz', line)
                if freq_match:
                    freq_ghz = float(freq_match.group(1))
                    info['frequency'] = freq_ghz
                    info['channel'] = self.frequency_to_channel(freq_ghz)
            
            # 比特率
            elif 'Bit Rate=' in line:
                rate_match = re.search(r'Bit Rate=([\d.]+) (\w+/s)', line)
                if rate_match:
                    rate = float(rate_match.group(1))
                    unit = rate_match.group(2)
                    if unit == 'Mb/s':
                        info['link_speed_mbps'] = rate
        
        return info
    
    def parse_wireless_stats(self, stats_output, interface):
        """解析無線統計資訊"""
        stats = {}
        
        for line in stats_output.split('\n'):
            if interface in line:
                parts = line.split()
                if len(parts) >= 4:
                    try:
                        # 信號品質
                        quality = parts[2].split('/')[0]
                        stats['link_quality'] = int(quality)
                        
                        # 信號等級
                        signal_level = int(parts[3])
                        stats['signal_strength'] = signal_level
                        
                        # 噪音等級
                        if len(parts) > 4:
                            noise_level = int(parts[4])
                            stats['noise_level'] = noise_level
                            stats['signal_to_noise'] = signal_level - noise_level
                    except (ValueError, IndexError):
                        pass
        
        return stats
    
    def generate_wifi_recommendations(self, analysis_results):
        """生成WiFi優化建議"""
        recommendations = []
        
        # 分析頻道掃描結果
        if 'channel_scan' in analysis_results:
            scan_data = analysis_results['channel_scan']
            
            # 檢查頻道擁塞
            if 'channel_utilization' in scan_data:
                for channel_info in scan_data['channel_utilization']:
                    if channel_info.get('utilization_percent', 0) > 80:
                        recommendations.append({
                            "type": "channel_congestion",
                            "priority": "high",
                            "message": f"頻道{channel_info['channel']}嚴重擁塞({channel_info['utilization_percent']}%)，建議更換頻道",
                            "suggested_action": "change_channel",
                            "suggested_channels": self.find_least_congested_channels(scan_data['channel_utilization'])
                        })
        
        # 分析當前連接
        if 'current_connection' in analysis_results:
            conn_data = analysis_results['current_connection']
            
            # 檢查信號強度
            signal_strength = conn_data.get('signal_strength', 0)
            if signal_strength < -70:
                recommendations.append({
                    "type": "weak_signal",
                    "priority": "medium",
                    "message": f"當前信號強度較弱({signal_strength}dBm)，可能影響網路效能",
                    "suggested_action": "improve_signal",
                    "suggestions": ["移動靠近路由器", "檢查障礙物", "考慮增加信號延伸器"]
                })
            
            # 檢查鏈路品質
            link_quality = conn_data.get('link_quality', 0)
            if link_quality < 50:
                recommendations.append({
                    "type": "poor_link_quality",
                    "priority": "medium",
                    "message": f"連結品質較差({link_quality}%)，建議檢查網路環境",
                    "suggested_action": "optimize_connection"
                })
        
        # 如果沒有發現問題
        if not recommendations:
            recommendations.append({
                "type": "all_good",
                "priority": "info",
                "message": "WiFi網路運作正常，無需特別優化",
                "suggested_action": "maintain"
            })
        
        return recommendations
    
    def find_least_congested_channels(self, channel_utilization):
        """尋找最不擁塞的頻道"""
        # 排序頻道利用率
        sorted_channels = sorted(channel_utilization, key=lambda x: x.get('utilization_percent', 100))
        
        # 回傳前3個最不擁塞的頻道
        return [ch['channel'] for ch in sorted_channels[:3]]
    
    def frequency_to_channel(self, freq_ghz):
        """將頻率轉換為頻道號"""
        if freq_ghz < 3:  # 2.4GHz
            return int((freq_ghz - 2.412) / 0.005) + 1
        else:  # 5GHz
            return int((freq_ghz - 5.000) / 0.005)

# 使用範例
if __name__ == "__main__":
    wifi_diag = WiFiDiagnostics()
    
    print("Starting comprehensive WiFi analysis...")
    results = wifi_diag.run_comprehensive_analysis()
    
    print("\nWiFi Analysis Results:")
    print("=" * 50)
    print(json.dumps(results, indent=2, ensure_ascii=False))
```

## WiFi診斷工具整合

### RTK Controller集成

```go
// internal/diagnostics/wifi_diagnostics.go
package diagnostics

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
)

type WiFiDiagnostics struct {
    logger Logger
    mqtt   MQTTClient
}

func NewWiFiDiagnostics(logger Logger, mqtt MQTTClient) *WiFiDiagnostics {
    return &WiFiDiagnostics{
        logger: logger,
        mqtt:   mqtt,
    }
}

func (wd *WiFiDiagnostics) RunChannelScan(ctx context.Context, deviceID string, args map[string]interface{}) (*DiagnosticResult, error) {
    wd.logger.Info("Starting WiFi channel scan", "device", deviceID)
    
    command := &Command{
        ID:     generateCommandID(),
        Op:     "wifi_scan_channels",
        Schema: "cmd.wifi_scan_channels/1.0",
        Args:   args,
    }
    
    result, err := wd.sendCommandToDevice(ctx, deviceID, command)
    if err != nil {
        return nil, fmt.Errorf("WiFi channel scan failed: %w", err)
    }
    
    // 分析掃描結果
    analysis := wd.analyzeChannelScanResult(result)
    
    return &DiagnosticResult{
        TestType:  "wifi_channel_scan",
        DeviceID:  deviceID,
        Status:    "completed",
        Result:    result,
        Analysis:  analysis,
        Timestamp: time.Now(),
    }, nil
}

func (wd *WiFiDiagnostics) RunInterferenceAnalysis(ctx context.Context, deviceID string, args map[string]interface{}) (*DiagnosticResult, error) {
    wd.logger.Info("Starting WiFi interference analysis", "device", deviceID)
    
    command := &Command{
        ID:     generateCommandID(),
        Op:     "wifi_analyze_interference",
        Schema: "cmd.wifi_analyze_interference/1.0",
        Args:   args,
    }
    
    result, err := wd.sendCommandToDevice(ctx, deviceID, command)
    if err != nil {
        return nil, fmt.Errorf("WiFi interference analysis failed: %w", err)
    }
    
    return &DiagnosticResult{
        TestType:  "wifi_interference_analysis",
        DeviceID:  deviceID,
        Status:    "completed",
        Result:    result,
        Timestamp: time.Now(),
    }, nil
}

func (wd *WiFiDiagnostics) analyzeChannelScanResult(result map[string]interface{}) *WiFiAnalysis {
    analysis := &WiFiAnalysis{
        Overall: "unknown",
        Issues:  make([]WiFiIssue, 0),
        Recommendations: make([]string, 0),
    }
    
    // 檢查頻道擁塞
    if channelUtil, ok := result["channel_utilization"].([]interface{}); ok {
        congestionIssues := wd.checkChannelCongestion(channelUtil)
        analysis.Issues = append(analysis.Issues, congestionIssues...)
    }
    
    // 檢查頻段使用
    if bandAnalysis, ok := result["band_analysis"].(map[string]interface{}); ok {
        bandIssues := wd.checkBandUsage(bandAnalysis)
        analysis.Issues = append(analysis.Issues, bandIssues...)
    }
    
    // 生成整體評級
    analysis.Overall = wd.calculateOverallWiFiGrade(analysis.Issues)
    
    // 生成建議
    analysis.Recommendations = wd.generateWiFiRecommendations(analysis.Issues)
    
    return analysis
}

type WiFiAnalysis struct {
    Overall         string      `json:"overall_grade"`
    Issues          []WiFiIssue `json:"issues"`
    Recommendations []string    `json:"recommendations"`
}

type WiFiIssue struct {
    Type        string      `json:"type"`
    Severity    string      `json:"severity"`
    Description string      `json:"description"`
    Channel     int         `json:"channel,omitempty"`
    Value       float64     `json:"value,omitempty"`
}

func (wd *WiFiDiagnostics) checkChannelCongestion(channelUtil []interface{}) []WiFiIssue {
    var issues []WiFiIssue
    
    for _, ch := range channelUtil {
        if chMap, ok := ch.(map[string]interface{}); ok {
            channel := int(chMap["channel"].(float64))
            utilization := chMap["utilization_percent"].(float64)
            
            if utilization > 80 {
                issues = append(issues, WiFiIssue{
                    Type:        "channel_congestion",
                    Severity:    "high",
                    Description: fmt.Sprintf("頻道%d嚴重擁塞", channel),
                    Channel:     channel,
                    Value:       utilization,
                })
            } else if utilization > 60 {
                issues = append(issues, WiFiIssue{
                    Type:        "channel_congestion",
                    Severity:    "medium",
                    Description: fmt.Sprintf("頻道%d中度擁塞", channel),
                    Channel:     channel,
                    Value:       utilization,
                })
            }
        }
    }
    
    return issues
}
```

### CLI命令支援

```go
// internal/cli/wifi_commands.go
package cli

import (
    "encoding/json"
    "fmt"
    "github.com/spf13/cobra"
)

func (h *Handler) buildWiFiCommands() *cobra.Command {
    wifiCmd := &cobra.Command{
        Use:   "wifi",
        Short: "WiFi diagnostic commands",
        Long:  "Run various WiFi diagnostic tests and analysis",
    }
    
    // 頻道掃描命令
    scanCmd := &cobra.Command{
        Use:   "scan [device_id]",
        Short: "Scan WiFi channels",
        Args:  cobra.ExactArgs(1),
        RunE:  h.handleWiFiScan,
    }
    
    scanCmd.Flags().StringSlice("channels", []string{}, "Specific channels to scan")
    scanCmd.Flags().String("band", "both", "Band to scan: 2.4ghz, 5ghz, both")
    scanCmd.Flags().Int("duration", 500, "Duration per channel in ms")
    
    // 干擾分析命令
    interferenceCmd := &cobra.Command{
        Use:   "interference [device_id]",
        Short: "Analyze WiFi interference",
        Args:  cobra.ExactArgs(1),
        RunE:  h.handleWiFiInterference,
    }
    
    interferenceCmd.Flags().Int("channel", 0, "Current channel for analysis")
    interferenceCmd.Flags().Int("duration", 30, "Analysis duration in seconds")
    
    // 信號強度映射
    signalMapCmd := &cobra.Command{
        Use:   "signal_map [device_id]",
        Short: "Create signal strength map",
        Args:  cobra.ExactArgs(1),
        RunE:  h.handleWiFiSignalMap,
    }
    
    signalMapCmd.Flags().String("ssid", "", "Target SSID for mapping")
    
    wifiCmd.AddCommand(scanCmd, interferenceCmd, signalMapCmd)
    
    return wifiCmd
}

func (h *Handler) handleWiFiScan(cmd *cobra.Command, args []string) error {
    deviceID := args[0]
    
    channels, _ := cmd.Flags().GetStringSlice("channels")
    band, _ := cmd.Flags().GetString("band")
    duration, _ := cmd.Flags().GetInt("duration")
    
    scanArgs := map[string]interface{}{
        "band":                  band,
        "duration_per_channel":  duration,
        "include_hidden":        true,
    }
    
    if len(channels) > 0 {
        scanArgs["channels"] = channels
    }
    
    fmt.Printf("Scanning WiFi channels on device %s...\n", deviceID)
    
    result, err := h.wifiDiagnostics.RunChannelScan(cmd.Context(), deviceID, scanArgs)
    if err != nil {
        return fmt.Errorf("WiFi scan failed: %w", err)
    }
    
    return h.displayWiFiScanResult(result)
}

func (h *Handler) displayWiFiScanResult(result *DiagnosticResult) error {
    fmt.Printf("WiFi Channel Scan Results for Device %s\n", result.DeviceID)
    fmt.Printf("=========================================\n")
    fmt.Printf("Status: %s\n", result.Status)
    fmt.Printf("Timestamp: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
    
    if result.Status == "completed" {
        resultData := result.Result
        
        // 顯示掃描摘要
        if summary, ok := resultData["scan_summary"].(map[string]interface{}); ok {
            fmt.Printf("\nScan Summary:\n")
            fmt.Printf("  Total Networks: %.0f\n", summary["total_networks"].(float64))
            fmt.Printf("  Channels Scanned: %.0f\n", summary["channels_scanned"].(float64))
            fmt.Printf("  Strongest Signal: %.0f dBm\n", summary["strongest_signal"].(float64))
            fmt.Printf("  Weakest Signal: %.0f dBm\n", summary["weakest_signal"].(float64))
        }
        
        // 顯示頻道利用率
        if channelUtil, ok := resultData["channel_utilization"].([]interface{}); ok {
            fmt.Printf("\nChannel Utilization:\n")
            fmt.Printf("%-8s %-10s %-12s %-15s %s\n", "Channel", "Frequency", "Networks", "Utilization", "Interference")
            fmt.Printf("%-8s %-10s %-12s %-15s %s\n", "-------", "---------", "--------", "-----------", "------------")
            
            for _, ch := range channelUtil {
                if chMap, ok := ch.(map[string]interface{}); ok {
                    fmt.Printf("%-8.0f %-10.0f %-12.0f %-15.1f%% %s\n",
                        chMap["channel"].(float64),
                        chMap["frequency"].(float64),
                        chMap["network_count"].(float64),
                        chMap["utilization_percent"].(float64),
                        chMap["interference_level"].(string))
                }
            }
        }
        
        // 顯示分析結果
        if result.Analysis != nil {
            fmt.Printf("\nAnalysis Results:\n")
            fmt.Printf("  Overall Grade: %s\n", result.Analysis.Overall)
            
            if len(result.Analysis.Issues) > 0 {
                fmt.Printf("  Issues Found:\n")
                for _, issue := range result.Analysis.Issues {
                    fmt.Printf("    - %s (%s): %s\n", issue.Type, issue.Severity, issue.Description)
                }
            }
            
            if len(result.Analysis.Recommendations) > 0 {
                fmt.Printf("  Recommendations:\n")
                for _, rec := range result.Analysis.Recommendations {
                    fmt.Printf("    - %s\n", rec)
                }
            }
        }
    } else {
        fmt.Printf("Error: %s\n", result.Error)
    }
    
    return nil
}
```

## 自動化WiFi優化

### 智慧頻道選擇

```python
class WiFiOptimizer:
    def __init__(self):
        self.channel_weights = {
            # 2.4GHz 非重疊頻道權重較高
            1: 1.0, 6: 1.0, 11: 1.0,
            # 其他2.4GHz頻道
            2: 0.3, 3: 0.3, 4: 0.3, 5: 0.3,
            7: 0.3, 8: 0.3, 9: 0.3, 10: 0.3,
            # 5GHz頻道
            36: 0.9, 40: 0.9, 44: 0.9, 48: 0.9,
            149: 0.9, 153: 0.9, 157: 0.9, 161: 0.9
        }
    
    def recommend_optimal_channel(self, scan_results):
        """推薦最佳頻道"""
        channel_scores = {}
        
        for channel_info in scan_results.get("channel_utilization", []):
            channel = channel_info["channel"]
            utilization = channel_info["utilization_percent"]
            interference = channel_info["interference_level"]
            
            # 基礎分數 (100 - 利用率)
            base_score = 100 - utilization
            
            # 干擾懲罰
            interference_penalty = {
                "minimal": 0,
                "low": 10,
                "medium": 25,
                "high": 50
            }.get(interference, 0)
            
            # 頻道權重
            channel_weight = self.channel_weights.get(channel, 0.5)
            
            # 最終分數
            final_score = (base_score - interference_penalty) * channel_weight
            channel_scores[channel] = max(final_score, 0)
        
        # 排序並回傳最佳頻道
        sorted_channels = sorted(channel_scores.items(), key=lambda x: x[1], reverse=True)
        
        return {
            "recommended_channel": sorted_channels[0][0] if sorted_channels else None,
            "channel_scores": dict(sorted_channels[:5]),  # 前5名
            "reasoning": self.generate_channel_reasoning(sorted_channels[0] if sorted_channels else None)
        }
    
    def generate_channel_reasoning(self, best_channel_info):
        """生成頻道選擇理由"""
        if not best_channel_info:
            return "無可用頻道"
        
        channel, score = best_channel_info
        
        if channel in [1, 6, 11]:
            return f"頻道{channel}是2.4GHz非重疊頻道，干擾較少"
        elif channel >= 36:
            return f"頻道{channel}是5GHz頻道，頻寬更大且干擾較少"
        else:
            return f"頻道{channel}當前使用率較低"
```

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Commands and Events Reference](../core/COMMANDS_EVENTS_REFERENCE.md)
- [Network Diagnostics Guide](NETWORK_DIAGNOSTICS.md)
- [Troubleshooting Guide](../guides/TROUBLESHOOTING_GUIDE.md)
- [IEEE 802.11 Standards](https://standards.ieee.org/findstds/standard/802.11-2016.html)
- [WiFi Analyzer Tools](https://github.com/VREMSoftwareDevelopment/WiFiAnalyzer)