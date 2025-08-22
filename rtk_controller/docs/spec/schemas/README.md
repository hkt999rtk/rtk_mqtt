# RTK MQTT Protocol JSON Schemas

本目錄包含 RTK MQTT 協議的完整 JSON Schema 定義，用於消息驗證和開發參考。

## Schema 組織結構

### 基礎 Schema

#### base.json
所有 RTK MQTT 消息的基礎 schema，定義通用欄位結構：
- `schema`: 消息類型標識
- `ts`: Unix 時間戳
- `device_id`: 設備標識符
- `payload`: 業務資料包裝
- `trace`: 可選追蹤資訊
- `meta`: 可選元資料

#### state.json
設備狀態消息的 schema，包含：
- 設備健康狀態
- 運行時間
- 連接狀態
- 設備能力

#### attr.json
設備屬性消息的 schema，包含：
- 設備類型
- 製造商信息
- 硬體/韌體版本
- 設備能力清單

### 命令 Schema

#### cmd-request.json
命令請求的通用 schema，定義：
- 命令 ID 和操作名稱
- 命令參數結構
- 超時和回應設定

#### cmd-ack.json
命令確認消息的 schema

#### cmd-result.json
命令執行結果的通用 schema

#### cmd-error.json
命令錯誤回應的 schema

### 具體命令 Schema

#### cmd-wifi-scan.json
WiFi 頻道掃描命令：
- 掃描類型（主動/被動）
- 掃描持續時間
- 指定頻道
- 是否包含隱藏網絡

#### cmd-speed-test.json
網絡速度測試命令：
- 測試服務器選擇
- 測試持續時間
- 測試方向（下載/上傳/雙向）
- 測試檔案大小

#### cmd-wan-connectivity.json
WAN 連線測試命令：
- 測試目標主機列表
- 超時設定
- Ping 測試次數
- 路由追蹤選項

### 遙測 Schema

#### telemetry.json
遙測消息的基礎 schema

#### telemetry-cpu.json
CPU 遙測資料：
- CPU 使用率
- 負載平均值
- CPU 溫度
- 頻率和核心數

#### telemetry-network.json
網絡介面遙測資料：
- 接收/傳送字節數和封包數
- 錯誤統計
- 連接速度和狀態

#### telemetry-environment.json
環境感測器遙測資料：
- 溫濕度
- 氣壓
- 空氣品質
- 光照度和紫外線
- 運動偵測
- 噪音水平

### 事件 Schema

#### event.json
事件消息的基礎 schema

#### evt-wifi-connection-lost.json
WiFi 連接丟失事件：
- 客戶端 MAC 地址
- 網絡名稱和斷線原因
- 連接持續時間
- 信號強度資訊

#### evt-threshold.json
閾值超限事件：
- 感測器類型和當前值
- 閾值設定
- 超限持續時間

## 使用方式

### 1. 消息驗證
使用適當的 schema 文件驗證 MQTT 消息格式：

```bash
# 使用 ajv-cli 驗證消息
ajv validate -s base.json -d message.json
```

### 2. 程式碼生成
可以使用這些 schema 生成對應的程式碼結構：

#### Go 語言結構產生

```bash
# 基本 Go 結構生成
quicktype --src-lang schema --lang go --out types.go base.json

# 生成帶有 JSON tags 的 Go 結構
quicktype --src-lang schema --lang go --just-types --package-name types --out rtk_types.go base.json

# 為特定消息類型生成結構
quicktype --src-lang schema --lang go --top-level StateMessage --out state.go state.json
quicktype --src-lang schema --lang go --top-level TelemetryMessage --out telemetry.go telemetry-*.json
quicktype --src-lang schema --lang go --top-level CommandRequest --out commands.go cmd-*.json
```

