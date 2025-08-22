# RTK MQTT JSON Schema 參考

## 概述

本文檔定義RTK MQTT協議中所有消息類型的JSON Schema，用於消息驗證和開發參考。

## 通用Schema結構

### 基礎消息結構
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "schema": {
      "type": "string",
      "pattern": "^[a-z_]+\\.[a-z_]+/\\d+\\.\\d+$",
      "description": "消息架構標識"
    },
    "ts": {
      "type": "integer",
      "minimum": 0,
      "description": "Unix時間戳(毫秒)"
    },
    "device_id": {
      "type": "string",
      "minLength": 1,
      "maxLength": 64,
      "description": "設備標識符"
    }
  },
  "required": ["schema", "ts", "device_id"]
}
```

## 狀態消息Schema

### state/1.0
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "schema": {
      "type": "string",
      "enum": ["state/1.0"]
    },
    "ts": {
      "type": "integer",
      "minimum": 0
    },
    "device_id": {
      "type": "string",
      "minLength": 1,
      "maxLength": 64
    },
    "health": {
      "type": "string",
      "enum": ["ok", "warning", "error", "unknown"]
    },
    "uptime_s": {
      "type": "integer",
      "minimum": 0,
      "description": "設備運行時間(秒)"
    },
    "cpu_usage": {
      "type": "number",
      "minimum": 0,
      "maximum": 100,
      "description": "CPU使用率(%)"
    },
    "memory_usage": {
      "type": "number", 
      "minimum": 0,
      "maximum": 100,
      "description": "內存使用率(%)"
    },
    "connection_status": {
      "type": "string",
      "enum": ["connected", "disconnected", "connecting", "error"]
    }
  },
  "required": ["schema", "ts", "device_id", "health"]
}
```

## 命令Schema

### 命令請求通用格式
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "minLength": 1,
      "maxLength": 128,
      "description": "命令唯一標識符"
    },
    "op": {
      "type": "string",
      "minLength": 1,
      "maxLength": 64,
      "description": "操作名稱"
    },
    "schema": {
      "type": "string",
      "pattern": "^cmd\\.[a-z_\\.]+/\\d+\\.\\d+$",
      "description": "命令架構標識"
    },
    "args": {
      "type": "object",
      "description": "命令參數"
    },
    "timeout_ms": {
      "type": "integer",
      "minimum": 1000,
      "maximum": 600000,
      "description": "超時時間(毫秒)"
    },
    "expect": {
      "type": "string",
      "enum": ["ack", "result"],
      "description": "期望響應類型"
    },
    "ts": {
      "type": "integer",
      "minimum": 0
    }
  },
  "required": ["id", "op", "schema", "args", "timeout_ms", "ts"]
}
```

### WiFi掃描命令 (cmd.wifi_scan/1.0)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "op": {"enum": ["wifi_scan"]},
    "schema": {"enum": ["cmd.wifi_scan/1.0"]},
    "args": {
      "type": "object",
      "properties": {
        "scan_type": {
          "type": "string",
          "enum": ["active", "passive"],
          "default": "active"
        },
        "duration": {
          "type": "integer",
          "minimum": 1,
          "maximum": 120,
          "default": 10
        },
        "channels": {
          "type": "array",
          "items": {
            "type": "integer",
            "minimum": 1,
            "maximum": 165
          }
        },
        "include_hidden": {
          "type": "boolean",
          "default": true
        }
      },
      "additionalProperties": false
    },
    "timeout_ms": {"type": "integer"},
    "ts": {"type": "integer"}
  },
  "required": ["id", "op", "schema", "args", "timeout_ms", "ts"]
}
```

### 速度測試命令 (cmd.speed_test/1.0)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "op": {"enum": ["speed_test"]},
    "schema": {"enum": ["cmd.speed_test/1.0"]},
    "args": {
      "type": "object",
      "properties": {
        "server": {
          "type": "string",
          "description": "測試服務器地址或'auto'"
        },
        "duration": {
          "type": "integer",
          "minimum": 10,
          "maximum": 300,
          "default": 30
        },
        "direction": {
          "type": "string",
          "enum": ["download", "upload", "both"],
          "default": "both"
        }
      },
      "additionalProperties": false
    },
    "timeout_ms": {"type": "integer"},
    "ts": {"type": "integer"}
  },
  "required": ["id", "op", "schema", "args", "timeout_ms", "ts"]
}
```

### 系統信息查詢 (cmd.get_system_info/1.0)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "op": {"enum": ["get_system_info"]},
    "schema": {"enum": ["cmd.get_system_info/1.0"]},
    "args": {
      "type": "object",
      "properties": {
        "include": {
          "type": "array",
          "items": {
            "type": "string",
            "enum": ["cpu", "memory", "disk", "network", "all"]
          },
          "default": ["all"]
        },
        "detailed": {
          "type": "boolean",
          "default": false
        }
      },
      "additionalProperties": false
    },
    "timeout_ms": {"type": "integer"},
    "ts": {"type": "integer"}
  },
  "required": ["id", "op", "schema", "args", "timeout_ms", "ts"]
}
```

