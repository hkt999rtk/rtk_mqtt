#!/bin/bash

# RTK MQTT Wrapper 示範腳本

set -e

echo "=== RTK MQTT Wrapper Demo ==="
echo

# 顏色定義
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 檢查是否已建置
if [ ! -f "../build_dir/rtk_wrapper" ]; then
    echo -e "${RED}Error: rtk_wrapper binary not found. Please run 'make build' first.${NC}"
    exit 1
fi

# 檢查配置文件
if [ ! -f "configs/wrapper.yaml" ]; then
    echo -e "${RED}Error: wrapper.yaml config file not found.${NC}"
    exit 1
fi

# 函數：發送 MQTT 訊息（需要 mosquitto_pub）
send_mqtt_message() {
    local topic=$1
    local payload=$2
    local description=$3
    
    echo -e "${YELLOW}Sending: $description${NC}"
    echo "Topic: $topic"
    echo "Payload: $payload"
    
    if command -v mosquitto_pub >/dev/null 2>&1; then
        mosquitto_pub -h localhost -t "$topic" -m "$payload"
        echo -e "${GREEN}Message sent successfully${NC}"
    else
        echo -e "${YELLOW}mosquitto_pub not found, skipping actual MQTT send${NC}"
    fi
    echo
}

# 函數：監聽 MQTT 訊息（需要 mosquitto_sub）
start_mqtt_listener() {
    local topic=$1
    local description=$2
    
    echo -e "${YELLOW}Starting MQTT listener: $description${NC}"
    echo "Listening on topic: $topic"
    
    if command -v mosquitto_sub >/dev/null 2>&1; then
        echo "Press Ctrl+C to stop listening"
        mosquitto_sub -h localhost -t "$topic" &
        MQTT_SUB_PID=$!
        echo "MQTT listener started with PID: $MQTT_SUB_PID"
    else
        echo -e "${YELLOW}mosquitto_sub not found, skipping MQTT listening${NC}"
    fi
    echo
}

# 檢查 MQTT 工具是否可用
check_mqtt_tools() {
    echo "Checking MQTT tools availability..."
    if command -v mosquitto_pub >/dev/null 2>&1 && command -v mosquitto_sub >/dev/null 2>&1; then
        echo -e "${GREEN}MQTT tools (mosquitto_pub/mosquitto_sub) are available${NC}"
        MQTT_AVAILABLE=true
    else
        echo -e "${YELLOW}MQTT tools not found. Install mosquitto-clients for full demo:${NC}"
        echo "  Ubuntu/Debian: sudo apt-get install mosquitto-clients"
        echo "  macOS: brew install mosquitto"
        echo "  This demo will still show the wrapper functionality without MQTT."
        MQTT_AVAILABLE=false
    fi
    echo
}

# 檢查是否有 MQTT broker 運行
check_mqtt_broker() {
    echo "Checking MQTT broker availability..."
    if command -v mosquitto_pub >/dev/null 2>&1; then
        if timeout 3 mosquitto_pub -h localhost -t "test/connection" -m "test" >/dev/null 2>&1; then
            echo -e "${GREEN}MQTT broker is running on localhost:1883${NC}"
            BROKER_AVAILABLE=true
        else
            echo -e "${YELLOW}MQTT broker not responding on localhost:1883${NC}"
            echo "To start a local MQTT broker:"
            echo "  mosquitto -p 1883 -v"
            BROKER_AVAILABLE=false
        fi
    else
        BROKER_AVAILABLE=false
    fi
    echo
}

# 顯示配置資訊
show_config() {
    echo "=== Configuration Information ==="
    echo "Wrapper config: configs/wrapper.yaml"
    echo "Example wrapper config: configs/wrappers/example.yaml"
    echo
    
    if [ -f "configs/wrapper.yaml" ]; then
        echo "MQTT Broker: $(grep -A 10 'mqtt:' configs/wrapper.yaml | grep 'broker:' | awk '{print $2}' | tr -d '"')"
        echo "Client ID: $(grep -A 10 'mqtt:' configs/wrapper.yaml | grep 'client_id:' | awk '{print $2}' | tr -d '"')"
        echo "Default Tenant: $(grep -A 20 'rtk:' configs/wrapper.yaml | grep 'default_tenant:' | awk '{print $2}' | tr -d '"')"
        echo "Default Site: $(grep -A 20 'rtk:' configs/wrapper.yaml | grep 'default_site:' | awk '{print $2}' | tr -d '"')"
    fi
    echo
}

