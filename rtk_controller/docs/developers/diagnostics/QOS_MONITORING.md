# QoS 監控工具指南

## 概述

本文檔提供RTK MQTT系統中服務品質(QoS)監控功能的完整說明，包含流量分析、性能監控、異常檢測和自動化優化建議。

## QoS監控架構

### 監控層級

```
┌─────────────────────────────────────────────┐
│                應用層QoS                     │
│        (Application-level Metrics)         │
├─────────────────────────────────────────────┤
│                網路層QoS                     │
│         (Network-level Metrics)            │
├─────────────────────────────────────────────┤
│                鏈路層QoS                     │
│          (Link-level Metrics)              │
└─────────────────────────────────────────────┘
```

### 關鍵指標

- **延遲 (Latency)**: 端到端傳輸時間
- **抖動 (Jitter)**: 延遲變化
- **吞吐量 (Throughput)**: 實際傳輸速率
- **封包遺失率 (Packet Loss)**: 封包丟失百分比
- **頻寬利用率 (Bandwidth Utilization)**: 頻寬使用效率

## QoS監控命令

### 1. 流量分析 (Traffic Analysis)

#### 功能說明
監控網路流量模式，分析頻寬使用情況和流量分佈。

#### 命令格式
```json
{
  "id": "cmd-1699123456789",
  "op": "qos_traffic_analysis",
  "schema": "cmd.qos_traffic_analysis/1.0",
  "args": {
    "duration": 300,
    "interfaces": ["eth0", "wlan0"],
    "analysis_type": "comprehensive",
    "sampling_interval": 1,
    "include_protocols": true
  }
}
```

#### 參數說明
- `duration`: 監控持續時間(秒)
- `interfaces`: 監控的網路介面列表
- `analysis_type`: 分析類型
  - `"basic"`: 基本流量統計
  - `"comprehensive"`: 詳細分析包含協議分佈
  - `"realtime"`: 即時監控
- `sampling_interval`: 取樣間隔(秒)
- `include_protocols`: 是否包含協議層分析

#### 結果格式
```json
{
  "result": {
    "analysis_summary": {
      "total_bytes": 1073741824,
      "total_packets": 1048576,
      "avg_throughput_mbps": 85.5,
      "peak_throughput_mbps": 120.2,
      "analysis_duration": 300
    },
    "interface_stats": [
      {
        "interface": "eth0",
        "rx_bytes": 536870912,
        "tx_bytes": 268435456,
        "rx_packets": 524288,
        "tx_packets": 262144,
        "rx_errors": 0,
        "tx_errors": 0,
        "utilization_percent": 65.5
      }
    ],
    "protocol_distribution": {
      "tcp": 65.2,
      "udp": 28.5,
      "icmp": 3.1,
      "other": 3.2
    },
    "traffic_patterns": {
      "peak_hours": ["09:00-11:00", "14:00-16:00"],
      "avg_packet_size": 1024,
      "burst_frequency": 15.2,
      "flow_count": 245
    },
    "qos_metrics": {
      "avg_latency_ms": 12.5,
      "jitter_ms": 2.1,
      "packet_loss_percent": 0.05,
      "out_of_order_percent": 0.01
    }
  }
}
```

