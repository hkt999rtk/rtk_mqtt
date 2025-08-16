#!/usr/bin/env python3
"""
Generate diagrams using Graphviz (.dot format) for better quality
"""

import subprocess
from pathlib import Path
import os

def create_system_architecture_dot():
    """Create system architecture diagram in DOT format"""
    dot_content = '''
digraph mqtt_system {
    rankdir=TB;
    node [shape=box, style=filled, fontname="Arial"];
    edge [fontname="Arial"];
    
    // Title
    label="MQTT Diag System Architecture";
    labelloc=t;
    fontsize=20;
    fontname="Arial Bold";
    
    // Define clusters for better organization
    subgraph cluster_controller {
        label="Controller Layer";
        style=filled;
        color=lightgrey;
        
        controller [label="Controller\\n(Cloud/Local)\\n\\n• Send diagnosis commands\\n• Collect diagnostic data\\n• Monitor device health", 
                   fillcolor=lightgreen, shape=box];
    }
    
    subgraph cluster_broker {
        label="Message Broker";
        style=filled;
        color=lightblue;
        
        broker [label="MQTT Broker\\n\\n• Central message router\\n• Handle retained messages\\n• Manage LWT messages", 
               fillcolor=lightcyan, shape=ellipse];
    }
    
    subgraph cluster_devices {
        label="Device Layer";
        style=filled;
        color=lightyellow;
        
        wifi_router [label="WiFi Router\\n\\nPublishes:\\n• state\\n• wifi.scan\\n• evt.roam\\n• lwt", 
                    fillcolor=orange, shape=box];
        
        server [label="Server\\n\\nPublishes:\\n• state\\n• cpu_usage\\n• memory\\n• disk", 
               fillcolor=orange, shape=box];
        
        iot_sensor [label="IoT Sensor\\n\\nPublishes:\\n• state\\n• temperature\\n• humidity\\n• battery", 
                   fillcolor=orange, shape=box];
        
        network_switch [label="Network Switch\\n\\nPublishes:\\n• state\\n• port.stats\\n• evt.link\\n• interface", 
                       fillcolor=orange, shape=box];
    }
    
    // Connections
    controller -> broker [label="Commands\\n& Responses", color=red, penwidth=2];
    broker -> controller [color=red, penwidth=2];
    
    // Device connections
    wifi_router -> broker [label="state/telemetry/evt", color=blue];
    server -> broker [label="state/telemetry/evt", color=blue];
    iot_sensor -> broker [label="state/telemetry/evt", color=blue]; 
    network_switch -> broker [label="state/telemetry/evt", color=blue];
    
    broker -> wifi_router [label="cmd/req", color=green, style=dashed];
    broker -> server [label="cmd/req", color=green, style=dashed];
    broker -> iot_sensor [label="cmd/req", color=green, style=dashed];
    broker -> network_switch [label="cmd/req", color=green, style=dashed];
}
'''
    return dot_content

def create_topic_structure_dot():
    """Create topic structure hierarchy in DOT format"""
    dot_content = '''
digraph topic_structure {
    rankdir=TB;
    node [shape=box, style=filled, fontname="Arial"];
    edge [fontname="Arial"];
    
    // Title
    label="MQTT Topic Structure Hierarchy";
    labelloc=t;
    fontsize=18;
    fontname="Arial Bold";
    
    // Root topic
    root [label="rtk/v1/{tenant}/{site}/{device_id}/", 
          fillcolor=lightcoral, fontname="Courier Bold", fontsize=12];
    
    // Main topic branches
    state [label="state\\n(retained)\\n\\nDevice health\\nsummary", fillcolor=lightgreen];
    telemetry [label="telemetry/*\\n\\nMetrics &\\nperformance\\ndata", fillcolor=lightblue];
    evt [label="evt/*\\n\\nEvents, alerts\\n& errors", fillcolor=lightyellow];
    attr [label="attr\\n(retained)\\n\\nDevice\\nattributes", fillcolor=lightpink];
    lwt [label="lwt\\n(retained)\\n\\nOnline/offline\\nstatus", fillcolor=lightgray];
    cmd [label="cmd/*\\n\\nCommands &\\nresponses", fillcolor=lightcyan];
    
    // Command subtopics
    cmd_req [label="req\\n(downlink)\\n\\nCommand\\nrequests", fillcolor=white, shape=ellipse];
    cmd_ack [label="ack\\n(uplink)\\n\\nCommand\\nack", fillcolor=white, shape=ellipse];
    cmd_res [label="res\\n(uplink)\\n\\nCommand\\nresults", fillcolor=white, shape=ellipse];
    
    // Connections
    root -> {state, telemetry, evt, attr, lwt, cmd} [penwidth=2];
    cmd -> {cmd_req, cmd_ack, cmd_res} [color=gray];
    
    // Direction indicators
    uplink [label="Uplink\\n(Device → Controller)", 
            fillcolor=lightgreen, shape=note, style="filled,rounded"];
    downlink [label="Downlink\\n(Controller → Device)", 
             fillcolor=lightcoral, shape=note, style="filled,rounded"];
    
    // Examples
    example1 [label="Example:\\nrtk/v1/office/floor1/router-001/evt/wifi.roam_miss", 
             fillcolor=white, shape=plaintext, fontname="Courier", fontsize=10];
    example2 [label="Example:\\nrtk/v1/factory/workshop/camera-003/telemetry/temperature", 
             fillcolor=white, shape=plaintext, fontname="Courier", fontsize=10];
    example3 [label="Example:\\nrtk/v1/corporate/building-a/laptop-005/cmd/req", 
             fillcolor=white, shape=plaintext, fontname="Courier", fontsize=10];
    
    // Invisible edges for layout
    {rank=same; uplink, downlink}
    {rank=same; example1, example2, example3}
}
'''
    return dot_content

