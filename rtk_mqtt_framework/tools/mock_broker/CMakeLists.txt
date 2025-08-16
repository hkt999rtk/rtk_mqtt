cmake_minimum_required(VERSION 3.10)

# Mock MQTT Broker 測試工具
project(mock_mqtt_broker VERSION 1.0.0 LANGUAGES C)

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

# 建立可執行檔
add_executable(mock_mqtt_broker
    mock_mqtt_broker.c
)

# 連結 pthread 庫
target_link_libraries(mock_mqtt_broker 
    pthread
)

# 設置輸出屬性
set_target_properties(mock_mqtt_broker PROPERTIES
    VERSION ${PROJECT_VERSION}
    RUNTIME_OUTPUT_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}
)

# 安裝設定
install(TARGETS mock_mqtt_broker
    RUNTIME DESTINATION bin
)

# 產生編譯資訊
message(STATUS "Mock MQTT Broker Configuration:")
message(STATUS "  Version: ${PROJECT_VERSION}")
message(STATUS "  Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "  Output: ${CMAKE_CURRENT_BINARY_DIR}/mock_mqtt_broker")
message(STATUS "  Features: RTK MQTT message logging")
message(STATUS "  Install: bin/mock_mqtt_broker")