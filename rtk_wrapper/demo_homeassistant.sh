#!/bin/bash

# Home Assistant Wrapper 示範腳本

set -e

echo "=== Home Assistant Wrapper Demo ==="
echo

# 顏色定義
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 檢查是否已建置
if [ ! -f "../build_dir/rtk_wrapper" ]; then
    echo -e "${RED}Error: rtk_wrapper binary not found. Please run 'make build' first.${NC}"
    exit 1
fi

# Home Assistant 範例訊息
demo_ha_uplink_messages() {
    echo "=== Home Assistant Uplink Messages Demo (HA → RTK) ==="
    echo
    
    local ha_messages=(
        "homeassistant/light/living_room/state|{\"state\":\"on\",\"brightness\":200,\"color_temp\":300,\"rgb_color\":[255,128,0]}|Living Room Light Status"
        "homeassistant/switch/kitchen_outlet/state|{\"state\":\"on\",\"power\":150.5,\"energy\":2.5,\"voltage\":230.1,\"current\":0.65}|Kitchen Outlet with Energy Monitoring"
        "homeassistant/sensor/bedroom_temp/state|{\"state\":\"23.5\",\"unit_of_measurement\":\"°C\",\"device_class\":\"temperature\"}|Bedroom Temperature Sensor"
        "homeassistant/sensor/bathroom_humidity/state|{\"state\":\"65.2\",\"unit_of_measurement\":\"%\",\"device_class\":\"humidity\"}|Bathroom Humidity Sensor"
        "homeassistant/climate/main_thermostat/state|{\"current_temperature\":22.5,\"temperature\":24.0,\"mode\":\"heat\",\"fan_mode\":\"auto\"}|Main Thermostat Status"
        "homeassistant/cover/living_room_blinds/state|{\"state\":\"open\",\"position\":75,\"tilt_position\":45}|Living Room Blinds Position"
        "homeassistant/binary_sensor/front_door/state|{\"state\":\"off\",\"device_class\":\"door\"}|Front Door Sensor (Closed)"
        "homeassistant/fan/bedroom_ceiling/state|{\"state\":\"on\",\"speed\":\"medium\",\"oscillating\":true}|Bedroom Ceiling Fan"
        "homeassistant/light/office/desk_lamp/state|{\"state\":\"on\",\"brightness\":180,\"color_temp\":250}|Office Desk Lamp (Nested Location)"
    )
    
    for msg in "${ha_messages[@]}"; do
        IFS='|' read -r topic payload description <<< "$msg"
        
        echo -e "${BLUE}Example: $description${NC}"
        echo -e "${YELLOW}Original Home Assistant Format:${NC}"
        echo "  Topic: $topic"
        echo "  Payload: $payload"
        echo
        
        # 解析設備資訊
        IFS='/' read -r prefix device_class device_name message_type <<< "$topic"
        if [[ "$topic" =~ homeassistant/[^/]+/[^/]+/[^/]+/state ]]; then
            # 巢狀格式: homeassistant/light/office/desk_lamp/state
            IFS='/' read -r prefix device_class location device_name message_type <<< "$topic"
            rtk_device_id="${location}_${device_name}"
        else
            rtk_device_id="$device_name"
        fi
        
        echo -e "${GREEN}Expected RTK Format:${NC}"
        rtk_topic="rtk/v1/home/main/$rtk_device_id/$message_type"
        echo "  Topic: $rtk_topic"
        echo "  Schema: $message_type/1.0"
        echo "  Device ID: $rtk_device_id"
        echo "  Device Class: $device_class"
        
        # 顯示轉換後的 payload 範例
        case $device_class in
            "light")
                echo "  RTK Payload: {\"health\":\"ok\",\"power_state\":\"on\",\"brightness\":78,\"color_temperature\":300,\"rgb_color\":[255,128,0]}"
                ;;
            "switch")
                echo "  RTK Payload: {\"health\":\"ok\",\"power_state\":\"on\",\"power_consumption\":150.5,\"energy_consumption\":2.5,\"voltage\":230.1,\"current\":0.65}"
                ;;
            "sensor")
                echo "  RTK Payload: {\"health\":\"ok\",\"temperature\":23.5,\"sensor_state\":\"23.5\"}"
                ;;
            "climate")
                echo "  RTK Payload: {\"health\":\"ok\",\"current_temperature\":22.5,\"target_temperature\":24.0,\"hvac_mode\":\"heat\",\"fan_mode\":\"auto\"}"
                ;;
            "cover")
                echo "  RTK Payload: {\"health\":\"ok\",\"cover_state\":\"open\",\"position\":75,\"tilt_position\":45}"
                ;;
            "binary_sensor")
                echo "  RTK Payload: {\"health\":\"ok\",\"binary_state\":\"off\",\"sensor_type\":\"door\"}"
                ;;
            "fan")
                echo "  RTK Payload: {\"health\":\"ok\",\"power_state\":\"on\",\"fan_speed\":\"medium\",\"oscillating\":true}"
                ;;
        esac
        echo
        echo "---"
    done
}