def create_message_flow_dot():
    """Create message flow sequence diagram in DOT format"""
    dot_content = '''
digraph message_flow {
    rankdir=TB;
    node [shape=box, style=filled, fontname="Arial"];
    edge [fontname="Arial"];
    
    // Title
    label="MQTT Diagnostic Communication Flow";
    labelloc=t;
    fontsize=18;
    fontname="Arial Bold";
    
    // Participants
    device [label="Device", fillcolor=lightblue, shape=box];
    broker [label="MQTT Broker", fillcolor=lightcyan, shape=ellipse];
    controller [label="Controller", fillcolor=lightgreen, shape=box];
    
    // Timeline flow (using invisible nodes for sequence)
    t1 [style=invisible];
    t2 [style=invisible];
    t3 [style=invisible];
    t4 [style=invisible];
    t5 [style=invisible];
    t6 [style=invisible];
    
    // Flow steps
    step1 [label="1. Event Trigger\\nevt/wifi.xxx", fillcolor=pink, shape=note];
    step2 [label="2. Diagnosis Request\\ncmd/req", fillcolor=lightblue, shape=note];
    step3 [label="3. Command Ack\\ncmd/ack", fillcolor=lightgreen, shape=note];
    step4 [label="4. Data Collection\\n[Processing...]", fillcolor=lightyellow, shape=note];
    step5 [label="5. Diagnosis Result\\ncmd/res", fillcolor=orange, shape=note];
    step6 [label="6. State Update\\nstate (updated)", fillcolor=purple, shape=note];
    
    // Sequence layout
    {rank=same; device, broker, controller}
    
    // Flow connections
    device -> step1 -> broker -> controller [color=red, penwidth=2, label="Event"];
    controller -> step2 -> broker -> device [color=blue, penwidth=2, label="Request"];
    device -> step3 -> broker -> controller [color=green, penwidth=2, label="Ack"];
    device -> step4 [color=gray, style=dashed];
    device -> step5 -> broker -> controller [color=orange, penwidth=2, label="Result"];
    device -> step6 -> broker -> controller [color=purple, penwidth=2, label="Update"];
    
    // Invisible sequence for layout
    t1 -> t2 -> t3 -> t4 -> t5 -> t6 [style=invisible];
    {rank=same; t1, step1}
    {rank=same; t2, step2}
    {rank=same; t3, step3}
    {rank=same; t4, step4}
    {rank=same; t5, step5}
    {rank=same; t6, step6}
}
'''
    return dot_content