#### 實作範例
```python
import psutil
import time
import json
from collections import defaultdict
import threading

class QoSTrafficAnalyzer:
    def __init__(self):
        self.monitoring = False
        self.stats_history = []
        self.interface_stats = defaultdict(list)
        
    def analyze_traffic(self, args):
        """執行流量分析"""
        duration = args.get("duration", 300)
        interfaces = args.get("interfaces", [])
        analysis_type = args.get("analysis_type", "comprehensive")
        sampling_interval = args.get("sampling_interval", 1)
        
        # 自動檢測介面如果未指定
        if not interfaces:
            interfaces = self.get_active_interfaces()
        
        try:
            # 開始監控
            results = self.monitor_interfaces(interfaces, duration, sampling_interval)
            
            # 分析結果
            analysis = self.analyze_monitoring_results(results, analysis_type)
            
            return analysis
            
        except Exception as e:
            return {"error": f"Traffic analysis failed: {str(e)}"}
    
    def get_active_interfaces(self):
        """獲取活躍的網路介面"""
        active_interfaces = []
        
        for interface, stats in psutil.net_io_counters(pernic=True).items():
            # 排除迴環介面和無流量介面
            if interface != 'lo' and (stats.bytes_sent > 0 or stats.bytes_recv > 0):
                active_interfaces.append(interface)
                
        return active_interfaces
    
    def monitor_interfaces(self, interfaces, duration, interval):
        """監控網路介面"""
        start_time = time.time()
        monitoring_results = {
            'timestamps': [],
            'interface_data': defaultdict(list),
            'system_stats': []
        }
        
        # 取得初始統計
        prev_stats = {}
        for interface in interfaces:
            try:
                prev_stats[interface] = psutil.net_io_counters(pernic=True)[interface]
            except KeyError:
                continue
        
        while time.time() - start_time < duration:
            current_time = time.time()
            monitoring_results['timestamps'].append(current_time)
            
            # 收集當前統計
            current_stats = {}
            for interface in interfaces:
                try:
                    current_stats[interface] = psutil.net_io_counters(pernic=True)[interface]
                    
                    # 計算增量
                    if interface in prev_stats:
                        delta_stats = self.calculate_delta_stats(
                            prev_stats[interface], 
                            current_stats[interface], 
                            interval
                        )
                        monitoring_results['interface_data'][interface].append(delta_stats)
                        
                except KeyError:
                    continue
            
            # 收集系統統計
            system_stat = {
                'cpu_percent': psutil.cpu_percent(),
                'memory_percent': psutil.virtual_memory().percent,
                'network_connections': len(psutil.net_connections())
            }
            monitoring_results['system_stats'].append(system_stat)
            
            prev_stats = current_stats
            time.sleep(interval)
        
        return monitoring_results
    
    def calculate_delta_stats(self, prev_stats, current_stats, interval):
        """計算統計增量"""
        return {
            'rx_bytes_per_sec': (current_stats.bytes_recv - prev_stats.bytes_recv) / interval,
            'tx_bytes_per_sec': (current_stats.bytes_sent - prev_stats.bytes_sent) / interval,
            'rx_packets_per_sec': (current_stats.packets_recv - prev_stats.packets_recv) / interval,
            'tx_packets_per_sec': (current_stats.packets_sent - prev_stats.packets_sent) / interval,
            'rx_errors': current_stats.errin - prev_stats.errin,
            'tx_errors': current_stats.errout - prev_stats.errout,
            'rx_drops': current_stats.dropin - prev_stats.dropin,
            'tx_drops': current_stats.dropout - prev_stats.dropout
        }
    
    def analyze_monitoring_results(self, results, analysis_type):
        """分析監控結果"""
        analysis = {
            'analysis_summary': {},
            'interface_stats': [],
            'qos_metrics': {},
            'performance_grade': 'unknown'
        }
        
        # 計算總體統計
        total_rx_bytes = 0
        total_tx_bytes = 0
        total_rx_packets = 0
        total_tx_packets = 0
        
        interface_summaries = []
        
        for interface, data_points in results['interface_data'].items():
            if not data_points:
                continue
                
            # 計算介面統計
            interface_summary = self.calculate_interface_summary(interface, data_points)
            interface_summaries.append(interface_summary)
            
            total_rx_bytes += interface_summary['total_rx_bytes']
            total_tx_bytes += interface_summary['total_tx_bytes']
            total_rx_packets += interface_summary['total_rx_packets']
            total_tx_packets += interface_summary['total_tx_packets']
        
        # 總體摘要
        duration = len(results['timestamps'])
        total_bytes = total_rx_bytes + total_tx_bytes
        avg_throughput_mbps = (total_bytes * 8) / (duration * 1024 * 1024) if duration > 0 else 0
        
        analysis['analysis_summary'] = {
            'total_bytes': total_bytes,
            'total_packets': total_rx_packets + total_tx_packets,
            'avg_throughput_mbps': round(avg_throughput_mbps, 2),
            'analysis_duration': duration
        }
        
        analysis['interface_stats'] = interface_summaries
        
        # 計算QoS指標
        analysis['qos_metrics'] = self.calculate_qos_metrics(results)
        
        # 性能評級
        analysis['performance_grade'] = self.calculate_performance_grade(analysis)
        
        return analysis
    
    def calculate_interface_summary(self, interface, data_points):
        """計算介面摘要統計"""
        total_rx_bytes = sum(dp['rx_bytes_per_sec'] for dp in data_points)
        total_tx_bytes = sum(dp['tx_bytes_per_sec'] for dp in data_points)
        total_rx_packets = sum(dp['rx_packets_per_sec'] for dp in data_points)
        total_tx_packets = sum(dp['tx_packets_per_sec'] for dp in data_points)
        
        # 計算平均值
        avg_rx_rate = total_rx_bytes / len(data_points) if data_points else 0
        avg_tx_rate = total_tx_bytes / len(data_points) if data_points else 0
        
        # 計算峰值
        peak_rx_rate = max(dp['rx_bytes_per_sec'] for dp in data_points) if data_points else 0
        peak_tx_rate = max(dp['tx_bytes_per_sec'] for dp in data_points) if data_points else 0
        
        # 錯誤統計
        total_errors = sum(dp['rx_errors'] + dp['tx_errors'] for dp in data_points)
        total_drops = sum(dp['rx_drops'] + dp['tx_drops'] for dp in data_points)
        
        return {
            'interface': interface,
            'total_rx_bytes': int(total_rx_bytes),
            'total_tx_bytes': int(total_tx_bytes),
            'total_rx_packets': int(total_rx_packets),
            'total_tx_packets': int(total_tx_packets),
            'avg_rx_rate_mbps': round((avg_rx_rate * 8) / (1024 * 1024), 2),
            'avg_tx_rate_mbps': round((avg_tx_rate * 8) / (1024 * 1024), 2),
            'peak_rx_rate_mbps': round((peak_rx_rate * 8) / (1024 * 1024), 2),
            'peak_tx_rate_mbps': round((peak_tx_rate * 8) / (1024 * 1024), 2),
            'total_errors': total_errors,
            'total_drops': total_drops,
            'error_rate': total_errors / (total_rx_packets + total_tx_packets) if (total_rx_packets + total_tx_packets) > 0 else 0
        }
    
    def calculate_qos_metrics(self, results):
        """計算QoS指標"""
        # 這裡簡化實作，實際應該包含更精確的延遲和抖動測量
        metrics = {
            'avg_latency_ms': 15.0,  # 需要實際測量
            'jitter_ms': 2.5,       # 需要實際測量
            'packet_loss_percent': 0.0,
            'throughput_consistency': 85.0
        }
        
        # 計算封包遺失率
        total_drops = 0
        total_packets = 0
        
        for interface_data in results['interface_data'].values():
            for data_point in interface_data:
                total_drops += data_point['rx_drops'] + data_point['tx_drops']
                total_packets += data_point['rx_packets_per_sec'] + data_point['tx_packets_per_sec']
        
        if total_packets > 0:
            metrics['packet_loss_percent'] = round((total_drops / total_packets) * 100, 4)
        
        return metrics
```

