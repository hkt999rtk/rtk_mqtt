#!/usr/bin/env python3
"""
Generate sequence diagrams using PlantUML for better quality
"""

import subprocess
from pathlib import Path
import os

def create_generic_sequence_puml():
    """Create generic MQTT sequence diagram in PlantUML format"""
    puml_content = '''
@startuml generic_sequence
!theme plain
title MQTT Diagnostic Communication Sequence

participant "Device" as D
participant "MQTT Broker" as B  
participant "Controller" as C

note over D: 觸發條件發生

D -> B: evt/wifi.xxx
activate D
B -> C: 
note right: 1. 事件觸發

C -> B: cmd/req
B -> D: diagnosis.get
note left: 2. 診斷請求

D -> B: cmd/ack
B -> C: 
note right: 3. 命令確認

note over D: [診斷資料收集中]\n3-15 秒
D -> D: 資料收集處理

D -> B: cmd/res
B -> C: 
note right: 4. 診斷結果\n(JSON 5-10KB)

D -> B: state (updated)
B -> C: 
note right: 5. 狀態更新

alt 需要修復動作
    C -> B: cmd/req (action)
    B -> D: 
    note left: 6. 修復動作 (選用)
    
    D -> B: cmd/ack
    B -> C: 
    
    D -> B: cmd/res
    B -> C: 
end

deactivate D

@enduml
'''
    return puml_content

def create_roaming_sequence_puml():
    """Create WiFi roaming diagnosis sequence in PlantUML format"""
    puml_content = '''
@startuml roaming_sequence
!theme plain
title WiFi Roaming Diagnosis Sequence
subtitle Device: office-ap-001

participant "Device\\n(office-ap-001)" as D
participant "MQTT Broker" as B
participant "Controller" as C

note over D: [RSSI < -70dB for 10s]\\n漫遊未觸發

D -> B: evt/wifi.roam_miss (QoS 1)
activate D
B -> C: 
note right: Event: 漫遊未觸發

C -> B: cmd/req diagnosis.get\\ntype: wifi.roaming (QoS 1) 
B -> D: 
note left: Request: 詳細診斷

D -> B: cmd/ack (< 1s)
B -> C: 
note right: Ack: 接收命令

note over D: [收集診斷資料: 3-15s]\\n• 掃描歷史\\n• RF 統計\\n• 候選 AP 分析
D -> D: 診斷資料收集

D -> B: cmd/res (JSON 5-10KB)
B -> C: 
note right: Result: 診斷資料\\n• 根因分析\\n• 建議動作

C -> B: cmd/req wifi.scan
B -> D: 
note left: Action: 環境掃描

D -> B: cmd/ack
B -> C: 
note right: Ack: 執行掃描

note over D: [執行掃描: 2-5s]
D -> D: 環境掃描

D -> B: cmd/res
B -> C: 
note right: Result: 掃描完成

D -> B: state (updated)\\nhealth: warn→ok
B -> C: 
note right: State: 健康狀態更新

deactivate D

@enduml
'''
    return puml_content

def create_connection_failure_sequence_puml():
    """Create connection failure sequence in PlantUML format"""
    puml_content = '''
@startuml connection_failure_sequence
!theme plain
title WiFi Connection Failure Diagnosis Sequence
subtitle Device: laptop-005

participant "Device\\n(laptop-005)" as D
participant "MQTT Broker" as B
participant "Controller" as C

note over D: [連線嘗試開始]

note over D: [WPA3 SAE 認證失敗]

D -> B: evt/wifi.connect_fail\\nseverity: error\\nstage: authentication
activate D
B -> C: 
note right: Event: 連線失敗

C -> B: cmd/req diagnosis.get\\ninclude_auth_details\\n(timeout: 20s)
B -> D: 
note left: Request: 連線診斷

D -> B: cmd/ack
B -> C: 
note right: Ack: 開始分析

note over D: [分析連線過程]\\n• SAE 交換記錄\\n• RF 環境分析\\n• AP 能力檢測
D -> D: 連線失敗分析

D -> B: cmd/res\\n(詳細認證日誌)
B -> C: 
note right: Result: 失敗分析\\n• SAE 超時\\n• 備用 AP 建議

C -> B: cmd/req wifi.connect\\ntarget: 備用AP
B -> D: 
note left: Action: 連線重試

D -> B: cmd/ack
B -> C: 
note right: Ack: 重試中

note over D: [連線成功]

D -> B: cmd/res
B -> C: 
note right: Result: 連線成功

D -> B: state (updated)\\nhealth: error→ok
B -> C: 
note right: State: connected

deactivate D

@enduml
'''
    return puml_content