**生成的 Go 結構範例**:
```go
// 自動生成的 RTK MQTT 基礎結構
package types

import "encoding/json"

type RTKMessage struct {
    Schema   string                 `json:"schema"`
    Ts       int64                  `json:"ts"`
    DeviceID string                 `json:"device_id"`
    Payload  map[string]interface{} `json:"payload"`
    Trace    *TraceInfo             `json:"trace,omitempty"`
    Meta     map[string]interface{} `json:"meta,omitempty"`
}

type TraceInfo struct {
    ReqID         *string `json:"req_id,omitempty"`
    CorrelationID *string `json:"correlation_id,omitempty"`
    SessionID     *string `json:"session_id,omitempty"`
    TraceID       *string `json:"trace_id,omitempty"`
}

type StatePayload struct {
    Health           string  `json:"health"`
    UptimeS         int64   `json:"uptime_s"`
    ConnectionStatus string  `json:"connection_status"`
    CPUUsage        float64 `json:"cpu_usage,omitempty"`
    MemoryUsage     float64 `json:"memory_usage,omitempty"`
}
```

#### C/C++ 結構產生

```bash
# 生成 C++ 結構（使用 nlohmann/json）
quicktype --src-lang schema --lang cpp --namespace RTK --out rtk_types.hpp base.json

# 生成 C 結構（簡化版本）
quicktype --src-lang schema --lang c --out rtk_types.h base.json

# 為嵌入式系統生成緊湊的 C 結構
quicktype --src-lang schema --lang c --density compact --out rtk_embedded.h base.json state.json telemetry-*.json
```

**生成的 C++ 結構範例**:
```cpp
// 自動生成的 RTK MQTT C++ 結構
#pragma once

#include <nlohmann/json.hpp>
#include <optional>
#include <string>
#include <vector>
#include <map>

namespace RTK {
    using nlohmann::json;

    struct TraceInfo {
        std::optional<std::string> req_id;
        std::optional<std::string> correlation_id;
        std::optional<std::string> session_id;
        std::optional<std::string> trace_id;
    };

    struct RTKMessage {
        std::string schema;
        int64_t ts;
        std::string device_id;
        json payload;
        std::optional<TraceInfo> trace;
        std::optional<json> meta;
    };

    struct StatePayload {
        std::string health;
        int64_t uptime_s;
        std::string connection_status;
        std::optional<double> cpu_usage;
        std::optional<double> memory_usage;
    };

    // JSON 序列化/反序列化函數
    void to_json(json& j, const RTKMessage& msg);
    void from_json(const json& j, RTKMessage& msg);
}
```

**生成的 C 結構範例**:
```c
// 自動生成的 RTK MQTT C 結構
#ifndef RTK_TYPES_H
#define RTK_TYPES_H

#include <stdint.h>
#include <stdbool.h>

// 設備狀態枚舉
typedef enum {
    RTK_HEALTH_OK = 0,
    RTK_HEALTH_WARNING = 1,
    RTK_HEALTH_ERROR = 2,
    RTK_HEALTH_CRITICAL = 3
} rtk_health_t;

// 連接狀態枚舉
typedef enum {
    RTK_CONN_DISCONNECTED = 0,
    RTK_CONN_CONNECTING = 1,
    RTK_CONN_CONNECTED = 2,
    RTK_CONN_ERROR = 3
} rtk_connection_status_t;

// 追蹤資訊結構
typedef struct {
    char req_id[64];
    char correlation_id[64];
    char session_id[128];
    char trace_id[128];
    bool has_req_id;
    bool has_correlation_id;
    bool has_session_id;
    bool has_trace_id;
} rtk_trace_info_t;

// RTK 消息基礎結構
typedef struct {
    char schema[64];
    int64_t ts;
    char device_id[32];
    rtk_trace_info_t trace;
    bool has_trace;
} rtk_message_base_t;

// 設備狀態載荷
typedef struct {
    rtk_health_t health;
    int64_t uptime_s;
    rtk_connection_status_t connection_status;
    double cpu_usage;
    double memory_usage;
    bool has_cpu_usage;
    bool has_memory_usage;
} rtk_state_payload_t;

// 完整狀態消息
typedef struct {
    rtk_message_base_t base;
    rtk_state_payload_t payload;
} rtk_state_message_t;

// 網絡介面統計
typedef struct {
    char name[32];
    uint64_t rx_bytes;
    uint64_t tx_bytes;
    uint64_t rx_packets;
    uint64_t tx_packets;
    uint32_t rx_errors;
    uint32_t tx_errors;
    uint32_t speed_mbps;
    bool interface_up;
} rtk_network_interface_t;

// 網絡遙測載荷
typedef struct {
    rtk_network_interface_t interfaces[8];  // 最多8個網絡介面
    uint8_t interface_count;
} rtk_network_telemetry_payload_t;

#endif // RTK_TYPES_H
```

