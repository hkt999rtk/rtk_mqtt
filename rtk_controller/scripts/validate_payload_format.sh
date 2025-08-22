#!/bin/bash

# RTK MQTT Payload Format Validation Script
# 檢查所有文檔中的 JSON 範例是否符合 payload 格式規範

set -e

DOCS_DIR="docs/developers/devices"
CORE_DOCS_DIR="docs/developers/core"
ERRORS=0
WARNINGS=0
TOTAL_EXAMPLES=0

echo "🔍 RTK MQTT Payload Format Validation"
echo "======================================"

# 顏色定義
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 驗證單個文件
validate_file() {
    local file=$1
    local filename=$(basename "$file")
    
    echo -e "\n${BLUE}📄 檢查文件: $filename${NC}"
    
    # 統計 JSON 範例數量
    local total_schemas=$(grep -c '"schema":' "$file" 2>/dev/null || echo "0")
    TOTAL_EXAMPLES=$((TOTAL_EXAMPLES + total_schemas))
    
    if [ "$total_schemas" -eq 0 ]; then
        echo "  ℹ️  沒有找到 JSON 範例"
        return
    fi
    
    echo "  📊 發現 $total_schemas 個 JSON 範例"
    
    # 檢查 payload 結構
    local schemas_with_payload=$(grep -A 5 '"schema":' "$file" | grep -B 5 '"payload":' | grep -c '"schema":' 2>/dev/null || echo "0")
    local schemas_with_device_id=$(grep -A 5 '"schema":' "$file" | grep -B 5 '"device_id":' | grep -c '"schema":' 2>/dev/null || echo "0")
    
    # 計算符合率
    local payload_rate=0
    local device_id_rate=0
    if [ "$total_schemas" -gt 0 ]; then
        payload_rate=$((schemas_with_payload * 100 / total_schemas))
        device_id_rate=$((schemas_with_device_id * 100 / total_schemas))
    fi
    
    echo "  📋 Payload 包裝: $schemas_with_payload/$total_schemas (${payload_rate}%)"
    echo "  🏷️  Device ID: $schemas_with_device_id/$total_schemas (${device_id_rate}%)"
    
    # 狀態評估
    if [ "$payload_rate" -eq 100 ] && [ "$device_id_rate" -eq 100 ]; then
        echo -e "  ${GREEN}✅ 完美 - 所有範例符合規範${NC}"
    elif [ "$payload_rate" -ge 90 ] && [ "$device_id_rate" -ge 90 ]; then
        echo -e "  ${GREEN}✅ 優秀 - 絕大多數範例符合規範${NC}"
        WARNINGS=$((WARNINGS + 1))
    elif [ "$payload_rate" -ge 80 ] && [ "$device_id_rate" -ge 80 ]; then
        echo -e "  ${YELLOW}⚠️  良好 - 大部分範例符合規範${NC}"
        WARNINGS=$((WARNINGS + 1))
    else
        echo -e "  ${RED}❌ 需要改進 - 許多範例不符合規範${NC}"
        ERRORS=$((ERRORS + 1))
    fi
    
    # 檢查具體的格式問題
    local problematic_lines=$(grep -n -A 10 '"schema":' "$file" | grep -B 3 -A 7 '"health":\|"cpu_usage":\|"event_type":' | grep -v '"payload":' | grep -E '^[0-9]+[-:].*"(health|cpu_usage|event_type)":' | head -3)
    
    if [ ! -z "$problematic_lines" ]; then
        echo -e "  ${YELLOW}⚠️  發現可能的格式問題:${NC}"
        echo "$problematic_lines" | while read line; do
            echo "    - $line"
        done
    fi
}

# 驗證設備整合文檔
echo -e "\n${BLUE}🏠 檢查設備整合文檔${NC}"
if [ -d "$DOCS_DIR" ]; then
    for file in "$DOCS_DIR"/*.md; do
        if [ -f "$file" ]; then
            validate_file "$file"
        fi
    done
else
    echo -e "${RED}❌ 設備整合文檔目錄不存在: $DOCS_DIR${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 驗證核心文檔
echo -e "\n${BLUE}🔧 檢查核心文檔${NC}"
if [ -d "$CORE_DOCS_DIR" ]; then
    for file in "$CORE_DOCS_DIR"/*.md; do
        if [ -f "$file" ] && grep -q '"schema":' "$file"; then
            validate_file "$file"
        fi
    done
else
    echo -e "${YELLOW}⚠️  核心文檔目錄不存在: $CORE_DOCS_DIR${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

# 生成總結報告
echo -e "\n${BLUE}📊 驗證總結${NC}"
echo "===================="
echo "📈 總計檢查: $TOTAL_EXAMPLES 個 JSON 範例"
echo "🎯 完美文檔: $((5 - ERRORS - WARNINGS)) 個"
echo "⚠️  需要注意: $WARNINGS 個"
echo "❌ 需要修正: $ERRORS 個"

# 返回狀態碼
if [ "$ERRORS" -eq 0 ] && [ "$WARNINGS" -eq 0 ]; then
    echo -e "\n${GREEN}🎉 所有文檔都符合 RTK MQTT Payload 格式規範！${NC}"
    exit 0
elif [ "$ERRORS" -eq 0 ]; then
    echo -e "\n${GREEN}✅ 驗證通過 - 大部分文檔符合規範，有少量警告${NC}"
    exit 0
else
    echo -e "\n${RED}❌ 驗證失敗 - 發現 $ERRORS 個嚴重問題需要修正${NC}"
    exit 1
fi