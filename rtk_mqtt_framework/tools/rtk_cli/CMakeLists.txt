cmake_minimum_required(VERSION 3.10)

# RTK CLI 命令列工具
project(rtk_cli VERSION 1.0.0 LANGUAGES C)

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

# 建立可執行檔
add_executable(rtk_cli
    rtk_cli.c
)

# 連結 RTK MQTT Framework (直接鏈接構建目標)
target_link_libraries(rtk_cli 
    rtk_mqtt_framework  # 直接使用庫目標
    ${CMAKE_DL_LIBS}    # 動態載入庫
    m   # 數學庫
)

# 設置輸出屬性
set_target_properties(rtk_cli PROPERTIES
    VERSION ${PROJECT_VERSION}
    RUNTIME_OUTPUT_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}
)

# 安裝設定
install(TARGETS rtk_cli
    RUNTIME DESTINATION bin
)

# 產生編譯資訊
message(STATUS "RTK CLI Configuration:")
message(STATUS "  Version: ${PROJECT_VERSION}")
message(STATUS "  Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "  Output: ${CMAKE_CURRENT_BINARY_DIR}/rtk_cli")
message(STATUS "  Install: bin/rtk_cli")