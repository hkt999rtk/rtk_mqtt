# 🏠 家用網路診斷系統 (LLM + MCP) 架構報告

## 1. 系統目標
- 提供家用網路使用者一個自動化診斷與排查平台  
- 透過 **LLM 判斷使用者問題意圖 (Intent)**  
- 由 **MCP 工具層** 提供可調用的資訊收集、測試與調整能力  
- 生成 **診斷報告** 與 **修復建議**，協助使用者快速定位問題  

---

## 2. 設計流程總覽
1. **使用者輸入**：自然語句（多為英文/中文抱怨或問題）  
2. **LLM 分類**：判斷 Intent（如 no_internet / slow_speed / roaming_issue）  
3. **MCP 工具調用**：依 Intent 決定使用哪些 Read/Test/Act 工具  
4. **資料收集與分析**：工具回傳 metrics / 狀態，LLM 解讀並比對閾值  
5. **輸出建議**：產生人類可理解的報告與可選動作（dry run → act）  

---

## 3. 意圖分類 (Intents)
- **A. no_internet**：完全不能上網  
- **B. slow_speed**：速度低於方案  
- **C. unstable_disconnect**：斷斷續續 / 不穩定  
- **D. weak_signal_coverage**：某些房間 Wi-Fi 收訊差  
- **E. realtime_latency**：遊戲 / 視訊會議延遲高  
- **F. device_specific_issue**：特定 IoT / 裝置連不上  
- **G. roaming_issue**：設備不漫遊、黏遠端 AP  
- **H. mesh_backhaul_issue**：Mesh 節點 / 回程瓶頸  
- **I. dhcp_dns_issue_advanced**：進階 DHCP/DNS 問題  

---

## 4. 系統組件 (Components)

### 4.1 LLM 前端
- **Intent 分類器**：將使用者語句映射到 Intent  
- **Utterance 樣本庫**：收錄常見用戶英文提問  
- **Policy Orchestrator**：決定「讀 → 測 → 調」的工具呼叫流程  
- **報告生成器**：輸出診斷報告（發現 → 歸因 → 建議）  

### 4.2 MCP 工具層
分為三大類：  

#### (1) Read 工具（盤點 / 現況收集）
- `net.get_topology`：AP / Router / Mesh 拓撲  
- `wifi.get_radios`：AP 的 RF 配置（頻段 / 信道 / 功率）  
- `clients.list`：所有裝置狀態（RSSI, 頻段, 漫遊歷史）  
- `dhcpdns.get_config`：DHCP / DNS 設定、重疊檢查  
- `traffic.top_talkers`：頻寬佔用清單  

#### (2) Test 工具（測試 / 探測）
- `net.ping`：延遲 / 抖動 / 丟包  
- `net.speedtest`：Router / Client 實測速度  
- `wifi.survey`：鄰居干擾 / 信道利用率  
- `mesh.get_backhaul`：回程類型與品質  
- `wifi.roam_probe`：漫遊切換延遲  

#### (3) Act 工具（調整 / 優化）
- `wifi.set_power`：調整 AP 功率  
- `wifi.set_channel`：調整信道與頻寬  
- `wifi.set_roaming`：開啟 802.11r/k/v，調 RSSI 閾值  
- `mesh.set_backhaul`：切換回程模式（有線/無線/專用）  
- `dhcpdns.set`：統一 DHCP/DNS 來源，避免衝突  
- `wifi.client_steer`：手動導引裝置到最佳 AP  

---

## 5. 工作流程範例

### 5.1 使用者問題 → 工具呼叫範例
- **問題**：「My Wi-Fi is connected but I can’t get online.」  
  → Intent = `no_internet`  
  → 工具呼叫：  
  - `net.get_wan_status`  
  - `net.ping(gateway, 8.8.8.8)`  
  - `dhcpdns.get_config`  

- **問題**：「Zoom keeps freezing during meetings.」  
  → Intent = `realtime_latency`  
  → 工具呼叫：  
  - `net.ping(isp_gateway)`  
  - `net.speedtest(client)`  
  - `traffic.top_talkers`  
  - `qos.get_status`  

- **問題**：「My phone doesn’t switch to the closer Wi-Fi point.」  
  → Intent = `roaming_issue`  
  → 工具呼叫：  
  - `clients.list`  
  - `wifi.roam_probe(phone)`  
  - **建議動作**：`wifi.set_roaming` 或 `wifi.set_power`  

---

## 6. 報告輸出格式 (建議)
- **發現 (Findings)**：實測數據與異常指標  
- **歸因 (Root Cause)**：映射 Decision Tree 節點  
- **建議 (Recommendations)**：分「立即行動 / 結構調整 / 進階優化」  
- **追蹤 (Follow-up)**：可排程重測 / 長期監控  

---

## 7. 安全與回滾
- **Dry Run**：先模擬影響範圍（多少裝置受影響）  
- **Change Set**：每次變更有 ID，可回滾  
- **Scope**：變更可限定到 AP / Client / 節點  
- **Approval Token**：高風險操作需使用者批准  

---

## 8. 快速落地建議
1. **先實作 Read/Test 工具**（低風險、高價值）  
2. **再加 Act 工具**（需審批與回滾）  
3. **建立常見意圖樣本庫**（英文常見抱怨對應 Intent）  
4. **設定預設閾值**（RSSI、Jitter、Loss、Backhaul）  
5. **統一回傳格式**（status, metrics, evidence, advice, confidence）  

---

# 📌 總結
這套系統將：
- 使用 **LLM** 做意圖判斷與策略規劃  
- 透過 **MCP 工具層** 取得家中網路全貌（AP / NIC / IoT）  
- 提供可讀、可操作的 **診斷報告與修復建議**  

最終達成「像網路工程師一樣」的自動化診斷體驗。

