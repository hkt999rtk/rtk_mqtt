#!/bin/bash

# 簡化的 ARM 靜態庫構建腳本
# 專注於構建核心 RTK MQTT Framework 庫

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="$PROJECT_ROOT/build-arm-minimal"
OUTPUT_DIR="$PROJECT_ROOT/dist-arm"

echo "=== RTK MQTT Framework ARM 靜態庫構建 ==="
echo "項目根目錄: $PROJECT_ROOT"
echo "構建目錄: $BUILD_DIR"
echo "輸出目錄: $OUTPUT_DIR"

# 清理並創建目錄
rm -rf "$BUILD_DIR" "$OUTPUT_DIR"
mkdir -p "$BUILD_DIR" "$OUTPUT_DIR"
cd "$BUILD_DIR"

# ARM 工具鏈設定
ARM_CC="arm-none-eabi-gcc"
ARM_AR="arm-none-eabi-ar"

# ARM 編譯標志
ARM_CFLAGS="-mcpu=cortex-m4 -mthumb -mfloat-abi=hard -mfpu=fpv4-sp-d16"
ARM_CFLAGS="$ARM_CFLAGS -ffunction-sections -fdata-sections"
ARM_CFLAGS="$ARM_CFLAGS -Wall -Wextra -O2"
ARM_CFLAGS="$ARM_CFLAGS -DRTK_PLATFORM_FREERTOS=1 -DRTK_TARGET_ARM=1"
ARM_CFLAGS="$ARM_CFLAGS -DRTK_USE_LIGHTWEIGHT_JSON=1 -DRTK_MINIMAL_MEMORY=1"

# 檢查工具鏈
if ! command -v $ARM_CC >/dev/null 2>&1; then
    echo "❌ ARM 工具鏈 $ARM_CC 不可用"
    exit 1
fi

echo "✅ 使用 ARM 工具鏈: $ARM_CC"

# RTK Framework 核心源碼
FRAMEWORK_SOURCES=(
    "$PROJECT_ROOT/framework/src/mqtt_client.c"
    "$PROJECT_ROOT/framework/src/topic_builder.c"
    "$PROJECT_ROOT/framework/src/message_codec.c"
    "$PROJECT_ROOT/framework/src/schema_validator.c"
    "$PROJECT_ROOT/framework/src/plugin_manager.c"
    "$PROJECT_ROOT/framework/src/freertos_compat.c"
    "$PROJECT_ROOT/framework/src/json_pool.c"
)

# 包含路徑
INCLUDE_DIRS=(
    "-I$PROJECT_ROOT/framework/include"
    "-I$PROJECT_ROOT/external/cjson"
)

echo "編譯 RTK Framework 核心..."

# 編譯各個源文件
OBJECT_FILES=()
for src_file in "${FRAMEWORK_SOURCES[@]}"; do
    if [[ -f "$src_file" ]]; then
        obj_file="$(basename "$src_file" .c).o"
        echo "編譯: $(basename "$src_file")"
        $ARM_CC $ARM_CFLAGS "${INCLUDE_DIRS[@]}" -c "$src_file" -o "$obj_file"
        OBJECT_FILES+=("$obj_file")
    else
        echo "⚠️  源文件不存在: $src_file"
    fi
done

# 編譯簡化的 cJSON (僅必要部分)
if [[ -f "$PROJECT_ROOT/external/cjson/cJSON.c" ]]; then
    echo "編譯簡化的 cJSON..."
    # 使用特殊標志來避免標準庫依賴
    $ARM_CC $ARM_CFLAGS "${INCLUDE_DIRS[@]}" -DCJSON_HIDE_SYMBOLS -c "$PROJECT_ROOT/external/cjson/cJSON.c" -o "cjson.o"
    OBJECT_FILES+=("cjson.o")
fi

# 創建靜態庫
if [[ ${#OBJECT_FILES[@]} -gt 0 ]]; then
    echo "創建 ARM 靜態庫..."
    $ARM_AR rcs "librtk_mqtt_framework_arm.a" "${OBJECT_FILES[@]}"
    
    # 復制到輸出目錄
    cp "librtk_mqtt_framework_arm.a" "$OUTPUT_DIR/"
    cp -r "$PROJECT_ROOT/framework/include/" "$OUTPUT_DIR/include/"
    
    # 顯示庫信息
    echo ""
    echo "✅ ARM 靜態庫構建成功"
    echo "輸出文件: $OUTPUT_DIR/librtk_mqtt_framework_arm.a"
    
    # 顯示庫大小和內容
    ls -la "$OUTPUT_DIR/librtk_mqtt_framework_arm.a"
    echo ""
    echo "庫內容:"
    $ARM_AR -t "$OUTPUT_DIR/librtk_mqtt_framework_arm.a"
    
    # 創建使用說明
    cat > "$OUTPUT_DIR/README_ARM.md" << EOF
# RTK MQTT Framework ARM 靜態庫

## 文件說明

- \`librtk_mqtt_framework_arm.a\` - ARM Cortex-M4 靜態庫
- \`include/\` - 頭文件目錄

## 編譯選項

此庫使用以下編譯選項構建：
- CPU: ARM Cortex-M4
- FPU: Hard float (fpv4-sp-d16)
- 優化: -O2
- 特殊宏: RTK_PLATFORM_FREERTOS, RTK_TARGET_ARM, RTK_USE_LIGHTWEIGHT_JSON

## 使用方法

在您的 ARM 項目中：

\`\`\`makefile
# 連結庫
LDFLAGS += -L/path/to/rtk-framework/dist-arm -lrtk_mqtt_framework_arm

# 包含頭文件
CFLAGS += -I/path/to/rtk-framework/dist-arm/include
\`\`\`

## 系統需求

- FreeRTOS 操作系統
- ARM Cortex-M4 微控制器
- 支持硬浮點運算

## 零外部依賴

此庫已整合所有必要組件，無需額外安裝 MQTT 或 JSON 處理庫。
EOF

    echo "📚 使用說明已生成: $OUTPUT_DIR/README_ARM.md"
    echo ""
    echo "🎉 ARM 靜態庫構建完成！"
    
else
    echo "❌ 沒有找到可編譯的源文件"
    exit 1
fi