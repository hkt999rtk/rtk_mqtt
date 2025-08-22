# RTK MQTT 協議規格文檔

RTK MQTT 是一個專為 IoT 設備診斷和網路管理設計的通信協議，基於 MQTT 3.1.1 標準構建，支援 LLM 自動化診斷功能。

## 📋 文檔結構

| 檔案 | 內容 | 狀態 |
|------|------|------|
| [01-overview.md](01-overview.md) | 協議概述與變更歷史 | ✅ |
| [02-prerequisites.md](02-prerequisites.md) | 前置要求與依賴程式庫 | ✅ |
| [03-terminology.md](03-terminology.md) | 術語定義與設計原則 | ✅ |
| [04-topic-structure.md](04-topic-structure.md) | Topic 命名空間完整說明 | ✅ |
| [05-device-management.md](05-device-management.md) | Device ID 規範與設備管理 | ✅ |
| [06-message-format.md](06-message-format.md) | 共通 Payload 格式與規則 | ✅ |
| [07-mqtt-usage.md](07-mqtt-usage.md) | MQTT 使用時機與最佳實踐 | ✅ |
| [08-uplink-messages.md](08-uplink-messages.md) | 上行訊息結構 (Device → Controller) | ✅ |
| [09-downlink-commands.md](09-downlink-commands.md) | 下行命令結構 (Controller → Device) | ✅ |
| [10-diagnostics-protocol.md](10-diagnostics-protocol.md) | 診斷協議詳細規格 | ✅ |
| [11-examples.md](11-examples.md) | 完整範例與應用場景 | ✅ |
| [12-implementation-guide.md](12-implementation-guide.md) | 實作指南與建議 | ✅ |
| [schemas/](schemas/) | JSON Schema 檔案夾 | ✅ |

## 🚀 快速開始

1. **新手入門**: 從 [協議概述](01-overview.md) 開始了解 RTK MQTT 的基本概念
2. **環境設置**: 參考 [前置要求](02-prerequisites.md) 安裝必要的程式庫
3. **實作開發**: 查看 [實作指南](12-implementation-guide.md) 了解如何實作協議
4. **範例參考**: 瀏覽 [完整範例](11-examples.md) 了解實際應用

## 🏗️ 協議架構

```
┌─────────────────┐    MQTT Topics    ┌─────────────────┐
│   IoT Devices   │ ←──────────────→ │   Controller    │
│                 │                   │                 │
│ • 路由器/AP     │    上行訊息        │ • 拓撲管理      │
│ • 交換機        │    - state        │ • 診斷分析      │
│ • NIC/感測器    │    - telemetry    │ • 命令控制      │
│ • 智慧設備      │    - evt          │ • LLM 整合      │
│                 │    - attr         │                 │
│                 │                   │                 │
│                 │    下行命令        │                 │
│                 │    - cmd/req      │                 │
│                 │    - cmd/ack      │                 │
│                 │    - cmd/res      │                 │
└─────────────────┘                   └─────────────────┘
```

## 📝 主要特色

- **標準化通信**: 基於 MQTT 3.1.1，確保互操作性
- **結構化數據**: JSON 格式訊息，支援 Schema 驗證
- **設備管理**: 自動設備發現與生命週期管理
- **網路診斷**: 內建網路診斷與分析功能
- **LLM 整合**: 支援 AI 驅動的自動化診斷
- **擴展性**: 支援多租戶、多站點部署
- **可靠性**: LWT 機制確保設備狀態追蹤

## 🔧 支援的設備類型

- **網路設備**: 路由器、交換機、存取點
- **終端設備**: NIC、感測器、智慧設備  
- **診斷工具**: 網路分析器、測試設備
- **管理系統**: 控制器、監控平台

## 📚 相關資源

- **開發工具**: [../developers/](../developers/) - 開發者工具與整合指南
- **測試框架**: [../../test/](../../test/) - 測試工具與範例
- **配置文件**: [../../configs/](../../configs/) - 配置範例
- **原始碼**: [../../](../../) - RTK Controller 實作

## 🆘 支援與貢獻

如遇問題或需要協助，請：

1. 查看 [實作指南](12-implementation-guide.md) 的故障排除章節
2. 參考 [完整範例](11-examples.md) 中的常見場景
3. 檢查 [開發者文檔](../developers/) 的相關指南

---

**版本**: v1.0  
**最後更新**: 2024-08-20  
**協議狀態**: 穩定版本