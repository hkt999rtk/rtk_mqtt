cmake_minimum_required(VERSION 3.10)

# RTK MQTT Framework 範例專案
project(rtk_mqtt_framework_examples VERSION 1.0.0 LANGUAGES C)

# 設置 C 標準
set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)

message(STATUS "Building RTK MQTT Framework Examples")
message(STATUS "  Framework version: ${PROJECT_VERSION}")
message(STATUS "  Build type: ${CMAKE_BUILD_TYPE}")

# 使用內建的 RTK Framework 目標 (同一構建過程中)
# 包含 framework 的頭文件目錄
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/../framework/include)

# 建立插件示範應用程式
message(STATUS "Building plugin demo application")

# 建立示範應用
add_executable(plugin_demo plugin_demo.c)

# 連結 RTK MQTT Framework (直接鏈接構建目標)
target_link_libraries(plugin_demo 
    rtk_mqtt_framework  # 直接使用庫目標
    ${CMAKE_DL_LIBS}    # 動態載入庫
)

# 設置輸出目錄
set_target_properties(plugin_demo PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}
)

message(STATUS "Demo application: ${CMAKE_CURRENT_BINARY_DIR}/plugin_demo")

# 建立插件範例 (使用相同的方法)
add_subdirectory(wifi_router)
add_subdirectory(iot_sensor)
add_subdirectory(smart_switch)

message(STATUS "Examples configured successfully")