def create_roaming_sequence_dot():
    """Create roaming diagnosis sequence in DOT format"""
    dot_content = '''
digraph roaming_sequence {
    rankdir=TB;
    node [shape=box, style=filled, fontname="Arial"];
    edge [fontname="Arial"];
    
    // Title
    label="WiFi Roaming Diagnosis Sequence";
    labelloc=t;
    fontsize=18;
    fontname="Arial Bold";
    
    // Main process flow
    trigger [label="RSSI < -70dB for 10s\\nRoaming not triggered", 
            fillcolor=pink, shape=ellipse];
    
    event [label="Device publishes:\\nevt/wifi.roam_miss\\n(QoS 1)", 
          fillcolor=red, fontcolor=white];
    
    request [label="Controller requests:\\ncmd/req diagnosis.get\\ntype: wifi.roaming", 
            fillcolor=blue, fontcolor=white];
    
    ack [label="Device acknowledges:\\ncmd/ack\\n(< 1s response)", 
        fillcolor=green, fontcolor=white];
    
    collect [label="Data Collection (3-15s):\\n• Scan history\\n• RF statistics\\n• Candidate AP analysis", 
            fillcolor=yellow];
    
    result [label="Device responds:\\ncmd/res (JSON 5-10KB)\\n• Root cause analysis\\n• Recommended actions", 
           fillcolor=orange];
    
    action [label="Controller triggers:\\ncmd/req wifi.scan\\nEnvironment scan", 
           fillcolor=blue, fontcolor=white];
    
    scan_ack [label="Device confirms:\\ncmd/ack\\nExecuting scan", 
             fillcolor=green, fontcolor=white];
    
    scan_exec [label="Execute Scan (2-5s)\\nEnvironment analysis", 
              fillcolor=yellow];
    
    scan_result [label="Scan complete:\\ncmd/res\\nScan results", 
                fillcolor=orange];
    
    state_update [label="State update:\\nstate (updated)\\nhealth: warn→ok", 
                 fillcolor=purple, fontcolor=white];
    
    // Flow
    trigger -> event -> request -> ack -> collect -> result;
    result -> action -> scan_ack -> scan_exec -> scan_result -> state_update;
    
    // Add timing annotations
    timing1 [label="Immediate", shape=plaintext, fontcolor=red];
    timing2 [label="< 1 second", shape=plaintext, fontcolor=green];
    timing3 [label="3-15 seconds", shape=plaintext, fontcolor=orange];
    timing4 [label="2-5 seconds", shape=plaintext, fontcolor=orange];
    
    event -> timing1 [style=dashed, color=red];
    ack -> timing2 [style=dashed, color=green];
    collect -> timing3 [style=dashed, color=orange];
    scan_exec -> timing4 [style=dashed, color=orange];
}
'''
    return dot_content

def create_connection_failure_dot():
    """Create connection failure sequence in DOT format"""
    dot_content = '''
digraph connection_failure {
    rankdir=TB;
    node [shape=box, style=filled, fontname="Arial"];
    edge [fontname="Arial"];
    
    // Title
    label="WiFi Connection Failure Diagnosis";
    labelloc=t;
    fontsize=18;
    fontname="Arial Bold";
    
    // Process flow
    attempt [label="Connection Attempt\\nWPA3 SAE Authentication", 
            fillcolor=lightblue, shape=ellipse];
    
    failure [label="Authentication Failed\\nSAE timeout", 
            fillcolor=pink, shape=ellipse];
    
    event [label="evt/wifi.connect_fail\\nseverity: error\\nstage: authentication", 
          fillcolor=red, fontcolor=white];
    
    request [label="cmd/req diagnosis.get\\ninclude_auth_details\\ntimeout: 20s", 
            fillcolor=blue, fontcolor=white];
    
    ack [label="cmd/ack\\nStart analysis", 
        fillcolor=green, fontcolor=white];
    
    analysis [label="Connection Analysis:\\n• SAE exchange records\\n• RF environment\\n• AP capability detection", 
             fillcolor=yellow];
    
    result [label="cmd/res\\nDetailed auth log\\n• SAE timeout\\n• Backup AP suggestions", 
           fillcolor=orange];
    
    retry [label="cmd/req wifi.connect\\ntarget: backup AP", 
          fillcolor=blue, fontcolor=white];
    
    retry_ack [label="cmd/ack\\nRetrying connection", 
              fillcolor=green, fontcolor=white];
    
    success [label="Connection Successful\\nto backup AP", 
            fillcolor=lightgreen, shape=ellipse];
    
    success_result [label="cmd/res\\nConnection success", 
                   fillcolor=orange];
    
    state_update [label="state (updated)\\nhealth: error→ok\\nstatus: connected", 
                 fillcolor=purple, fontcolor=white];
    
    // Flow connections
    attempt -> failure -> event -> request -> ack;
    ack -> analysis -> result -> retry -> retry_ack;
    retry_ack -> success -> success_result -> state_update;
    
    // Add device type annotation
    device_type [label="Device: laptop-005\\nTarget: CorporateNet-5G", 
                shape=note, fillcolor=lightcyan];
    
    attempt -> device_type [style=dashed, color=gray];
}
'''
    return dot_content

