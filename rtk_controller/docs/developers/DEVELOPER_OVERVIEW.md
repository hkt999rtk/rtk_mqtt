# RTK MQTT 開發者總覽

## 🎯 歡迎使用 RTK MQTT

RTK MQTT 是一個專為 IoT 設備診斷和網路管理設計的強大通信協議，基於 MQTT 3.1.1 標準，整合 AI 驅動的自動化診斷功能。

### 🚀 為什麼選擇 RTK MQTT？

- **✅ 標準化**: 基於成熟的 MQTT 協議，確保互操作性
- **🔧 易整合**: 支援多種設備類型和程式語言
- **🤖 AI 驅動**: 內建 LLM 整合，提供智慧診斷能力
- **📊 全面監控**: 網路拓撲、效能診斷、QoS 管理一站式解決
- **🔒 安全可靠**: 支援 TLS/SSL 加密和多租戶隔離

## 📚 學習路徑

### 🟢 入門級 (0-30 分鐘)
1. **[快速開始](guides/QUICK_START_GUIDE.md)** - 15 分鐘完成第一個設備整合
2. **[MQTT 協議規格](core/MQTT_PROTOCOL_SPEC.md)** - 理解基本概念和訊息格式
3. **[主題結構指南](core/TOPIC_STRUCTURE.md)** - 掌握 MQTT 主題命名規範

### 🟡 進階級 (30-120 分鐘)
4. **[整合指南](INTEGRATION_GUIDE.md)** - 深入了解通用整合模式
5. **[設備專用指南](devices/)** - 選擇你的設備類型深入學習
6. **[測試整合](guides/TESTING_INTEGRATION.md)** - 建立完整的測試流程

### 🔴 專家級 (2+ 小時)
7. **[進階功能](guides/ADVANCED_FEATURES.md)** - 自定義命令和複雜事件處理
8. **[診斷工具](diagnostics/)** - 網路診斷和 QoS 監控
9. **[部署指南](guides/DEPLOYMENT_GUIDE.md)** - 生產環境部署

## 🏗️ 架構概覽

### 系統組件
```
┌─────────────────────┐    MQTT     ┌─────────────────────┐
│   IoT Devices       │ ◄─────────► │   RTK Controller    │
│                     │             │                     │
│ • AP/Router         │   Messages  │ • 拓撲管理          │
│ • Switch            │   Commands  │ • 診斷分析          │
│ • NIC/Sensors       │   Events    │ • LLM 整合          │
│ • Smart Devices     │             │ • QoS 管理          │
└─────────────────────┘             └─────────────────────┘
                │                                │
                └─────────── MQTT Broker ──────────┘
```

### 訊息流向
```
設備感測 → 資料收集 → MQTT 發布 → Controller 接收 
    ↑                                        ↓
設備執行 ← 命令下發 ← 決策制定 ← 資料分析
```

### 主題層級結構
```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}
   │    │     │      │        │           │
   │    │     │      │        │           └─ state/telemetry/evt/cmd/attr
   │    │     │      │        └─ 設備唯一標識 (MAC地址)
   │    │     │      └─ 場域識別 (如 floor1, building_a)
   │    │     └─ 租戶識別 (組織或用戶群組)
   │    └─ 協議版本
   └─ 協議命名空間
```

## 🔧 支援的設備類型

### 🌐 網路設備
- **[AP/Router](devices/AP_ROUTER_INTEGRATION.md)**: WiFi 管理、客戶端追蹤、拓撲發現
- **[Switch](devices/SWITCH_INTEGRATION.md)**: 埠管理、VLAN 配置、流量統計
- **[Mesh Node](devices/MESH_NODE_INTEGRATION.md)**: Mesh 拓撲、回程監控、負載平衡

### 💻 終端設備
- **[NIC](devices/NIC_INTEGRATION.md)**: 網卡狀態、速度測試、電源管理
- **[IoT 設備](devices/IOT_DEVICE_INTEGRATION.md)**: 感測器、智慧家電、工業設備

## 📋 核心訊息類型

### 上行訊息 (Device → Controller)
| 類型 | 用途 | 頻率 | 保留 |
|------|------|------|------|
| `state` | 設備健康狀態 | 每5分鐘 | ✅ |
| `telemetry/{metric}` | 效能指標 | 每30秒 | ❌ |
| `evt/{event_type}` | 事件通知 | 即時 | ❌ |
| `attr` | 設備屬性 | 啟動/變更時 | ✅ |
| `topology/update` | 拓撲變化 | 變更時 | ❌ |