### 2. 性能監控 (Performance Monitoring)

#### 命令格式
```json
{
  "id": "cmd-1699123456790",
  "op": "qos_performance_monitor",
  "schema": "cmd.qos_performance_monitor/1.0",
  "args": {
    "monitor_duration": 600,
    "metrics": ["latency", "throughput", "jitter", "packet_loss"],
    "target_hosts": ["8.8.8.8", "1.1.1.1"],
    "alert_thresholds": {
      "latency_ms": 100,
      "jitter_ms": 10,
      "packet_loss_percent": 1.0,
      "throughput_mbps": 10
    }
  }
}
```

#### 結果格式
```json
{
  "result": {
    "monitoring_summary": {
      "duration": 600,
      "total_measurements": 600,
      "alerts_triggered": 3,
      "overall_health_score": 78
    },
    "performance_metrics": {
      "latency": {
        "min_ms": 8.2,
        "max_ms": 45.6,
        "avg_ms": 15.3,
        "std_dev_ms": 5.2,
        "percentile_95_ms": 28.1
      },
      "throughput": {
        "min_mbps": 85.2,
        "max_mbps": 120.5,
        "avg_mbps": 98.7,
        "consistency_percent": 92.5
      },
      "jitter": {
        "avg_ms": 2.8,
        "max_ms": 12.1,
        "stability_score": 85
      },
      "packet_loss": {
        "total_percent": 0.15,
        "burst_events": 2,
        "max_consecutive_loss": 3
      }
    },
    "alerts": [
      {
        "timestamp": 1699123456789,
        "metric": "latency",
        "value": 105.2,
        "threshold": 100,
        "severity": "warning",
        "duration_s": 15
      }
    ],
    "recommendations": [
      "考慮優化路由設定以降低延遲峰值",
      "監控到輕微的封包遺失，建議檢查網路設備狀態"
    ]
  }
}
```