## 命令響應Schema

### 成功響應格式
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "description": "對應的命令ID"
    },
    "schema": {
      "type": "string",
      "pattern": "^cmd\\.[a-z_\\.]+\\.result/\\d+\\.\\d+$"
    },
    "status": {
      "type": "string",
      "enum": ["completed", "failed", "timeout"]
    },
    "result": {
      "type": "object",
      "description": "命令執行結果"
    },
    "error": {
      "type": ["string", "null"],
      "description": "錯誤信息"
    },
    "ts": {
      "type": "integer",
      "minimum": 0
    }
  },
  "required": ["id", "schema", "status", "ts"]
}
```

### WiFi掃描結果 (cmd.wifi_scan.result/1.0)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "schema": {"enum": ["cmd.wifi_scan.result/1.0"]},
    "status": {"enum": ["completed"]},
    "result": {
      "type": "object",
      "properties": {
        "networks": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "ssid": {"type": "string"},
              "bssid": {
                "type": "string",
                "pattern": "^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$"
              },
              "channel": {
                "type": "integer",
                "minimum": 1,
                "maximum": 165
              },
              "frequency": {
                "type": "integer",
                "minimum": 2400,
                "maximum": 6000
              },
              "signal_strength": {
                "type": "integer",
                "minimum": -100,
                "maximum": 0
              },
              "security": {"type": "string"},
              "bandwidth": {"type": "string"},
              "hidden": {"type": "boolean"}
            },
            "required": ["ssid", "bssid", "channel", "signal_strength"]
          }
        },
        "scan_duration": {"type": "integer"},
        "channels_scanned": {
          "type": "array",
          "items": {"type": "integer"}
        }
      },
      "required": ["networks"]
    },
    "ts": {"type": "integer"}
  },
  "required": ["id", "schema", "status", "result", "ts"]
}
```

## 事件Schema

### 事件通用格式
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "schema": {
      "type": "string",
      "pattern": "^evt\\.[a-z_\\.]+/\\d+\\.\\d+$"
    },
    "ts": {
      "type": "integer",
      "minimum": 0
    },
    "device_id": {
      "type": "string",
      "minLength": 1,
      "maxLength": 64
    },
    "data": {
      "type": "object",
      "description": "事件數據"
    }
  },
  "required": ["schema", "ts", "device_id", "data"]
}
```

### WiFi連接丟失事件 (evt.wifi.connection_lost/1.0)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "schema": {"enum": ["evt.wifi.connection_lost/1.0"]},
    "ts": {"type": "integer"},
    "device_id": {"type": "string"},
    "data": {
      "type": "object",
      "properties": {
        "client_mac": {
          "type": "string",
          "pattern": "^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$"
        },
        "ssid": {"type": "string"},
        "reason": {
          "type": "string",
          "enum": ["signal_weak", "timeout", "user_disconnect", "error"]
        },
        "duration_connected": {
          "type": "integer",
          "minimum": 0,
          "description": "連接持續時間(秒)"
        },
        "signal_strength": {
          "type": "integer",
          "minimum": -100,
          "maximum": 0
        }
      },
      "required": ["client_mac", "ssid", "reason"]
    }
  },
  "required": ["schema", "ts", "device_id", "data"]
}
```

## 遙測數據Schema