# 測試 wrapper 基本功能
test_wrapper_basic() {
    echo "=== Testing Wrapper Basic Functions ==="
    
    echo "1. Testing version command:"
    ../build_dir/rtk_wrapper version
    echo
    
    echo "2. Testing help command:"
    ../build_dir/rtk_wrapper --help | head -10
    echo
    
    echo "3. Testing config validation:"
    if ../build_dir/rtk_wrapper --config configs/wrapper.yaml --log-level debug 2>&1 | head -5; then
        echo -e "${GREEN}Config validation passed${NC}"
    else
        echo -e "${RED}Config validation failed${NC}"
    fi
    echo
}

# 示範上行訊息轉換（設備 → RTK）
demo_uplink_transformation() {
    echo "=== Demo: Uplink Message Transformation (Device → RTK) ==="
    
    local example_messages=(
        "example/sensor01/state|{\"status\":\"online\",\"temperature\":23.5,\"humidity\":65.2}|Temperature Sensor Status"
        "example/light01/state|{\"power_state\":\"on\",\"brightness\":80,\"color\":\"#FF0000\"}|Smart Light Status"  
        "example/switch01/telemetry|{\"status\":\"active\",\"power\":120.5,\"energy\":45.2}|Smart Switch Telemetry"
        "example/climate01/event|{\"status\":\"warning\",\"temperature\":35.0,\"alert\":\"high_temp\"}|Climate Alert Event"
    )
    
    for msg in "${example_messages[@]}"; do
        IFS='|' read -r topic payload description <<< "$msg"
        
        echo -e "${YELLOW}Example: $description${NC}"
        echo "Original (Device Format):"
        echo "  Topic: $topic"
        echo "  Payload: $payload"
        echo
        
        echo "Expected RTK Format:"
        device_id=$(echo $topic | cut -d'/' -f2)
        message_type=$(echo $topic | cut -d'/' -f3)
        rtk_topic="rtk/v1/home/main/$device_id/$message_type"
        echo "  Topic: $rtk_topic"
        echo "  Payload: {\"schema\":\"$message_type/1.0\",\"ts\":$(date +%s)000,\"device_id\":\"$device_id\",\"payload\":{...}}"
        echo
        
        if [ "$MQTT_AVAILABLE" = true ] && [ "$BROKER_AVAILABLE" = true ]; then
            send_mqtt_message "$topic" "$payload" "Sending uplink message"
            sleep 1
        fi
        echo "---"
    done
}

# 示範下行訊息轉換（RTK → 設備）
demo_downlink_transformation() {
    echo "=== Demo: Downlink Message Transformation (RTK → Device) ==="
    
    local rtk_commands=(
        "rtk/v1/home/main/light01/cmd/req|{\"schema\":\"cmd.turn_on/1.0\",\"ts\":$(date +%s)000,\"device_id\":\"light01\",\"payload\":{\"command\":\"turn_on\",\"brightness\":75}}|Turn on smart light"
        "rtk/v1/home/main/switch01/cmd/req|{\"schema\":\"cmd.turn_off/1.0\",\"ts\":$(date +%s)000,\"device_id\":\"switch01\",\"payload\":{\"command\":\"turn_off\"}}|Turn off smart switch"
        "rtk/v1/home/main/climate01/cmd/req|{\"schema\":\"cmd.set_temperature/1.0\",\"ts\":$(date +%s)000,\"device_id\":\"climate01\",\"payload\":{\"command\":\"set_temperature\",\"temperature\":22.0}}|Set climate temperature"
    )
    
    for cmd in "${rtk_commands[@]}"; do
        IFS='|' read -r topic payload description <<< "$cmd"
        
        echo -e "${YELLOW}Example: $description${NC}"
        echo "Original (RTK Format):"
        echo "  Topic: $topic"
        echo "  Payload: $payload"
        echo
        
        device_id=$(echo $topic | cut -d'/' -f5)
        device_topic="example/$device_id/set"
        echo "Expected Device Format:"
        echo "  Topic: $device_topic"
        echo "  Payload: {\"action\":\"...\",\"value\":...}"
        echo
        
        if [ "$MQTT_AVAILABLE" = true ] && [ "$BROKER_AVAILABLE" = true ]; then
            send_mqtt_message "$topic" "$payload" "Sending downlink command"
            sleep 1
        fi
        echo "---"
    done
}