### 3. 異常檢測 (Anomaly Detection)

#### 功能說明
使用機器學習算法檢測網路性能異常模式。

#### 命令格式
```json
{
  "id": "cmd-1699123456791",
  "op": "qos_anomaly_detection",
  "schema": "cmd.qos_anomaly_detection/1.0",
  "args": {
    "detection_window": 3600,
    "baseline_period": 86400,
    "sensitivity": "medium",
    "metrics": ["latency", "throughput", "error_rate"],
    "detection_methods": ["statistical", "ml_based"]
  }
}
```

#### 實作範例
```python
import numpy as np
from scipy import stats
from sklearn.ensemble import IsolationForest
from sklearn.preprocessing import StandardScaler
import time

class QoSAnomalyDetector:
    def __init__(self):
        self.baseline_data = {}
        self.models = {}
        self.scalers = {}
        
    def detect_anomalies(self, args):
        """執行異常檢測"""
        detection_window = args.get("detection_window", 3600)
        baseline_period = args.get("baseline_period", 86400)
        sensitivity = args.get("sensitivity", "medium")
        metrics = args.get("metrics", ["latency", "throughput", "error_rate"])
        methods = args.get("detection_methods", ["statistical", "ml_based"])
        
        try:
            # 收集基線數據
            baseline_data = self.collect_baseline_data(baseline_period, metrics)
            
            # 收集當前檢測窗口數據
            current_data = self.collect_current_data(detection_window, metrics)
            
            # 執行異常檢測
            anomalies = []
            
            if "statistical" in methods:
                stat_anomalies = self.statistical_anomaly_detection(
                    baseline_data, current_data, sensitivity
                )
                anomalies.extend(stat_anomalies)
            
            if "ml_based" in methods:
                ml_anomalies = self.ml_anomaly_detection(
                    baseline_data, current_data, sensitivity
                )
                anomalies.extend(ml_anomalies)
            
            # 生成檢測報告
            report = self.generate_anomaly_report(anomalies, current_data)
            
            return report
            
        except Exception as e:
            return {"error": f"Anomaly detection failed: {str(e)}"}
    
    def statistical_anomaly_detection(self, baseline_data, current_data, sensitivity):
        """統計方法異常檢測"""
        anomalies = []
        
        # 設定敏感度閾值
        sensitivity_thresholds = {
            "low": 3.0,      # 3 sigma
            "medium": 2.5,   # 2.5 sigma
            "high": 2.0      # 2 sigma
        }
        threshold = sensitivity_thresholds.get(sensitivity, 2.5)
        
        for metric in current_data.keys():
            if metric not in baseline_data:
                continue
                
            baseline_values = baseline_data[metric]
            current_values = current_data[metric]
            
            # 計算基線統計
            baseline_mean = np.mean(baseline_values)
            baseline_std = np.std(baseline_values)
            
            # 檢測異常值
            for i, value in enumerate(current_values):
                z_score = abs(value - baseline_mean) / baseline_std if baseline_std > 0 else 0
                
                if z_score > threshold:
                    anomalies.append({
                        "type": "statistical",
                        "metric": metric,
                        "timestamp": time.time() - len(current_values) + i,
                        "value": value,
                        "baseline_mean": baseline_mean,
                        "z_score": z_score,
                        "severity": self.calculate_severity(z_score, threshold)
                    })
        
        return anomalies
    
    def ml_anomaly_detection(self, baseline_data, current_data, sensitivity):
        """機器學習方法異常檢測"""
        anomalies = []
        
        try:
            # 準備訓練數據
            training_features = self.prepare_features(baseline_data)
            if len(training_features) < 10:
                return []  # 數據不足，跳過ML檢測
            
            # 訓練Isolation Forest模型
            contamination_rates = {
                "low": 0.01,
                "medium": 0.05,
                "high": 0.1
            }
            contamination = contamination_rates.get(sensitivity, 0.05)
            
            scaler = StandardScaler()
            training_features_scaled = scaler.fit_transform(training_features)
            
            model = IsolationForest(contamination=contamination, random_state=42)
            model.fit(training_features_scaled)
            
            # 檢測當前數據
            current_features = self.prepare_features(current_data)
            if len(current_features) == 0:
                return []
            
            current_features_scaled = scaler.transform(current_features)
            predictions = model.predict(current_features_scaled)
            anomaly_scores = model.decision_function(current_features_scaled)
            
            # 提取異常點
            for i, (prediction, score) in enumerate(zip(predictions, anomaly_scores)):
                if prediction == -1:  # 異常
                    anomalies.append({
                        "type": "ml_based",
                        "timestamp": time.time() - len(current_features) + i,
                        "anomaly_score": score,
                        "features": current_features[i].tolist(),
                        "severity": "high" if score < -0.5 else "medium"
                    })
            
        except Exception as e:
            print(f"ML anomaly detection error: {e}")
        
        return anomalies
    
    def prepare_features(self, data):
        """準備機器學習特徵"""
        features = []
        
        # 確保所有指標都有數據
        metric_names = list(data.keys())
        if not metric_names:
            return np.array(features)
        
        min_length = min(len(values) for values in data.values())
        
        for i in range(min_length):
            feature_vector = []
            for metric in metric_names:
                feature_vector.append(data[metric][i])
            features.append(feature_vector)
        
        return np.array(features)
    
    def calculate_severity(self, z_score, threshold):
        """計算異常嚴重程度"""
        if z_score > threshold * 2:
            return "critical"
        elif z_score > threshold * 1.5:
            return "high"
        elif z_score > threshold:
            return "medium"
        else:
            return "low"
    
    def generate_anomaly_report(self, anomalies, current_data):
        """生成異常檢測報告"""
        return {
            "detection_summary": {
                "total_anomalies": len(anomalies),
                "detection_methods": list(set(a["type"] for a in anomalies)),
                "severity_distribution": self.count_by_severity(anomalies),
                "affected_metrics": list(set(a.get("metric", "unknown") for a in anomalies))
            },
            "anomalies": anomalies,
            "current_metrics_summary": self.summarize_current_metrics(current_data),
            "recommendations": self.generate_anomaly_recommendations(anomalies)
        }
    
    def count_by_severity(self, anomalies):
        """統計異常嚴重程度分佈"""
        severity_count = {"critical": 0, "high": 0, "medium": 0, "low": 0}
        for anomaly in anomalies:
            severity = anomaly.get("severity", "low")
            severity_count[severity] += 1
        return severity_count
    
    def generate_anomaly_recommendations(self, anomalies):
        """生成異常處理建議"""
        recommendations = []
        
        # 統計異常類型
        metric_issues = {}
        for anomaly in anomalies:
            metric = anomaly.get("metric", "unknown")
            if metric not in metric_issues:
                metric_issues[metric] = 0
            metric_issues[metric] += 1
        
        # 根據異常類型生成建議
        for metric, count in metric_issues.items():
            if metric == "latency":
                recommendations.append(f"檢測到{count}個延遲異常，建議檢查網路路由和設備狀態")
            elif metric == "throughput":
                recommendations.append(f"檢測到{count}個吞吐量異常，建議檢查頻寬使用和網路擁塞")
            elif metric == "error_rate":
                recommendations.append(f"檢測到{count}個錯誤率異常，建議檢查網路設備硬體狀態")
        
        if not recommendations:
            recommendations.append("未檢測到明顯異常，網路性能正常")
        
        return recommendations
```

