#!/bin/bash

# RTK MQTT Framework 使用者範例測試腳本
# 這個腳本檢查所有範例的基本結構和 Makefile 語法

echo "RTK MQTT Framework 使用者範例測試"
echo "================================="
echo

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

success_count=0
total_count=0

# 測試函式
test_example() {
    local example_dir="$1"
    local example_name="$2"
    
    echo -e "${BLUE}=== 測試 $example_name ===${NC}"
    
    if [ ! -d "$example_dir" ]; then
        echo -e "${RED}❌ 目錄不存在: $example_dir${NC}"
        return 1
    fi
    
    cd "$example_dir"
    
    # 檢查必要檔案
    local required_files=("Makefile" "README.md")
    local optional_files=()
    
    # 根據範例類型設定檔案需求
    case "$example_name" in
        "01_hello_world")
            required_files+=("main.c")
            optional_files+=("config.json")
            ;;
        "02_basic_sensor")
            required_files+=("sensor.c" "config.json")
            ;;
        "03_complete_device")
            required_files+=("device.c" "config.json")
            ;;
        "04_cross_platform")
            required_files+=("cross_platform_device.c")
            ;;
    esac
    
    # 檢查必要檔案
    local files_ok=true
    for file in "${required_files[@]}"; do
        if [ -f "$file" ]; then
            echo -e "  ${GREEN}✅ $file 存在${NC}"
        else
            echo -e "  ${RED}❌ $file 缺失${NC}"
            files_ok=false
        fi
    done
    
    # 檢查選用檔案
    for file in "${optional_files[@]}"; do
        if [ -f "$file" ]; then
            echo -e "  ${YELLOW}📄 $file 存在 (選用)${NC}"
        fi
    done
    
    # 檢查 Makefile 語法
    if [ -f "Makefile" ]; then
        if make -n help >/dev/null 2>&1; then
            echo -e "  ${GREEN}✅ Makefile 語法正確${NC}"
        else
            echo -e "  ${YELLOW}⚠️  Makefile 語法檢查警告${NC}"
        fi
        
        # 檢查 Makefile 中的目標
        local targets=("all" "clean" "help")
        for target in "${targets[@]}"; do
            if grep -q "^$target:" Makefile; then
                echo -e "  ${GREEN}✅ Makefile 包含 '$target' 目標${NC}"
            else
                echo -e "  ${YELLOW}⚠️  Makefile 缺少 '$target' 目標${NC}"
            fi
        done
    fi
    
    # 檢查 README.md 內容
    if [ -f "README.md" ]; then
        local readme_size=$(wc -l < README.md)
        if [ "$readme_size" -gt 50 ]; then
            echo -e "  ${GREEN}✅ README.md 內容豐富 ($readme_size 行)${NC}"
        else
            echo -e "  ${YELLOW}⚠️  README.md 內容較少 ($readme_size 行)${NC}"
        fi
        
        # 檢查 README 是否包含關鍵章節
        local sections=("學習目標" "快速開始" "故障排除")
        for section in "${sections[@]}"; do
            if grep -q "$section" README.md; then
                echo -e "  ${GREEN}✅ README 包含 '$section' 章節${NC}"
            else
                echo -e "  ${YELLOW}⚠️  README 缺少 '$section' 章節${NC}"
            fi
        done
    fi
    
    # 檢查源代碼大小
    if ls *.c >/dev/null 2>&1; then
        local code_files=(*.c)
        for code_file in "${code_files[@]}"; do
            local lines=$(wc -l < "$code_file")
            case "$example_name" in
                "01_hello_world")
                    if [ "$lines" -le 50 ]; then
                        echo -e "  ${GREEN}✅ $code_file 大小適中 ($lines 行，符合簡單範例)${NC}"
                    else
                        echo -e "  ${YELLOW}⚠️  $code_file 對簡單範例來說可能過長 ($lines 行)${NC}"
                    fi
                    ;;
                "02_basic_sensor")
                    if [ "$lines" -ge 100 ] && [ "$lines" -le 300 ]; then
                        echo -e "  ${GREEN}✅ $code_file 大小適中 ($lines 行，符合中級範例)${NC}"
                    else
                        echo -e "  ${YELLOW}⚠️  $code_file 大小 ($lines 行) 可能不符合中級範例預期${NC}"
                    fi
                    ;;
                "03_complete_device"|"04_cross_platform")
                    if [ "$lines" -ge 300 ]; then
                        echo -e "  ${GREEN}✅ $code_file 大小適中 ($lines 行，符合高級範例)${NC}"
                    else
                        echo -e "  ${YELLOW}⚠️  $code_file 對高級範例來說可能過短 ($lines 行)${NC}"
                    fi
                    ;;
            esac
        done
    fi
    
    # 檢查配置檔案 (如果存在)
    if [ -f "config.json" ]; then
        if python3 -m json.tool config.json >/dev/null 2>&1; then
            echo -e "  ${GREEN}✅ config.json 格式正確${NC}"
        else
            echo -e "  ${RED}❌ config.json 格式錯誤${NC}"
            files_ok=false
        fi
    fi
    
    cd ..
    
    if [ "$files_ok" = true ]; then
        echo -e "  ${GREEN}🎉 $example_name 基本結構檢查通過${NC}"
        ((success_count++))
    else
        echo -e "  ${RED}💥 $example_name 基本結構檢查失敗${NC}"
    fi
    
    ((total_count++))
    echo
}