# Home Assistant 下行訊息示範
demo_ha_downlink_messages() {
    echo "=== Home Assistant Downlink Messages Demo (RTK → HA) ==="
    echo
    
    local rtk_commands=(
        "rtk/v1/home/main/living_room_light/cmd/req|{\"schema\":\"cmd.turn_on/1.0\",\"ts\":$(date +%s)000,\"device_id\":\"living_room_light\",\"payload\":{\"command\":\"turn_on\",\"brightness\":80}}|Turn on living room light with 80% brightness"
        "rtk/v1/home/main/kitchen_outlet/cmd/req|{\"schema\":\"cmd.turn_off/1.0\",\"ts\":$(date +%s)000,\"device_id\":\"kitchen_outlet\",\"payload\":{\"command\":\"turn_off\"}}|Turn off kitchen outlet"
        "rtk/v1/home/main/main_thermostat/cmd/req|{\"schema\":\"cmd.set_temperature/1.0\",\"ts\":$(date +%s)000,\"device_id\":\"main_thermostat\",\"payload\":{\"command\":\"set_temperature\",\"temperature\":22.0}}|Set thermostat to 22°C"
        "rtk/v1/home/main/living_room_blinds/cmd/req|{\"schema\":\"cmd.set_position/1.0\",\"ts\":$(date +%s)000,\"device_id\":\"living_room_blinds\",\"payload\":{\"command\":\"set_position\",\"position\":50}}|Set blinds to 50% position"
        "rtk/v1/home/main/office_desk_lamp/cmd/req|{\"schema\":\"cmd.set_brightness/1.0\",\"ts\":$(date +%s)000,\"device_id\":\"office_desk_lamp\",\"payload\":{\"command\":\"set_brightness\",\"brightness\":60}}|Set desk lamp brightness to 60%"
    )
    
    for cmd in "${rtk_commands[@]}"; do
        IFS='|' read -r topic payload description <<< "$cmd"
        
        echo -e "${BLUE}Example: $description${NC}"
        echo -e "${YELLOW}Original RTK Command:${NC}"
        echo "  Topic: $topic"
        echo "  Payload: $payload"
        echo
        
        # 解析設備 ID 並推斷設備類型
        device_id=$(echo "$topic" | cut -d'/' -f5)
        
        # 推斷 HA 設備類型
        if [[ $device_id == *"light"* ]] || [[ $device_id == *"lamp"* ]]; then
            device_class="light"
        elif [[ $device_id == *"switch"* ]] || [[ $device_id == *"outlet"* ]]; then
            device_class="switch"
        elif [[ $device_id == *"thermostat"* ]] || [[ $device_id == *"climate"* ]]; then
            device_class="climate"
        elif [[ $device_id == *"blind"* ]] || [[ $device_id == *"cover"* ]]; then
            device_class="cover"
        else
            device_class="switch"
        fi
        
        ha_topic="homeassistant/$device_class/$device_id/set"
        
        echo -e "${GREEN}Expected Home Assistant Command:${NC}"
        echo "  Topic: $ha_topic"
        
        # 根據命令類型顯示預期的 HA payload
        if [[ $payload == *"turn_on"* ]]; then
            if [[ $payload == *"brightness"* ]]; then
                echo "  HA Payload: {\"state\":\"ON\",\"brightness\":204}"  # 80% * 255
            else
                echo "  HA Payload: {\"state\":\"ON\"}"
            fi
        elif [[ $payload == *"turn_off"* ]]; then
            echo "  HA Payload: {\"state\":\"OFF\"}"
        elif [[ $payload == *"set_temperature"* ]]; then
            echo "  HA Payload: {\"temperature\":22.0}"
        elif [[ $payload == *"set_position"* ]]; then
            echo "  HA Payload: {\"position\":50}"
        elif [[ $payload == *"set_brightness"* ]]; then
            echo "  HA Payload: {\"brightness\":153}"  # 60% * 255
        fi
        
        echo
        echo "---"
    done
}

