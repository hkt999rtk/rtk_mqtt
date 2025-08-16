# MQTT WiFi 診斷通訊流程與時序圖

## 1. 通用診斷流程時序圖

![MQTT Message Flow Diagram](message_flow.png)

**詳細序列圖**：
![Generic MQTT Sequence](generic_sequence_simple.png)

## 2. 漫遊診斷完整流程

![WiFi Roaming Diagnosis Sequence](roaming_sequence.png)

**詳細序列圖**：
![Roaming Sequence](roaming_sequence_simple.png)

## 3. 連線失敗診斷流程

![Connection Failure Diagnosis Sequence](connection_failure_diagnosis_sequence.png)

## 4. ARP 遺失診斷流程

![ARP Loss Diagnosis Sequence](arp_loss_diagnosis_sequence.png)

## 5. 訂閱模式與 Topic 路由

### 5.1 Controller 訂閱策略

```
全域監控:
├── rtk/v1/+/+/+/evt/#           # 所有事件
├── rtk/v1/+/+/+/lwt             # 設備上下線
└── rtk/v1/+/+/+/state           # 設備狀態

特定場域監控:
├── rtk/v1/office/+/+/evt/#      # 辦公室事件
├── rtk/v1/factory/+/+/evt/wifi.# # 工廠 WiFi 事件
└── rtk/v1/corporate/+/+/cmd/ack # 企業命令確認

設備類型監控:
├── rtk/v1/+/+/smart-camera-+/evt/#    # 智能攝影機
├── rtk/v1/+/+/laptop-+/evt/wifi.#     # 筆記型電腦 WiFi
└── rtk/v1/+/+/office-ap-+/telemetry/# # 辦公室 AP 遙測
```


## 6. 錯誤處理與恢復流程

### 6.1 命令超時處理

![Command Timeout Sequence](command_timeout_sequence.png)

### 6.2 連線中斷恢復

![Connection Recovery Sequence](connection_recovery_sequence.png)

## 7. 效能最佳化建議

### 7.1 訊息大小控制
- **Event 訊息**: < 1KB (關鍵資訊)
- **Diagnostic 結果**: < 10KB (詳細分析)
- **Telemetry**: < 500B (定期指標)
- **State**: < 2KB (完整狀態)

### 7.2 頻率控制
- **Events**: 立即發送 + 5分鐘冷卻期
- **Diagnostics**: 按需請求，最大 1 次/分鐘
- **Telemetry**: 10-60 秒間隔
- **State**: 30-60 秒或狀態變更時

### 7.3 批次處理
```json
{
  "id": "batch-diag-001",
  "op": "diagnosis.batch_get",
  "args": {
    "types": ["wifi.roaming", "wifi.interference", "wifi.performance"],
    "correlation_id": "network_issue_001"
  }
}
```