#### 批量生成腳本

```bash
#!/bin/bash
# generate_all_types.sh - 批量生成所有語言的類型定義

echo "Generating Go types..."
mkdir -p generated/go
quicktype --src-lang schema --lang go --package-name rtktypes --out generated/go/types.go *.json

echo "Generating C++ types..."
mkdir -p generated/cpp
quicktype --src-lang schema --lang cpp --namespace RTK --out generated/cpp/rtk_types.hpp *.json

echo "Generating C types..."
mkdir -p generated/c
quicktype --src-lang schema --lang c --out generated/c/rtk_types.h *.json

echo "Generating TypeScript types..."
mkdir -p generated/typescript
quicktype --src-lang schema --lang typescript --out generated/typescript/types.ts *.json

echo "Generating Python types..."
mkdir -p generated/python
quicktype --src-lang schema --lang python --out generated/python/rtk_types.py *.json

echo "Type generation complete!"
```

#### 嵌入式系統特化版本

對於記憶體受限的嵌入式系統，可以生成更緊湊的 C 結構：

```bash
# 生成嵌入式 C 結構（記憶體優化）
quicktype --src-lang schema --lang c \
  --density compact \
  --no-combine-classes \
  --acronym-style original \
  --out rtk_embedded.h \
  base.json state.json telemetry-cpu.json telemetry-network.json
```

**嵌入式 C 結構範例**:
```c
// 適用於嵌入式系統的緊湊結構
#ifndef RTK_EMBEDDED_H
#define RTK_EMBEDDED_H

#include <stdint.h>
#include <stdbool.h>

// 緊湊的消息頭部（24 bytes）
typedef struct __attribute__((packed)) {
    uint32_t schema_hash;      // Schema 的 CRC32
    uint64_t ts;               // 時間戳
    uint32_t device_id_hash;   // Device ID 的 CRC32
    uint32_t payload_size;     // Payload 大小
} rtk_header_compact_t;

// 緊湊的狀態消息（16 bytes）
typedef struct __attribute__((packed)) {
    uint8_t health;            // 0=ok, 1=warning, 2=error, 3=critical
    uint8_t connection_status; // 0=disconnected, 1=connecting, 2=connected
    uint32_t uptime_s;         // 運行時間
    uint16_t cpu_usage_x100;   // CPU 使用率 * 100 (0-10000)
    uint16_t memory_usage_x100;// 記憶體使用率 * 100
    uint32_t reserved;         // 保留欄位
} rtk_state_compact_t;

// 緊湑的網絡統計（32 bytes per interface）
typedef struct __attribute__((packed)) {
    uint32_t rx_bytes_kb;      // 接收 KB 數
    uint32_t tx_bytes_kb;      // 傳送 KB 數
    uint16_t rx_packets_k;     // 接收封包數 (千)
    uint16_t tx_packets_k;     // 傳送封包數 (千)
    uint16_t rx_errors;        // 接收錯誤數
    uint16_t tx_errors;        // 傳送錯誤數
    uint16_t speed_mbps;       // 連接速度
    uint8_t status;            // 介面狀態
    uint8_t interface_id;      // 介面編號
    uint32_t reserved;         // 保留欄位
} rtk_network_if_compact_t;

// 消息總大小控制在 256 bytes 內
#define RTK_MAX_NETWORK_INTERFACES 6
typedef struct __attribute__((packed)) {
    rtk_header_compact_t header;
    union {
        rtk_state_compact_t state;
        struct {
            uint8_t interface_count;
            rtk_network_if_compact_t interfaces[RTK_MAX_NETWORK_INTERFACES];
        } network;
        uint8_t raw_payload[224]; // 256 - 32 (header) = 224
    } payload;
} rtk_message_compact_t;

// 編碼/解碼函數聲明
int rtk_encode_state_compact(const rtk_state_compact_t* state, uint8_t* buffer, size_t buffer_size);
int rtk_decode_state_compact(const uint8_t* buffer, size_t buffer_size, rtk_state_compact_t* state);
uint32_t rtk_calculate_schema_hash(const char* schema_string);

#endif // RTK_EMBEDDED_H
```

