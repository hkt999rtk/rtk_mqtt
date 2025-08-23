#!/bin/bash

# RTK Home Network Simulator Demo Script
# 演示模擬器的基本功能

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 函數定義
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 檢查必要條件
check_requirements() {
    log_info "檢查必要條件..."
    
    # 檢查 RTK 模擬器
    if [ ! -f "./build/rtk-simulator" ]; then
        log_error "RTK 模擬器未找到，請先執行 'make build'"
        exit 1
    fi
    
    # 檢查 MQTT Broker (可選)
    if command -v mosquitto_pub &> /dev/null; then
        log_success "發現 Mosquitto MQTT 工具"
        MQTT_AVAILABLE=true
    else
        log_warning "未找到 Mosquitto MQTT 工具，將跳過 MQTT 測試"
        MQTT_AVAILABLE=false
    fi
    
    log_success "必要條件檢查完成"
}

# 生成配置檔案
generate_configs() {
    log_info "生成配置檔案..."
    
    mkdir -p configs
    
    # 生成基本家庭配置
    ./build/rtk-simulator generate home_basic -o configs/demo_home_basic.yaml 2>/dev/null || {
        log_warning "配置生成功能尚未完全實作，使用現有配置"
    }
    
    log_success "配置檔案生成完成"
}

# 驗證配置
validate_configs() {
    log_info "驗證配置檔案..."
    
    for config in configs/*.yaml; do
        if [ -f "$config" ]; then
            log_info "驗證 $config..."
            if ./build/rtk-simulator validate "$config" > /dev/null 2>&1; then
                log_success "✓ $config 驗證通過"
            else
                log_warning "✗ $config 驗證失敗"
            fi
        fi
    done
}

# 演示模擬器功能
demo_simulator() {
    log_info "演示模擬器功能..."
    
    echo
    log_info "=== RTK Home Network Simulator Demo ==="
    echo
    
    # 1. 顯示版本資訊
    log_info "1. 版本資訊:"
    ./build/rtk-simulator --version
    echo
    
    # 2. 顯示可用命令
    log_info "2. 可用命令:"
    ./build/rtk-simulator --help | grep -A 10 "Available Commands:"
    echo
    
    # 3. 配置驗證演示
    log_info "3. 配置驗證演示:"
    if [ -f "configs/home_basic.yaml" ]; then
        ./build/rtk-simulator validate configs/home_basic.yaml
    else
        log_warning "home_basic.yaml 配置檔案不存在"
    fi
    echo
    
    # 4. 乾運行模式演示
    log_info "4. 乾運行模式演示 (5秒):"
    timeout 5s ./build/rtk-simulator run --dry-run --verbose || true
    echo
    
    # 5. 實際模擬演示 (如果有 MQTT)
    if [ "$MQTT_AVAILABLE" = true ]; then
        log_info "5. 啟動短期模擬 (10秒):"
        log_warning "注意：需要 MQTT Broker 運行在 localhost:1883"
        echo "如果您有 MQTT Broker，模擬器將嘗試連接..."
        echo "取消模擬請按 Ctrl+C"
        echo
        
        # 啟動 10 秒的模擬
        timeout 10s ./build/rtk-simulator run --verbose 2>/dev/null || {
            log_warning "模擬因為 MQTT 連接問題而停止，這是正常的"
        }
    else
        log_info "5. 跳過實際模擬 (需要 MQTT Broker)"
    fi
    
    echo
    log_success "演示完成！"
}

# 顯示後續步驟
show_next_steps() {
    echo
    log_info "=== 後續步驟 ==="
    echo
    echo "1. 安裝 MQTT Broker (推薦 Mosquitto):"
    echo "   macOS: brew install mosquitto"
    echo "   Ubuntu: sudo apt-get install mosquitto mosquitto-clients"
    echo "   啟動: mosquitto -v"
    echo
    echo "2. 運行完整模擬:"
    echo "   ./build/rtk-simulator run -c configs/home_basic.yaml"
    echo
    echo "3. 監控 MQTT 訊息:"
    echo "   mosquitto_sub -h localhost -t 'rtk/v1/+/+/+/+'"
    echo
    echo "4. 生成自定義配置:"
    echo "   ./build/rtk-simulator generate home_advanced -o my_config.yaml"
    echo
    echo "5. 查看更多建構選項:"
    echo "   make help"
    echo
    echo "6. 查看完整文檔:"
    echo "   less README.md"
    echo
}

# 主執行流程
main() {
    echo
    log_info "=== RTK Home Network Simulator Demo ==="
    log_info "展示 RTK 家用網路環境模擬器的基本功能"
    echo
    
    check_requirements
    echo
    
    generate_configs
    echo
    
    validate_configs
    echo
    
    demo_simulator
    
    show_next_steps
}

# 錯誤處理
trap 'log_error "腳本執行中止"; exit 1' ERR

# 執行主函數
main "$@"