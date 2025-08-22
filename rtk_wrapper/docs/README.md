# MQTT Wrapper 中介層文檔

本目錄包含 RTK MQTT Wrapper 中介層的完整文檔。

## 📁 文檔結構

| 檔案 | 說明 |
|------|------|
| [OVERVIEW.md](OVERVIEW.md) | 架構概述和設計原則 |
| [ARCHITECTURE.md](ARCHITECTURE.md) | 詳細架構設計和介面定義 |
| [IMPLEMENTATION.md](IMPLEMENTATION.md) | 具體實現範例和多廠牌支援 |
| [BUILD_INTEGRATION.md](BUILD_INTEGRATION.md) | 構建系統整合和部署指南 |
| [DEV_PLAN.md](DEV_PLAN.md) | 開發計劃和時程安排 |
| [CONFIG_REFERENCE.md](CONFIG_REFERENCE.md) | 配置參考和範例 |

## 🚀 快速開始

1. **了解架構**: 閱讀 [OVERVIEW.md](OVERVIEW.md) 了解 Wrapper 的基本概念
2. **查看設計**: 閱讀 [ARCHITECTURE.md](ARCHITECTURE.md) 了解詳細的技術設計
3. **實現 Wrapper**: 參考 [IMPLEMENTATION.md](IMPLEMENTATION.md) 實現具體的設備支援
4. **建置部署**: 按照 [BUILD_INTEGRATION.md](BUILD_INTEGRATION.md) 整合到現有系統

## 🎯 主要特性

- **雙向轉換**: 支援 Device ↔ RTK 的雙向訊息轉換
- **多廠牌支援**: 智能註冊機制支援各種設備廠牌
- **彈性 JSON**: 處理任意格式的 JSON 訊息
- **高效能**: 低延遲訊息轉換 (< 10ms)
- **可觀測性**: 完整的監控和日志系統

## 🔧 支援的設備廠牌

- **Home Assistant**: 完整支援 HA 生態系統
- **Tasmota**: 支援 STATE 和 SENSOR 訊息
- **Xiaomi/Mi**: 支援 miio 和 miot 協議
- **自定義設備**: 提供開發模板

## 📋 開發狀態

當前專案尚未實現，需要按照開發計劃逐步建立。請參考 [DEV_PLAN.md](DEV_PLAN.md) 了解詳細的實施步驟。