# 測試所有範例
examples=(
    "01_hello_world:Hello World 範例"
    "02_basic_sensor:基本感測器範例"  
    "03_complete_device:完整設備範例"
    "04_cross_platform:跨平台範例"
)

for example in "${examples[@]}"; do
    IFS=':' read -r dir name <<< "$example"
    test_example "$dir" "$name"
done

# 檢查總覽 README
echo -e "${BLUE}=== 檢查總覽文檔 ===${NC}"
if [ -f "README.md" ]; then
    readme_size=$(wc -l < README.md)
    if [ "$readme_size" -gt 100 ]; then
        echo -e "${GREEN}✅ 總覽 README.md 內容豐富 ($readme_size 行)${NC}"
        ((success_count++))
    else
        echo -e "${YELLOW}⚠️  總覽 README.md 內容可能不夠詳細 ($readme_size 行)${NC}"
    fi
    ((total_count++))
else
    echo -e "${RED}❌ 總覽 README.md 不存在${NC}"
    ((total_count++))
fi

# 檢查目錄結構
echo -e "${BLUE}=== 檢查目錄結構 ===${NC}"
expected_dirs=("01_hello_world" "02_basic_sensor" "03_complete_device" "04_cross_platform")
structure_ok=true

for dir in "${expected_dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✅ $dir 目錄存在${NC}"
    else
        echo -e "${RED}❌ $dir 目錄缺失${NC}"
        structure_ok=false
    fi
done

if [ "$structure_ok" = true ]; then
    echo -e "${GREEN}🎉 目錄結構檢查通過${NC}"
    ((success_count++))
else
    echo -e "${RED}💥 目錄結構檢查失敗${NC}"
fi
((total_count++))

# 總結
echo
echo -e "${BLUE}=== 測試總結 ===${NC}"
echo "總測試項目: $total_count"
echo "通過項目: $success_count"
echo "失敗項目: $((total_count - success_count))"

if [ "$success_count" -eq "$total_count" ]; then
    echo -e "${GREEN}🎉 所有測試通過！範例結構完整。${NC}"
    exit 0
elif [ "$success_count" -gt $((total_count / 2)) ]; then
    echo -e "${YELLOW}⚠️  大部分測試通過，但仍有改進空間。${NC}"
    exit 1
else
    echo -e "${RED}💥 多項測試失敗，需要修復。${NC}"
    exit 2
fi