#### Arduino 專用結構

```c
// rtk_arduino.h - Arduino 平台專用
#ifndef RTK_ARDUINO_H
#define RTK_ARDUINO_H

#include <Arduino.h>

// 使用 Arduino String 類別
struct RTKArduinoMessage {
    String schema;
    unsigned long ts;
    String device_id;
    String payload_json;  // 儲存為 JSON 字串以節省記憶體
};

// 簡化的狀態結構
struct RTKStateArduino {
    uint8_t health;        // 0-3
    uint32_t uptime_s;
    uint8_t connection_status;
    float cpu_usage;       // 0.0-100.0
    float memory_usage;    // 0.0-100.0
};

// WiFi 掃描結果
struct RTKWiFiScanResult {
    String ssid;
    String bssid;
    int32_t rssi;
    uint8_t channel;
    uint8_t encryption_type;
};

// 輔助函數
String rtk_state_to_json(const RTKStateArduino& state);
bool rtk_parse_state_json(const String& json, RTKStateArduino& state);
void rtk_publish_state(const RTKStateArduino& state);

#endif // RTK_ARDUINO_H
```

### 3. 文檔生成
使用 schema 自動生成 API 文檔：

```bash
# 使用 json-schema-to-markdown
json-schema-to-markdown base.json > base.md
```

## Schema 版本管理

- **版本格式**: 使用 semantic versioning（major.minor）
- **向後相容**: minor 版本變更應保持向後相容
- **破壞性變更**: major 版本變更可包含不相容變更

## 工具安裝和環境設置

### 安裝 quicktype

```bash
# 使用 npm 安裝
npm install -g quicktype

# 使用 yarn 安裝
yarn global add quicktype

# 驗證安裝
quicktype --help
```

### 安裝其他相關工具

```bash
# JSON Schema 驗證工具
npm install -g ajv-cli

# Markdown 文檔生成工具
npm install -g json-schema-to-markdown

# C/C++ JSON 處理庫
# Ubuntu/Debian
sudo apt-get install nlohmann-json3-dev libcjson-dev

# macOS
brew install nlohmann-json cjson

# 嵌入式系統 JSON 庫
git clone https://github.com/DaveGamble/cJSON.git
```

### 開發工作流程範例

```bash
#!/bin/bash
# 完整的開發工作流程

# 1. 驗證所有 Schema 文件
echo "Validating schemas..."
for schema in *.json; do
    echo "Checking $schema..."
    ajv compile -s "$schema" || exit 1
done

# 2. 生成程式碼
echo "Generating code types..."
./generate_all_types.sh

# 3. 驗證生成的程式碼
echo "Validating generated Go code..."
cd generated/go && go build . && cd ../..

echo "Validating generated C++ code..."
cd generated/cpp && g++ -std=c++17 -c rtk_types.hpp && cd ../..

# 4. 生成文檔
echo "Generating documentation..."
for schema in *.json; do
    name=$(basename "$schema" .json)
    json-schema-to-markdown "$schema" > "../docs/${name}.md"
done

echo "Build complete!"
```

### 整合到 CI/CD

**GitHub Actions 範例**:
```yaml
name: Schema Validation and Code Generation

on: [push, pull_request]

jobs:
  validate-and-generate:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'
        
    - name: Install tools
      run: |
        npm install -g quicktype ajv-cli json-schema-to-markdown
        
    - name: Validate schemas
      run: |
        cd docs/spec/schemas
        for schema in *.json; do
          ajv compile -s "$schema"
        done
        
    - name: Generate types
      run: |
        cd docs/spec/schemas
        ./generate_all_types.sh
        
    - name: Test generated code
      run: |
        cd docs/spec/schemas/generated/go
        go mod init rtk-types && go build .
```

## 相關文檔

- [Message Format Specification](../06-message-format.md)
- [Schema Reference](../../developers/core/SCHEMA_REFERENCE.md)
- [Commands and Events Reference](../../developers/core/COMMANDS_EVENTS_REFERENCE.md)
- [quicktype 官方文檔](https://quicktype.io/)
- [JSON Schema 規範](https://json-schema.org/)