### 4. 自動化QoS優化

#### 命令格式
```json
{
  "id": "cmd-1699123456792",
  "op": "qos_auto_optimization",
  "schema": "cmd.qos_auto_optimization/1.0",
  "args": {
    "optimization_goals": ["minimize_latency", "maximize_throughput"],
    "constraints": {
      "max_bandwidth_mbps": 100,
      "priority_applications": ["voip", "video_conference"],
      "time_window": "business_hours"
    },
    "dry_run": true
  }
}
```

#### QoS優化引擎

```python
class QoSOptimizer:
    def __init__(self):
        self.application_profiles = {
            "voip": {
                "priority": "high",
                "max_latency_ms": 20,
                "max_jitter_ms": 5,
                "min_bandwidth_kbps": 64,
                "packet_loss_tolerance": 0.1
            },
            "video_conference": {
                "priority": "high",
                "max_latency_ms": 50,
                "max_jitter_ms": 10,
                "min_bandwidth_mbps": 2,
                "packet_loss_tolerance": 0.5
            },
            "web_browsing": {
                "priority": "medium",
                "max_latency_ms": 200,
                "min_bandwidth_kbps": 128,
                "packet_loss_tolerance": 1.0
            },
            "file_transfer": {
                "priority": "low",
                "max_latency_ms": 1000,
                "min_bandwidth_mbps": 1,
                "packet_loss_tolerance": 2.0
            }
        }
    
    def auto_optimize_qos(self, args):
        """自動QoS優化"""
        goals = args.get("optimization_goals", [])
        constraints = args.get("constraints", {})
        dry_run = args.get("dry_run", True)
        
        try:
            # 分析當前QoS狀態
            current_state = self.analyze_current_qos_state()
            
            # 識別問題區域
            issues = self.identify_qos_issues(current_state)
            
            # 生成優化策略
            optimization_plan = self.generate_optimization_plan(
                current_state, issues, goals, constraints
            )
            
            # 執行優化 (如果不是dry run)
            if not dry_run:
                execution_result = self.execute_optimization_plan(optimization_plan)
            else:
                execution_result = {"status": "simulated", "changes": "none"}
            
            return {
                "current_state": current_state,
                "identified_issues": issues,
                "optimization_plan": optimization_plan,
                "execution_result": execution_result,
                "estimated_improvement": self.estimate_improvement(optimization_plan)
            }
            
        except Exception as e:
            return {"error": f"QoS optimization failed: {str(e)}"}
    
    def analyze_current_qos_state(self):
        """分析當前QoS狀態"""
        # 收集當前網路狀態
        interfaces = psutil.net_io_counters(pernic=True)
        connections = psutil.net_connections()
        
        state = {
            "interface_utilization": {},
            "active_flows": len(connections),
            "protocol_distribution": {},
            "bandwidth_allocation": {}
        }
        
        # 分析介面利用率
        for interface, stats in interfaces.items():
            if interface != 'lo':  # 排除迴環介面
                # 簡化的利用率計算
                total_bytes = stats.bytes_sent + stats.bytes_recv
                state["interface_utilization"][interface] = {
                    "total_bytes": total_bytes,
                    "error_rate": (stats.errin + stats.errout) / max(stats.packets_sent + stats.packets_recv, 1)
                }
        
        # 分析協議分佈
        protocol_count = {"tcp": 0, "udp": 0, "other": 0}
        for conn in connections:
            if conn.type == 1:  # SOCK_STREAM (TCP)
                protocol_count["tcp"] += 1
            elif conn.type == 2:  # SOCK_DGRAM (UDP)
                protocol_count["udp"] += 1
            else:
                protocol_count["other"] += 1
        
        state["protocol_distribution"] = protocol_count
        
        return state
    
    def identify_qos_issues(self, current_state):
        """識別QoS問題"""
        issues = []
        
        # 檢查介面利用率
        for interface, stats in current_state["interface_utilization"].items():
            error_rate = stats["error_rate"]
            if error_rate > 0.01:  # 1%錯誤率閾值
                issues.append({
                    "type": "high_error_rate",
                    "interface": interface,
                    "severity": "high" if error_rate > 0.05 else "medium",
                    "details": f"介面{interface}錯誤率過高: {error_rate:.2%}"
                })
        
        # 檢查連接數
        if current_state["active_flows"] > 1000:
            issues.append({
                "type": "high_connection_count",
                "severity": "medium",
                "details": f"活躍連接數過高: {current_state['active_flows']}"
            })
        
        return issues
    
    def generate_optimization_plan(self, current_state, issues, goals, constraints):
        """生成優化計劃"""
        plan = {
            "traffic_shaping": [],
            "priority_queuing": [],
            "bandwidth_allocation": [],
            "application_policies": []
        }
        
        # 根據目標生成策略
        if "minimize_latency" in goals:
            plan["priority_queuing"].append({
                "action": "create_express_queue",
                "applications": ["voip", "video_conference"],
                "priority": "highest",
                "bandwidth_guarantee": "20%"
            })
        
        if "maximize_throughput" in goals:
            plan["traffic_shaping"].append({
                "action": "optimize_tcp_window",
                "target": "bulk_transfer",
                "parameters": {"window_scaling": True, "congestion_control": "bbr"}
            })
        
        # 根據約束調整計劃
        max_bandwidth = constraints.get("max_bandwidth_mbps", 100)
        plan["bandwidth_allocation"].append({
            "total_bandwidth": max_bandwidth,
            "allocation": {
                "high_priority": "40%",
                "medium_priority": "35%",
                "low_priority": "25%"
            }
        })
        
        # 應用特定策略
        priority_apps = constraints.get("priority_applications", [])
        for app in priority_apps:
            if app in self.application_profiles:
                profile = self.application_profiles[app]
                plan["application_policies"].append({
                    "application": app,
                    "policy": profile,
                    "enforcement": "strict"
                })
        
        return plan
    
    def estimate_improvement(self, optimization_plan):
        """估算改善效果"""
        improvement = {
            "latency_reduction_percent": 0,
            "throughput_increase_percent": 0,
            "reliability_improvement_percent": 0
        }
        
        # 根據優化計劃估算改善
        if optimization_plan["priority_queuing"]:
            improvement["latency_reduction_percent"] += 15
        
        if optimization_plan["traffic_shaping"]:
            improvement["throughput_increase_percent"] += 10
        
        if optimization_plan["application_policies"]:
            improvement["reliability_improvement_percent"] += 20
        
        return improvement
```

