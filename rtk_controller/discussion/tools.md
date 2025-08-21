# 工具設計總覽（給 LLM 調用的 MCP）

## 📡 資料來源基礎
**本系統基於現有 RTK MQTT 框架**，能夠透過標準化的 MQTT 診斷協議取得完整家用網路資訊：
- **每個 AP/Router**：RF 狀態、功率、信道、回程連線品質、韌體版本
- **每個 NIC/終端裝置**：RSSI、SNR、連線頻段、PHY 速率、漫遊歷史、驅動資訊
- **每個 IoT 裝置**：連線狀態、電源管理、協議支援、健康度指標

**MQTT Topic 架構**：`rtk/v1/{tenant}/{site}/{device_id}/{message_type}`
- `state` - 裝置健康摘要（retained）
- `telemetry/{metric}` - 效能指標
- `topology/update` - 拓撲變更通知
- 詳細規格參考：`../docs/SPEC.md`

## A) 工具分層與原則

- **Read（只讀）**：盤點、量測、抓指標。安全、可併發、高頻次。
- **Test（主動測試）**：產生可控流量與探測（ping、iperf、roam probe）。中等風險。
- **Act（調參/變更）**：調功率、頻道、閾值、DHCP/DNS 等。需 dry_run、rollback、scope 與審批。
- **格式統一**：每個工具回傳 `{status, metrics, evidence, advice, confidence, trace_id}`，便於 LLM 比較與溯源。

## B) 工具族群與用途

### 1. 拓撲與資產（Inventory / Topology）
- `net.get_topology()`：整體拓撲（主/副節點、回程型態、上下游關係、房間對應）
- `wifi.get_radios()`：每 AP/RF 的頻段、頻寬、發射功率、信道、DFS 狀態、韌體版
- `clients.list()`：所有終端（NIC/IoT）清單，含 RSSI、SNR、BSSID、頻段、PHY、速率、漫遊歷史

### 2. 無線現況與干擾（RF / Survey）
- `wifi.survey(band|channels?)`：鄰網密度、信道重疊、雜訊底噪、信道利用率
- `wifi.utilization(ap_id|bssid)`：每 BSS 的空口利用率、重傳率、PHY error
- `wifi.spectrum_snapshot(ap_id)`（可選）：簡化頻譜快照/干擾源指標

### 3. 回程與佈線（Mesh / Backhaul）
- `mesh.get_backhaul(ap_id)`：回程型態（wired/wireless/專用）、實測吞吐、RSSI/SINR
- `mesh.backhaul_test(ap_id)`：對主節點跑點對點吞吐/延遲測試

### 4. 連通性與效能（Connectivity / QoE）
- `net.ping(target, count, from=node|client)`：延遲/抖動/丟包
- `net.speedtest(scope=router|client)`：WAN 端/端點實測速度
- `qos.get_status()`：QoS/Smart Queue/Classification 設定與佔用

### 5. DHCP / DNS / IP（Advanced）
- `dhcpdns.get_config()`：DHCP 範圍、租期、作用伺服器清單、DNS 下發名單、衝突偵測
- `dhcpdns.scan_rogue()`：掃描 Rogue DHCP、重疊網段
- `name.resolve_test(hosts[])`：多 DNS 比較解析時間/成功率

### 6. 漫遊與客戶體驗（Roaming / Client Health）
- `wifi.roam_probe(client_id, path=[apA, apB])`：誘發/引導漫遊，量測切換延遲與丟包
- `clients.health(client_id)`：電源策略、驅動版本、連線中斷日誌、PHY 重試率

### 7. 流量與占用（Traffic / Abuse）
- `traffic.top_talkers(window)`：每裝置上/下行、協定分佈、長流維持
- `traffic.shaper_status()`：是否有上傳飽和、Bufferbloat 指標（如 FQ_Codel 啟用與否）

### 8. 動作（Act / Tuning）— 需乾跑與回滾
- `wifi.set_power(ap_id, level)`：調整 RF 功率（支援 dry_run 與建議值）
- `wifi.set_channel(ap_id, channel, width)`：設定信道/頻寬（支援 DFS 檢查）
- `wifi.set_roaming(ap_id, r=true, k=true, v=true, thresholds)`：開 11r/k/v 與 RSSI 離網/入網閾值
- `mesh.set_backhaul(ap_id, mode=wired|wireless|dedicated)`：更換回程模式
- `dhcpdns.set(authority=router|pihole|custom, range, dns[])`：統一 DHCP/DNS 派發源
- `wifi.client_steer(client_id, target_ap)` / `wifi.deauth(client_id)`：最後手段，需提示風險

**所有 Act 工具均應支援**：`dry_run`, `change_set_id`, `rollback(change_set_id)`, `scope`, `approval_token`


意圖 → 工具流程（Orchestration Blueprint）
LLM 判別 Intent 後，使用「讀→測→調」階梯式決策。閾值（可調參）示例：
RSSI_WARN=-70 dBm、JITTER_WARN=30 ms、LOSS_WARN=1%、UPLINK_MIN=5–10 Mbps（會議）、BACKHAUL_MIN = 寬頻額度的 50%。EOS
A. no_internet
net.get_wan_status → 若無 IP/PPPoE fail → 產出 ISP/帳密檢查建議。EOS
net.ping(gateway, 8.8.8.8, from=router) → 判外線/ISP。EOS
clients.list（是否全域失效或單一設備）。EOS
必要時 dhcpdns.get_config、dhcpdns.scan_rogue。EOS
Act（可選）：dhcpdns.set（單一 DHCP）、net.restart_wan（若你提供）。EOS
B. slow_speed
net.speedtest(scope=router) vs net.speedtest(scope=client) 比對瓶頸位。EOS
室內側慢 → wifi.survey + wifi.utilization + clients.list（2.4G 誤連、干擾）。EOS
Mesh 尾端慢 → mesh.get_backhaul / mesh.backhaul_test。EOS
Act：wifi.set_channel/width、mesh.set_backhaul、wifi.set_power。EOS
C. unstable_disconnect
net.ping(gateway, from=router)（外線），wifi.utilization（空口飽和）。EOS
clients.health（驅動/省電策略），traffic.top_talkers（上傳飽和）。EOS
Act：qos.enable_or_tune（若有）、wifi.set_power 微調、固件更新提示。EOS
D. weak_signal_coverage
clients.list（按房間/RSSI），wifi.survey。EOS
Act：wifi.set_power（均衡覆蓋）、新增節點建議、wifi.set_channel。EOS
E. realtime_latency
net.ping（Jitter/丟包）、traffic.shaper_status、traffic.top_talkers。EOS
Act：qos.set_profile(voice_video)、提醒限速/排程上傳任務。EOS
F. device_specific_issue
clients.health(device)、clients.list（頻段/加密）、wifi.survey（DFS/距離）。EOS
Act：為舊裝置建 2.4G SSID、關 WPA3/開混合、wifi.client_steer 到近 AP。EOS
G. roaming_issue
wifi.roam_probe(client)、clients.roam_history(client)。EOS
Act：wifi.set_roaming(r/k/v, thresholds)、wifi.set_power 均衡、必要時 client_steer。EOS
H. mesh_backhaul_issue
mesh.get_backhaul / mesh.backhaul_test。EOS
Act：mesh.set_backhaul(wired|dedicated)、節點移位建議（距離/牆阻）。EOS
I. dhcp_dns_issue_advanced
dhcpdns.get_config、dhcpdns.scan_rogue、name.resolve_test。EOS
Act：dhcpdns.set(single_authority)、統一 DNS 下發、rollback 機制。EOS