# 啟動 wrapper 並進行實際測試
run_live_test() {
    echo "=== Live Testing with MQTT Broker ==="
    
    if [ "$MQTT_AVAILABLE" != true ] || [ "$BROKER_AVAILABLE" != true ]; then
        echo -e "${YELLOW}Skipping live test - MQTT tools or broker not available${NC}"
        return
    fi
    
    echo "Starting wrapper in background..."
    ../build_dir/rtk_wrapper --config configs/wrapper.yaml --log-level info > wrapper.log 2>&1 &
    WRAPPER_PID=$!
    echo "Wrapper started with PID: $WRAPPER_PID"
    
    # 等待 wrapper 啟動
    sleep 3
    
    echo "Starting RTK message listener..."
    mosquitto_sub -h localhost -t "rtk/v1/+/+/+/+" > rtk_messages.log &
    RTK_SUB_PID=$!
    
    echo "Starting device message listener..."
    mosquitto_sub -h localhost -t "example/+/+" > device_messages.log &
    DEV_SUB_PID=$!
    
    sleep 2
    
    echo "Sending test messages..."
    
    # 發送上行訊息
    mosquitto_pub -h localhost -t "example/testsensor/state" -m '{"status":"online","temperature":24.5,"humidity":60.0}'
    sleep 1
    
    mosquitto_pub -h localhost -t "example/testlight/state" -m '{"power_state":"on","brightness":90}'
    sleep 1
    
    # 發送下行指令
    mosquitto_pub -h localhost -t "rtk/v1/home/main/testdevice/cmd/req" -m '{"schema":"cmd.turn_on/1.0","device_id":"testdevice","payload":{"command":"turn_on"}}'
    sleep 2
    
    echo "Stopping listeners and wrapper..."
    [ ! -z "$RTK_SUB_PID" ] && kill $RTK_SUB_PID 2>/dev/null
    [ ! -z "$DEV_SUB_PID" ] && kill $DEV_SUB_PID 2>/dev/null
    [ ! -z "$WRAPPER_PID" ] && kill $WRAPPER_PID 2>/dev/null
    
    sleep 1
    
    echo "Test Results:"
    if [ -f "rtk_messages.log" ] && [ -s "rtk_messages.log" ]; then
        echo -e "${GREEN}RTK Messages received:${NC}"
        cat rtk_messages.log
    else
        echo -e "${YELLOW}No RTK messages received${NC}"
    fi
    
    if [ -f "device_messages.log" ] && [ -s "device_messages.log" ]; then
        echo -e "${GREEN}Device Messages received:${NC}"
        cat device_messages.log
    else
        echo -e "${YELLOW}No device messages received${NC}"
    fi
    
    if [ -f "wrapper.log" ]; then
        echo -e "${GREEN}Wrapper Log (last 10 lines):${NC}"
        tail -10 wrapper.log
    fi
    
    # 清理
    rm -f rtk_messages.log device_messages.log wrapper.log
}

# 顯示支援的訊息格式
show_supported_formats() {
    echo "=== Supported Message Formats ==="
    echo
    echo "Uplink Topics (Device → RTK):"
    echo "  example/{device_id}/state     - Device status messages"
    echo "  example/{device_id}/telemetry - Telemetry/sensor data"  
    echo "  example/{device_id}/event     - Event/alert messages"
    echo
    echo "Downlink Topics (RTK → Device):"
    echo "  rtk/v1/{tenant}/{site}/{device_id}/cmd/req - Command requests"
    echo
    echo "Supported Device Types:"
    echo "  - sensor (temperature, humidity, etc.)"
    echo "  - light (brightness, color, power)"
    echo "  - switch (power control, energy monitoring)"
    echo "  - climate (temperature control, HVAC)"
    echo
}

# 主執行流程
main() {
    echo "RTK MQTT Wrapper Demo Script"
    echo "============================"
    echo
    
    # 基本檢查
    check_mqtt_tools
    check_mqtt_broker
    show_config
    
    # 測試基本功能
    test_wrapper_basic
    
    # 顯示支援的格式
    show_supported_formats
    
    # 示範訊息轉換
    demo_uplink_transformation
    demo_downlink_transformation
    
    # 如果可用，進行實際測試
    if [ "$1" = "--live" ]; then
        run_live_test
    else
        echo "To run live MQTT testing, use: $0 --live"
        echo "Make sure MQTT broker is running: mosquitto -p 1883 -v"
    fi
    
    echo
    echo -e "${GREEN}Demo completed!${NC}"
    echo "Next steps:"
    echo "1. Start a local MQTT broker: mosquitto -p 1883 -v"
    echo "2. Run the wrapper: ../build_dir/rtk_wrapper --config configs/wrapper.yaml"
    echo "3. Send test messages using mosquitto_pub"
    echo "4. Monitor RTK messages using mosquitto_sub"
}

# 執行主函數
main "$@"