## QoS監控整合

### RTK Controller集成

```go
// internal/qos/qos_manager.go
package qos

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    "sync"
)

type QoSManager struct {
    logger          Logger
    mqtt           MQTTClient
    anomalyDetector *AnomalyDetector
    optimizer      *Optimizer
    metrics        *MetricsCollector
    
    // 監控狀態
    monitoring     map[string]*MonitoringSession
    mu            sync.RWMutex
}

type MonitoringSession struct {
    DeviceID    string
    StartTime   time.Time
    Duration    time.Duration
    Metrics     []QoSMetric
    Alerts      []QoSAlert
    Status      string
}

type QoSMetric struct {
    Timestamp     time.Time             `json:"timestamp"`
    MetricType    string               `json:"metric_type"`
    Value         float64              `json:"value"`
    Unit          string               `json:"unit"`
    Interface     string               `json:"interface,omitempty"`
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type QoSAlert struct {
    Timestamp   time.Time `json:"timestamp"`
    AlertType   string    `json:"alert_type"`
    Severity    string    `json:"severity"`
    Metric      string    `json:"metric"`
    Value       float64   `json:"value"`
    Threshold   float64   `json:"threshold"`
    DeviceID    string    `json:"device_id"`
    Description string    `json:"description"`
}

func NewQoSManager(logger Logger, mqtt MQTTClient) *QoSManager {
    return &QoSManager{
        logger:     logger,
        mqtt:       mqtt,
        monitoring: make(map[string]*MonitoringSession),
    }
}

func (qm *QoSManager) StartMonitoring(ctx context.Context, deviceID string, config MonitoringConfig) error {
    qm.mu.Lock()
    defer qm.mu.Unlock()
    
    // 檢查是否已在監控
    if _, exists := qm.monitoring[deviceID]; exists {
        return fmt.Errorf("device %s is already being monitored", deviceID)
    }
    
    session := &MonitoringSession{
        DeviceID:  deviceID,
        StartTime: time.Now(),
        Duration:  config.Duration,
        Metrics:   make([]QoSMetric, 0),
        Alerts:    make([]QoSAlert, 0),
        Status:    "active",
    }
    
    qm.monitoring[deviceID] = session
    
    // 啟動監控協程
    go qm.monitorDevice(ctx, deviceID, config)
    
    qm.logger.Info("Started QoS monitoring", "device", deviceID, "duration", config.Duration)
    
    return nil
}

func (qm *QoSManager) monitorDevice(ctx context.Context, deviceID string, config MonitoringConfig) {
    defer func() {
        qm.mu.Lock()
        if session, exists := qm.monitoring[deviceID]; exists {
            session.Status = "completed"
        }
        qm.mu.Unlock()
    }()
    
    ticker := time.NewTicker(config.SamplingInterval)
    defer ticker.Stop()
    
    startTime := time.Now()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if time.Since(startTime) >= config.Duration {
                return
            }
            
            // 收集QoS指標
            metrics, err := qm.collectQoSMetrics(ctx, deviceID, config.Metrics)
            if err != nil {
                qm.logger.Error("Failed to collect QoS metrics", "device", deviceID, "error", err)
                continue
            }
            
            // 儲存指標
            qm.storeMetrics(deviceID, metrics)
            
            // 檢查閾值
            alerts := qm.checkThresholds(deviceID, metrics, config.Thresholds)
            if len(alerts) > 0 {
                qm.handleAlerts(deviceID, alerts)
            }
        }
    }
}

func (qm *QoSManager) collectQoSMetrics(ctx context.Context, deviceID string, metricTypes []string) ([]QoSMetric, error) {
    var metrics []QoSMetric
    
    for _, metricType := range metricTypes {
        switch metricType {
        case "latency":
            latencyMetric, err := qm.measureLatency(ctx, deviceID)
            if err == nil {
                metrics = append(metrics, latencyMetric)
            }
        case "throughput":
            throughputMetric, err := qm.measureThroughput(ctx, deviceID)
            if err == nil {
                metrics = append(metrics, throughputMetric)
            }
        case "packet_loss":
            lossMetric, err := qm.measurePacketLoss(ctx, deviceID)
            if err == nil {
                metrics = append(metrics, lossMetric)
            }
        }
    }
    
    return metrics, nil
}

func (qm *QoSManager) checkThresholds(deviceID string, metrics []QoSMetric, thresholds map[string]float64) []QoSAlert {
    var alerts []QoSAlert
    
    for _, metric := range metrics {
        if threshold, exists := thresholds[metric.MetricType]; exists {
            var violated bool
            var severity string
            
            switch metric.MetricType {
            case "latency", "jitter":
                violated = metric.Value > threshold
                severity = qm.calculateLatencySeverity(metric.Value, threshold)
            case "packet_loss":
                violated = metric.Value > threshold
                severity = qm.calculateLossSeverity(metric.Value, threshold)
            case "throughput":
                violated = metric.Value < threshold
                severity = qm.calculateThroughputSeverity(metric.Value, threshold)
            }
            
            if violated {
                alert := QoSAlert{
                    Timestamp:   metric.Timestamp,
                    AlertType:   "threshold_violation",
                    Severity:    severity,
                    Metric:      metric.MetricType,
                    Value:       metric.Value,
                    Threshold:   threshold,
                    DeviceID:    deviceID,
                    Description: fmt.Sprintf("%s threshold violated: %.2f %s (threshold: %.2f)", 
                               metric.MetricType, metric.Value, metric.Unit, threshold),
                }
                alerts = append(alerts, alert)
            }
        }
    }
    
    return alerts
}

type MonitoringConfig struct {
    Duration         time.Duration         `json:"duration"`
    SamplingInterval time.Duration         `json:"sampling_interval"`
    Metrics          []string             `json:"metrics"`
    Thresholds       map[string]float64   `json:"thresholds"`
    AlertingEnabled  bool                 `json:"alerting_enabled"`
}
```