def create_arp_loss_sequence_puml():
    """Create ARP loss sequence in PlantUML format"""
    puml_content = '''
@startuml arp_loss_sequence
!theme plain
title ARP Loss Diagnosis Sequence
subtitle Device: smart-camera-003

participant "Device\\n(smart-camera-003)" as D
participant "MQTT Broker" as B
participant "Controller" as C

note over D: [ARP 請求發送]

note over D: [連續 2 次無回應]

D -> B: evt/wifi.arp_loss\\nconsecutive: 2\\ngateway 無回應
activate D
B -> C: 
note right: Event: ARP 遺失

C -> B: cmd/req diagnosis.get\\ninclude_rf_stats\\ninclude_interference\\n(timeout: 25s)
B -> D: 
note left: Request: 網路診斷

D -> B: cmd/ack\\n估計完成: 20s
B -> C: 
note right: Ack: 分析中

note over D: [網路環境分析]\\n• RF 干擾掃描\\n• 流量分析\\n• 基帶統計
D -> D: 網路診斷分析

D -> B: cmd/res\\nroot_cause: channel_interference
B -> C: 
note right: Result: 網路分析\\n• 頻道干擾\\n• AP 負載波動

C -> B: cmd/req network.arp.config\\n增加超時時間
B -> D: 
note left: Action: 參數調整

D -> B: cmd/ack
B -> C: 
note right: Ack: 配置更新

D -> B: cmd/res
B -> C: 
note right: Result: 配置完成

D -> B: telemetry/connectivity\\n(每 10 秒)\\nARP 成功率改善
B -> C: 
note right: Telemetry: 持續監控

D -> B: state (updated)\\nhealth: warn→ok
B -> C: 
note right: State: 網路穩定

deactivate D

@enduml
'''
    return puml_content

def generate_puml_files():
    """Generate all .puml files"""
    output_dir = Path(__file__).parent.parent
    
    puml_files = [
        ('generic_sequence.puml', create_generic_sequence_puml()),
        ('roaming_sequence.puml', create_roaming_sequence_puml()),
        ('connection_failure_sequence.puml', create_connection_failure_sequence_puml()),
        ('arp_loss_sequence.puml', create_arp_loss_sequence_puml())
    ]
    
    generated_files = []
    for filename, content in puml_files:
        filepath = output_dir / filename
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        generated_files.append(filepath)
        print(f"✅ Generated: {filepath}")
    
    return generated_files

def convert_puml_to_png(puml_files):
    """Convert .puml files to PNG using PlantUML"""
    png_files = []
    
    # Look for PlantUML jar in the current directory first
    script_dir = Path(__file__).parent
    plantuml_jar = script_dir / "plantuml.jar"
    
    if plantuml_jar.exists():
        print(f"📦 Found PlantUML at: {plantuml_jar}")
        
        for puml_file in puml_files:
            png_file = puml_file.with_suffix('.png')
            
            try:
                # Use java -jar plantuml.jar
                result = subprocess.run([
                    'java', '-jar', str(plantuml_jar), '-tpng', str(puml_file)
                ], capture_output=True, text=True, check=True, cwd=puml_file.parent)
                
                png_files.append(png_file)
                print(f"✅ Converted: {png_file}")
                
            except subprocess.CalledProcessError as e:
                print(f"❌ Failed to convert {puml_file}: {e}")
                print(f"   Error output: {e.stderr}")
            except FileNotFoundError:
                print(f"❌ Java not found. Please install Java to run PlantUML.")
                print(f"   Install Java with: brew install openjdk")
                break
    else:
        print(f"❌ PlantUML jar not found at: {plantuml_jar}")
        print(f"   Please download plantuml.jar from https://plantuml.com/download")
        print(f"   and place it in: {script_dir}")
    
    return png_files

def main():
    """Main function to generate PlantUML sequence diagrams"""
    try:
        print("🚀 Generating PlantUML sequence diagram files...")
        puml_files = generate_puml_files()
        
        print("\n🎨 Converting PlantUML files to PNG...")
        png_files = convert_puml_to_png(puml_files)
        
        if png_files:
            print(f"\n🎉 Successfully generated {len(png_files)} PlantUML sequence diagrams!")
            for png_file in png_files:
                print(f"📁 {png_file}")
        else:
            print("\n⚠️  No PNG files were generated. Check if PlantUML is installed.")
            print("   Install with: brew install plantuml")
            print("   Or download from: https://plantuml.com/download")
            
    except Exception as e:
        print(f"💥 Error: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()