def create_arp_loss_dot():
    """Create ARP loss sequence in DOT format"""
    dot_content = '''
digraph arp_loss {
    rankdir=TB;
    node [shape=box, style=filled, fontname="Arial"];
    edge [fontname="Arial"];
    
    // Title
    label="ARP Loss Diagnosis Flow";
    labelloc=t;
    fontsize=18;
    fontname="Arial Bold";
    
    // Process flow
    arp_req [label="ARP Request Sent\\nto gateway", 
            fillcolor=lightblue, shape=ellipse];
    
    no_response [label="2 Consecutive\\nNo Response", 
                fillcolor=pink, shape=ellipse];
    
    event [label="evt/wifi.arp_loss\\nconsecutive: 2\\ngateway no response", 
          fillcolor=red, fontcolor=white];
    
    request [label="cmd/req diagnosis.get\\ninclude_rf_stats\\ninclude_interference\\ntimeout: 25s", 
            fillcolor=blue, fontcolor=white];
    
    ack [label="cmd/ack\\nEstimated completion: 20s\\nAnalysis in progress", 
        fillcolor=green, fontcolor=white];
    
    analysis [label="Network Analysis (20s):\\n• RF interference scan\\n• Traffic analysis\\n• Baseband statistics", 
             fillcolor=yellow];
    
    result [label="cmd/res\\nroot_cause: channel_interference\\n• Channel interference\\n• AP load fluctuation", 
           fillcolor=orange];
    
    config [label="cmd/req network.arp.config\\nIncrease timeout values", 
           fillcolor=blue, fontcolor=white];
    
    config_ack [label="cmd/ack\\nConfiguration updated", 
               fillcolor=green, fontcolor=white];
    
    config_result [label="cmd/res\\nConfiguration complete", 
                  fillcolor=orange];
    
    telemetry [label="telemetry/connectivity\\n(every 10 seconds)\\nARP success rate improving", 
              fillcolor=cyan];
    
    state_update [label="state (updated)\\nhealth: warn→ok\\nnetwork: stable", 
                 fillcolor=purple, fontcolor=white];
    
    // Flow connections
    arp_req -> no_response -> event -> request -> ack;
    ack -> analysis -> result -> config -> config_ack;
    config_ack -> config_result -> telemetry -> state_update;
    
    // Add device annotation
    device_info [label="Device: smart-camera-003\\nGateway: 192.168.10.1", 
                shape=note, fillcolor=lightcyan];
    
    arp_req -> device_info [style=dashed, color=gray];
    
    // Add continuous monitoring indicator
    monitoring [label="Continuous Monitoring\\nImproved ARP success rate", 
               shape=ellipse, fillcolor=lightgreen];
    
    telemetry -> monitoring [style=dashed, color=green];
}
'''
    return dot_content

def generate_dot_files():
    """Generate all .dot files"""
    output_dir = Path(__file__).parent.parent
    
    dot_files = [
        ('system_architecture.dot', create_system_architecture_dot()),
        ('topic_structure.dot', create_topic_structure_dot()),
        ('message_flow.dot', create_message_flow_dot()),
        ('roaming_sequence.dot', create_roaming_sequence_dot()),
        ('connection_failure.dot', create_connection_failure_dot()),
        ('arp_loss.dot', create_arp_loss_dot())
    ]
    
    generated_files = []
    for filename, content in dot_files:
        filepath = output_dir / filename
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        generated_files.append(filepath)
        print(f"✅ Generated: {filepath}")
    
    return generated_files

def convert_dot_to_png(dot_files):
    """Convert .dot files to PNG using Graphviz"""
    png_files = []
    
    for dot_file in dot_files:
        png_file = dot_file.with_suffix('.png')
        
        try:
            # Use dot command to convert to PNG
            result = subprocess.run([
                'dot', '-Tpng', '-Gdpi=300', str(dot_file), '-o', str(png_file)
            ], capture_output=True, text=True, check=True)
            
            png_files.append(png_file)
            print(f"✅ Converted: {png_file}")
            
        except subprocess.CalledProcessError as e:
            print(f"❌ Failed to convert {dot_file}: {e}")
            print(f"   Error output: {e.stderr}")
        except FileNotFoundError:
            print(f"❌ Graphviz not found. Please install with: brew install graphviz")
            break
    
    return png_files

def main():
    """Main function to generate diagrams"""
    try:
        print("🚀 Generating Graphviz DOT files...")
        dot_files = generate_dot_files()
        
        print("\n🎨 Converting DOT files to PNG...")
        png_files = convert_dot_to_png(dot_files)
        
        if png_files:
            print(f"\n🎉 Successfully generated {len(png_files)} PNG diagrams!")
            for png_file in png_files:
                print(f"📁 {png_file}")
        else:
            print("\n⚠️  No PNG files were generated. Check if Graphviz is installed.")
            
    except Exception as e:
        print(f"💥 Error: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()