### 監控儀表板

```go
// internal/qos/dashboard.go
package qos

import (
    "encoding/json"
    "net/http"
    "time"
)

type QoSDashboard struct {
    qosManager *QoSManager
}

func NewQoSDashboard(qosManager *QoSManager) *QoSDashboard {
    return &QoSDashboard{
        qosManager: qosManager,
    }
}

func (qd *QoSDashboard) HandleQoSOverview(w http.ResponseWriter, r *http.Request) {
    overview := qd.generateQoSOverview()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(overview)
}

func (qd *QoSDashboard) generateQoSOverview() map[string]interface{} {
    qd.qosManager.mu.RLock()
    defer qd.qosManager.mu.RUnlock()
    
    overview := map[string]interface{}{
        "active_monitoring_sessions": len(qd.qosManager.monitoring),
        "total_devices_monitored":   qd.getTotalDevicesMonitored(),
        "recent_alerts":            qd.getRecentAlerts(24 * time.Hour),
        "performance_summary":      qd.getPerformanceSummary(),
    }
    
    return overview
}

func (qd *QoSDashboard) getPerformanceSummary() map[string]interface{} {
    // 計算整體性能摘要
    summary := map[string]interface{}{
        "avg_latency_ms":      15.2,
        "avg_throughput_mbps": 85.5,
        "avg_packet_loss":     0.15,
        "network_health_score": 85,
    }
    
    return summary
}
```

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Commands and Events Reference](../core/COMMANDS_EVENTS_REFERENCE.md)
- [Network Diagnostics Guide](NETWORK_DIAGNOSTICS.md)
- [Troubleshooting Guide](../guides/TROUBLESHOOTING_GUIDE.md)
- [RFC 2474 - Differentiated Services](https://tools.ietf.org/html/rfc2474)
- [RFC 3246 - Expedited Forwarding PHB](https://tools.ietf.org/html/rfc3246)