### CPU遙測 (telemetry.cpu/1.0)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "schema": {"enum": ["telemetry.cpu/1.0"]},
    "ts": {"type": "integer"},
    "device_id": {"type": "string"},
    "data": {
      "type": "object",
      "properties": {
        "usage_percent": {
          "type": "number",
          "minimum": 0,
          "maximum": 100
        },
        "load_average": {
          "type": "array",
          "items": {"type": "number"},
          "minItems": 3,
          "maxItems": 3,
          "description": "1分鐘、5分鐘、15分鐘負載平均值"
        },
        "temperature": {
          "type": "number",
          "description": "CPU溫度(攝氏度)"
        }
      },
      "required": ["usage_percent"]
    }
  },
  "required": ["schema", "ts", "device_id", "data"]
}
```

### 網絡遙測 (telemetry.network/1.0)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "schema": {"enum": ["telemetry.network/1.0"]},
    "ts": {"type": "integer"},
    "device_id": {"type": "string"},
    "data": {
      "type": "object",
      "properties": {
        "interfaces": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name": {"type": "string"},
              "rx_bytes": {"type": "integer", "minimum": 0},
              "tx_bytes": {"type": "integer", "minimum": 0},
              "rx_packets": {"type": "integer", "minimum": 0},
              "tx_packets": {"type": "integer", "minimum": 0},
              "rx_errors": {"type": "integer", "minimum": 0},
              "tx_errors": {"type": "integer", "minimum": 0},
              "speed_mbps": {"type": "integer", "minimum": 0}
            },
            "required": ["name", "rx_bytes", "tx_bytes"]
          }
        }
      },
      "required": ["interfaces"]
    }
  },
  "required": ["schema", "ts", "device_id", "data"]
}
```

## 錯誤Schema

### 命令錯誤響應 (cmd.error/1.0)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "schema": {"enum": ["cmd.error/1.0"]},
    "status": {"enum": ["failed"]},
    "error": {
      "type": "object",
      "properties": {
        "code": {
          "type": "integer",
          "minimum": 1000,
          "maximum": 9999
        },
        "message": {
          "type": "string",
          "minLength": 1,
          "maxLength": 200
        },
        "details": {
          "type": "string",
          "maxLength": 500,
          "description": "詳細錯誤信息"
        }
      },
      "required": ["code", "message"]
    },
    "ts": {"type": "integer"}
  },
  "required": ["id", "schema", "status", "error", "ts"]
}
```

## 設備屬性Schema

### attr/1.0
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "schema": {"enum": ["attr/1.0"]},
    "ts": {"type": "integer"},
    "device_id": {"type": "string"},
    "device_type": {
      "type": "string",
      "enum": ["router", "switch", "access_point", "gateway", "sensor", "camera", "other"]
    },
    "manufacturer": {"type": "string"},
    "model": {"type": "string"},
    "firmware_version": {"type": "string"},
    "hardware_version": {"type": "string"},
    "serial_number": {"type": "string"},
    "mac_address": {
      "type": "string",
      "pattern": "^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$"
    },
    "capabilities": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": ["wifi", "ethernet", "mesh", "vpn", "qos", "firewall", "dhcp"]
      }
    }
  },
  "required": ["schema", "ts", "device_id", "device_type"]
}
```

## Schema驗證工具

### JavaScript驗證示例
```javascript
const Ajv = require('ajv');
const ajv = new Ajv();

// 載入Schema
const stateSchema = { /* state/1.0 schema */ };
const validate = ajv.compile(stateSchema);

// 驗證消息
const message = {
  "schema": "state/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "health": "ok",
  "uptime_s": 86400
};

const valid = validate(message);
if (!valid) {
  console.log(validate.errors);
}
```

### Python驗證示例
```python
import jsonschema

# 載入Schema
state_schema = { # state/1.0 schema }

# 驗證消息
message = {
    "schema": "state/1.0",
    "ts": 1699123456789,
    "device_id": "aabbccddeeff",
    "health": "ok",
    "uptime_s": 86400
}

try:
    jsonschema.validate(instance=message, schema=state_schema)
    print("消息格式正確")
except jsonschema.exceptions.ValidationError as e:
    print(f"驗證錯誤: {e.message}")
```

## 最佳實踐

### Schema設計原則
1. **向後兼容**: 新版本應該向後兼容
2. **字段可選**: 盡可能使字段為可選，提供默認值
3. **數據類型嚴格**: 明確定義數據類型和範圍
4. **清晰描述**: 為每個字段提供清晰的描述

### 版本管理
- 小版本變更(1.0 -> 1.1): 添加可選字段
- 大版本變更(1.0 -> 2.0): 可能包含不兼容的變更

### 錯誤處理
- 驗證失敗時記錄詳細錯誤信息
- 提供友好的錯誤消息
- 實施寬鬆的解析策略(忽略未知字段)

## 相關文檔

- [MQTT Protocol Specification](MQTT_PROTOCOL_SPEC.md) - 完整協議規範
- [Commands Reference](COMMANDS_EVENTS_REFERENCE.md) - 命令和事件參考
- [Topic Structure](TOPIC_STRUCTURE.md) - 主題結構詳解