cmake_minimum_required(VERSION 3.10)

# WiFi 路由器插件範例
set(PLUGIN_NAME wifi_router)
project(${PLUGIN_NAME}_plugin VERSION 1.0.0 LANGUAGES C)

# 設置 C 標準
set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)

# 編譯選項
if(CMAKE_COMPILER_IS_GNUCC)
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wall -Wextra -Werror")
endif()

# Debug 和 Release 配置
set(CMAKE_C_FLAGS_DEBUG "-g -O0 -DDEBUG")
set(CMAKE_C_FLAGS_RELEASE "-O2 -DNDEBUG")

# 使用內建的 RTK Framework (同一構建過程中)
# 包含 framework 的頭文件目錄
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/../../framework/include)

message(STATUS "RTK Framework include: ${CMAKE_CURRENT_SOURCE_DIR}/../../framework/include")
message(STATUS "RTK Framework library: rtk_mqtt_framework (build target)")

# 建立共享庫 (插件)
add_library(${PLUGIN_NAME}_plugin SHARED
    wifi_router_plugin.c
)

# 連結 RTK MQTT Framework (直接鏈接構建目標)
target_link_libraries(${PLUGIN_NAME}_plugin 
    rtk_mqtt_framework  # 直接使用庫目標
    m  # 數學庫
)

# 設置輸出屬性
set_target_properties(${PLUGIN_NAME}_plugin PROPERTIES
    VERSION ${PROJECT_VERSION}
    SOVERSION 1
    PREFIX ""  # 移除 lib 前綴
    OUTPUT_NAME ${PLUGIN_NAME}_plugin
    LIBRARY_OUTPUT_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}
)

# 安裝設定
install(TARGETS ${PLUGIN_NAME}_plugin
    LIBRARY DESTINATION lib/rtk_mqtt_framework/plugins
    RUNTIME DESTINATION lib/rtk_mqtt_framework/plugins
)

# 產生編譯資訊
message(STATUS "WiFi Router Plugin Configuration:")
message(STATUS "  Version: ${PROJECT_VERSION}")
message(STATUS "  Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "  Output: ${CMAKE_CURRENT_BINARY_DIR}/${PLUGIN_NAME}_plugin.so")
message(STATUS "  Install: lib/rtk_mqtt_framework/plugins/")