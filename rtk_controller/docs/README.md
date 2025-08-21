# RTK Controller 架構文檔

本目錄包含 RTK Controller LLM 診斷系統的完整架構文檔和 PlantUML 圖表。

## 📋 文檔概覽

### 主要架構文檔

| 文檔 | 描述 | 內容 |
|------|------|------|
| [`LLM_ARCHITECTURE.md`](./LLM_ARCHITECTURE.md) | LLM 系統詳細架構說明 | 系統組件、數據流、開發指南 |
| [`README.md`](./README.md) | 本文檔 | 架構文檔索引和使用指南 |

### PlantUML 架構圖

| 圖表文件 | 圖表類型 | 描述 |
|----------|----------|------|
| [`ARCH_CONTROLLER.puml`](./ARCH_CONTROLLER.puml) | 控制器架構圖 | RTK Controller 核心系統架構 |
| [`ARCH_TEST.puml`](./ARCH_TEST.puml) | 測試架構圖 | 完整測試體系和測試工具 |
| [`ARCH_TOOLS.puml`](./ARCH_TOOLS.puml) | 工具架構圖 | 22個診斷工具的詳細分類和功能 |
| [`ARCH_DATA_FLOW.puml`](./ARCH_DATA_FLOW.puml) | 數據流圖 | 系統內部數據流和處理流程 |
| [`ARCH_DEPLOYMENT.puml`](./ARCH_DEPLOYMENT.puml) | 部署架構圖 | 生產環境部署和生態系統整合 |

## 🏗️ 系統架構概覽

### 核心組件

RTK Controller 由以下主要組件構成：

1. **LLM 診斷引擎**
   - 工具註冊和執行
   - 會話管理
   - 指標收集
   - 智能分析

2. **診斷工具集 (22個工具)**
   - 基礎診斷工具 (6個)
   - WiFi 高級工具 (8個) 
   - Mesh 網路工具 (6個)
   - 配置管理工具 (8個)

3. **存儲層**
   - BuntDB 嵌入式數據庫
   - 類型轉換層
   - 事務支持

4. **通訊層**
   - MQTT 客戶端整合
   - 消息處理
   - 設備通訊

### 工具分類

#### 🔍 基礎診斷工具 (6個)
- `topology.get_full` - 網路拓撲獲取
- `clients.list` - 客戶端列表
- `network.speedtest_full` - 網路速度測試
- `diagnostics.wan_connectivity` - WAN 連通性測試
- `qos.get_status` - QoS 狀態查詢
- `traffic.get_stats` - 流量統計

#### 📡 WiFi 高級工具 (8個)
- `wifi.scan_channels` - 頻道掃描
- `wifi.analyze_interference` - 干擾分析
- `wifi.spectrum_utilization` - 頻譜使用率
- `wifi.signal_strength_map` - 信號強度地圖
- `wifi.coverage_analysis` - 覆蓋範圍分析
- `wifi.roaming_optimization` - 漫遊優化
- `wifi.throughput_analysis` - 輸送量分析
- `wifi.latency_profiling` - 延遲分析

#### 🕸️ Mesh 網路工具 (6個)
- `mesh.get_topology` - Mesh 拓撲視覺化
- `mesh.node_relationship` - 節點關係分析
- `mesh.path_optimization` - 路徑優化
- `mesh.backhaul_test` - 回程測試
- `mesh.load_balancing` - 負載均衡
- `mesh.failover_simulation` - 故障切換模擬

#### ⚙️ 配置管理工具 (8個)
- `config.wifi_settings` - WiFi 配置管理
- `config.qos_policies` - QoS 策略管理
- `config.security_settings` - 安全配置
- `config.band_steering` - 頻段引導
- `config.auto_optimize` - 自動優化
- `config.validate_changes` - 配置驗證
- `config.rollback_safe` - 安全回滾
- `config.impact_analysis` - 影響分析

## 🔧 使用方式

### 查看架構圖

1. **在線查看**：使用 PlantUML 在線編輯器
   - 訪問 [PlantUML Online](http://www.plantuml.com/plantuml/)
   - 複製 `.puml` 文件內容
   - 貼上並生成圖表

2. **本地生成**：使用 PlantUML 工具
   ```bash
   # 安裝 PlantUML
   brew install plantuml  # macOS
   
   # 生成圖片
   plantuml docs/ARCH_CONTROLLER.puml
   plantuml docs/ARCH_TEST.puml
   plantuml docs/ARCH_TOOLS.puml
   plantuml docs/ARCH_DATA_FLOW.puml
   plantuml docs/ARCH_DEPLOYMENT.puml
   ```

3. **IDE 整合**：在 VS Code 中使用 PlantUML 擴展
   - 安裝 "PlantUML" 擴展
   - 開啟 `.puml` 文件
   - 使用 `Alt+D` 預覽圖表

### 架構圖說明

#### 🏛️ ARCH_CONTROLLER.puml
展示 RTK Controller 核心系統架構，包括：
- CLI 接口層
- LLM 診斷引擎
- 存儲層
- 網路管理層
- MQTT 通訊層

#### 🧪 ARCH_TEST.puml
展示完整的測試體系架構：
- 單元測試層
- 模擬對象層
- 基準測試層
- 整合測試層

#### 🔧 ARCH_TOOLS.puml  
詳細展示 22個診斷工具的分類和功能：
- 工具類別（Read/Test/Act/WiFi）
- 所需能力
- 工具間關係
- 實現狀態

#### 🔄 ARCH_DATA_FLOW.puml
展示系統內部的數據流和處理流程：
- 工具執行流程
- 配置變更流程
- 錯誤處理和回滾
- 監控和健康檢查

#### 🚀 ARCH_DEPLOYMENT.puml
展示生產環境的部署架構：
- 系統組件部署
- 網路設備整合
- 監控和備份系統
- 外部系統整合

## 📈 開發狀態

### ✅ 已完成項目

- [x] 基礎診斷工具實現 (6個)
- [x] WiFi 高級診斷工具 (8個)  
- [x] Mesh 網路診斷工具 (6個)
- [x] 配置管理工具 (8個)
- [x] 工具引擎整合
- [x] 測試套件完成
- [x] 指標收集系統
- [x] 會話管理
- [x] 架構文檔

### 📊 統計數據

- **總工具數量**: 22個
- **工具類別**: 4種 (Read/Test/Act/WiFi)
- **測試覆蓋率**: 100%
- **文檔完整性**: 100%
- **編譯狀態**: ✅ 成功

## 🔗 相關資源

- [PLAN_A.md](../discussion/PLAN_A.md) - 開發計劃
- [CLAUDE.md](../CLAUDE.md) - 開發指南
- [rtk_controller/](../) - 源代碼目錄

## 🏷️ 版本信息

- **版本**: v1.0.0
- **最後更新**: 2025-08-21
- **Go 版本**: 1.23+
- **架構**: ARM64/AMD64
- **許可**: MIT

---

*此文檔由 RTK Controller 開發團隊維護*