### 下行命令 (Controller → Device)
| 類型 | 用途 | 流程 |
|------|------|------|
| `cmd/req` | 命令請求 | Controller → Device |
| `cmd/ack` | 命令確認 | Device → Controller |
| `cmd/res` | 命令結果 | Device → Controller |

## 🛠️ 開發工具

### 必要工具
- **MQTT Broker**: Mosquitto 或 RTK 專用 Broker
- **測試工具**: [MQTT 測試框架](tools/MQTT_TESTING_TOOLS.md)
- **CLI 工具**: [命令行介面](tools/CLI_TOOLS.md)
- **Schema 驗證**: JSON Schema 驗證器

### SDK 支援
- **C/C++**: 適用於嵌入式設備
- **Python**: 快速原型和測試
- **Go**: 高效能服務
- **JavaScript/Node.js**: Web 整合

詳見 **[SDK 參考指南](tools/SDK_REFERENCE.md)**

## 🔍 主要功能

### 🌐 網路診斷
- **速度測試**: 上下行頻寬測量
- **WAN 診斷**: DNS、路由、外部連通性檢測
- **延遲分析**: RTT、抖動、封包遺失率
- **WiFi 診斷**: 頻道掃描、干擾分析、訊號強度映射

### 🔄 拓撲管理
- **自動發現**: ARP、鄰居檢測、路由分析
- **即時追蹤**: 設備連線狀態變化
- **視覺化**: 網路拓撲圖形化顯示
- **歷史記錄**: 連線變化歷史追蹤

### ⚡ QoS 管理
- **流量分析**: 頻寬使用、應用分類
- **策略執行**: 動態 QoS 規則套用
- **異常檢測**: 流量異常和熱點識別
- **最佳化建議**: AI 驅動的效能建議

### 🤖 LLM 整合
- **智慧診斷**: 自動問題識別和解決方案
- **自然語言**: 語音/文字命令執行
- **工具整合**: Read/Test/Act 三類診斷工具
- **會話追蹤**: 多輪對話診斷流程

## ⚡ 快速開始檢查清單

### 📋 5 分鐘設置
- [ ] 安裝 MQTT Broker (Mosquitto)
- [ ] 下載 RTK MQTT SDK
- [ ] 配置基本 MQTT 連線
- [ ] 發送第一個 `state` 訊息

### 📋 15 分鐘整合
- [ ] 實作設備屬性 (`attr`) 發布
- [ ] 設置 LWT (Last Will Testament)
- [ ] 訂閱命令主題 (`cmd/req`)
- [ ] 實作基本命令處理器

### 📋 30 分鐘進階
- [ ] 新增 telemetry 數據發送
- [ ] 實作事件通知機制
- [ ] 設置錯誤處理和重連
- [ ] 整合診斷工具

## 🆘 需要幫助？

### 📖 文檔資源
- **[故障排除指南](guides/TROUBLESHOOTING_GUIDE.md)** - 常見問題解決
- **[API 完整參考](core/MQTT_API_REFERENCE.md)** - 所有命令和事件
- **[Schema 參考](core/SCHEMA_REFERENCE.md)** - JSON 格式規範

### 🤝 社群支援
- **[支援資源](release/SUPPORT_RESOURCES.md)** - 技術支援聯絡方式
- **[GitHub Issues](https://github.com/rtk-mqtt/issues)** - 問題回報和功能請求
- **[開發者論壇](https://forum.rtk-mqtt.com)** - 社群討論

### 🏆 認證程序
完成開發後，可申請 **[RTK 認證](release/CERTIFICATION_PROCESS.md)** 以確保相容性和品質。

---

## 🎯 下一步

選擇適合你的路徑開始 RTK MQTT 之旅：

1. **🚀 快速體驗**: [15分鐘快速開始](guides/QUICK_START_GUIDE.md)
2. **🔧 設備整合**: 選擇你的[設備類型指南](devices/)
3. **📚 深入學習**: [完整協議規格](core/MQTT_PROTOCOL_SPEC.md)
4. **🛠️ 工具開發**: [SDK 和工具指南](tools/)

歡迎加入 RTK MQTT 開發者社群，一起構建更智慧的 IoT 生態系統！