cmake_minimum_required(VERSION 3.10)

# RTK MQTT Framework 開發工具
project(rtk_mqtt_framework_tools VERSION 1.0.0 LANGUAGES C)

# 設置 C 標準
set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)

message(STATUS "Building RTK MQTT Framework Tools")
message(STATUS "  Tools version: ${PROJECT_VERSION}")
message(STATUS "  Build type: ${CMAKE_BUILD_TYPE}")

# 建立 RTK CLI 命令列工具
add_subdirectory(rtk_cli)

# 建立 Mock MQTT Broker 測試工具
add_subdirectory(mock_broker)

# 未來可添加其他工具
# add_subdirectory(test_framework)
# add_subdirectory(config_generator)

message(STATUS "Tools configured successfully")