# 顯示支援的 HA 設備類型
show_ha_device_types() {
    echo "=== Supported Home Assistant Device Types ==="
    echo
    
    echo -e "${BLUE}Light (homeassistant/light/*/*)${NC}"
    echo "  • State: on/off"
    echo "  • Brightness: 0-255 (converted to 0-100% in RTK)"
    echo "  • Color temperature: mireds"
    echo "  • RGB color: [r,g,b] array"
    echo "  • Commands: turn_on, turn_off, set_brightness"
    echo
    
    echo -e "${BLUE}Switch (homeassistant/switch/*/*)${NC}" 
    echo "  • State: on/off"
    echo "  • Power monitoring: power, energy, voltage, current"
    echo "  • Commands: turn_on, turn_off"
    echo
    
    echo -e "${BLUE}Sensor (homeassistant/sensor/*/*)${NC}"
    echo "  • Temperature, humidity, pressure sensors"
    echo "  • Battery level, illuminance"
    echo "  • Generic value and state"
    echo "  • Unit conversion support"
    echo
    
    echo -e "${BLUE}Climate (homeassistant/climate/*/*)${NC}"
    echo "  • Current and target temperature"
    echo "  • HVAC modes: heat, cool, auto, off"
    echo "  • Fan modes: auto, low, medium, high"
    echo "  • Commands: set_temperature, set_mode"
    echo
    
    echo -e "${BLUE}Cover (homeassistant/cover/*/*)${NC}"
    echo "  • State: open, closed, opening, closing"
    echo "  • Position: 0-100%"
    echo "  • Tilt position for blinds"
    echo "  • Commands: open, close, stop, set_position"
    echo
    
    echo -e "${BLUE}Binary Sensor (homeassistant/binary_sensor/*/*)${NC}"
    echo "  • State: on/off"
    echo "  • Device classes: door, window, motion, occupancy"
    echo "  • Read-only sensors"
    echo
    
    echo -e "${BLUE}Fan (homeassistant/fan/*/*)${NC}"
    echo "  • State: on/off"
    echo "  • Speed: low, medium, high"
    echo "  • Oscillating: true/false"
    echo "  • Commands: turn_on, turn_off, set_speed"
    echo
    
    echo -e "${BLUE}Lock (homeassistant/lock/*/*)${NC}"
    echo "  • State: locked/unlocked"
    echo "  • Commands: lock, unlock"
    echo
}

# 顯示 HA topic 模式
show_ha_topic_patterns() {
    echo "=== Home Assistant Topic Patterns ==="
    echo
    
    echo -e "${YELLOW}Uplink Patterns (HA → RTK):${NC}"
    echo "  1. homeassistant/{device_class}/{device_name}/state (Priority: 100)"
    echo "     Example: homeassistant/light/living_room/state"
    echo
    echo "  2. homeassistant/{device_class}/{device_name}/attributes (Priority: 95)"
    echo "     Example: homeassistant/light/living_room/attributes"
    echo
    echo "  3. homeassistant/{device_class}/{location}/{device_name}/state (Priority: 90)"
    echo "     Example: homeassistant/light/office/desk_lamp/state"
    echo
    echo "  4. homeassistant/{device_class}/{device_name}/availability (Priority: 85)"
    echo "     Example: homeassistant/sensor/temp_sensor/availability"
    echo
    
    echo -e "${YELLOW}Downlink Patterns (RTK → HA):${NC}"
    echo "  1. rtk/v1/{tenant}/{site}/{device_id}/cmd/req (Priority: 100)"
    echo "     Target: homeassistant/{inferred_class}/{device_id}/set"
    echo
}

# 測試連接性
test_ha_connectivity() {
    echo "=== Testing Home Assistant Connectivity ==="
    echo
    
    if command -v mosquitto_pub >/dev/null 2>&1; then
        echo -e "${GREEN}MQTT tools available${NC}"
        
        # 測試是否有 MQTT broker
        if timeout 3 mosquitto_pub -h localhost -t "test/ha_wrapper" -m "test" >/dev/null 2>&1; then
            echo -e "${GREEN}MQTT broker is accessible${NC}"
            
            echo "You can test Home Assistant wrapper with these commands:"
            echo
            echo "1. Start the wrapper:"
            echo "   ../build_dir/rtk_wrapper --config configs/wrapper.yaml"
            echo
            echo "2. In another terminal, send HA light state:"
            echo "   mosquitto_pub -h localhost -t 'homeassistant/light/test_light/state' -m '{\"state\":\"on\",\"brightness\":200}'"
            echo
            echo "3. Listen for RTK messages:"
            echo "   mosquitto_sub -h localhost -t 'rtk/v1/+/+/+/+'"
            echo
            echo "4. Send RTK command:"
            echo "   mosquitto_pub -h localhost -t 'rtk/v1/home/main/test_light/cmd/req' -m '{\"command\":\"turn_off\"}'"
            echo
            echo "5. Listen for HA commands:"
            echo "   mosquitto_sub -h localhost -t 'homeassistant/+/+/set'"
            
        else
            echo -e "${YELLOW}MQTT broker not responding. Start with: mosquitto -p 1883 -v${NC}"
        fi
    else
        echo -e "${YELLOW}MQTT tools not found. Install mosquitto-clients for testing.${NC}"
    fi
}

# 主函數
main() {
    echo "Home Assistant MQTT Wrapper Demo"
    echo "==============================="
    echo
    
    show_ha_device_types
    echo
    show_ha_topic_patterns
    echo
    demo_ha_uplink_messages
    echo
    demo_ha_downlink_messages
    echo
    test_ha_connectivity
    echo
    
    echo -e "${GREEN}Home Assistant Wrapper Demo completed!${NC}"
    echo
    echo "Key Features Demonstrated:"
    echo "✓ Support for 8 major HA device types"
    echo "✓ Bidirectional message transformation"
    echo "✓ Flexible topic pattern matching"
    echo "✓ Unit conversions (brightness 0-255 ↔ 0-100%)"
    echo "✓ Nested device location support"
    echo "✓ Intelligent device class inference"
    echo "✓ Comprehensive payload validation"
}

# 執行主函數
main "$@"