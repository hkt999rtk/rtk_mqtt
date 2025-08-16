cmake_minimum_required(VERSION 3.10)
project(rtk_freertos_device_example VERSION 1.0.0 LANGUAGES C)

# === 編譯器和平台設定 ===

# 設定為 FreeRTOS 交叉編譯
set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)

# 強制啟用 FreeRTOS 平台
set(RTK_TARGET_FREERTOS ON CACHE BOOL "Target FreeRTOS platform" FORCE)
set(RTK_USE_PUBSUBCLIENT ON CACHE BOOL "Use PubSubClient for FreeRTOS" FORCE)
set(RTK_USE_LIGHTWEIGHT_JSON ON CACHE BOOL "Use lightweight JSON" FORCE)

# ARM Cortex-M 編譯器設定 (如果使用交叉編譯)
if(DEFINED ARM_TOOLCHAIN_PREFIX)
    set(CMAKE_C_COMPILER ${ARM_TOOLCHAIN_PREFIX}gcc)
    set(CMAKE_CXX_COMPILER ${ARM_TOOLCHAIN_PREFIX}g++)
    set(CMAKE_ASM_COMPILER ${CMAKE_C_COMPILER})
    set(CMAKE_OBJCOPY ${ARM_TOOLCHAIN_PREFIX}objcopy)
    set(CMAKE_OBJDUMP ${ARM_TOOLCHAIN_PREFIX}objdump)
    set(CMAKE_SIZE ${ARM_TOOLCHAIN_PREFIX}size)
    
    # 針對 ARM Cortex-M4 優化
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -mcpu=cortex-m4 -mthumb -mfloat-abi=hard -mfpu=fpv4-sp-d16")
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -ffunction-sections -fdata-sections -fno-common")
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wall -Wextra -Werror")
    
    # 連結器設定
    set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} -mcpu=cortex-m4 -mthumb")
    set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} -mfloat-abi=hard -mfpu=fpv4-sp-d16")
    set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} -Wl,--gc-sections -Wl,--print-memory-usage")
endif()

# === 依賴和路徑設定 ===

# 設定 FreeRTOS 路徑
if(NOT DEFINED FREERTOS_PATH)
    set(FREERTOS_PATH "${CMAKE_CURRENT_SOURCE_DIR}/../../../third_party/FreeRTOS" 
        CACHE PATH "FreeRTOS source directory")
endif()

# 設定 PubSubClient 路徑 (用於 Arduino 函式庫)
if(NOT DEFINED PUBSUBCLIENT_PATH)
    set(PUBSUBCLIENT_PATH "${CMAKE_CURRENT_SOURCE_DIR}/../../../third_party/PubSubClient"
        CACHE PATH "PubSubClient source directory")
endif()

# 包含父目錄的 CMake 設定 (RTK Framework)
add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/../../ rtk_framework)

# === 原始碼檔案 ===

set(EXAMPLE_SOURCES
    main.c
)

set(EXAMPLE_HEADERS
    FreeRTOSConfig.h
)

# === 建立可執行檔 ===

add_executable(freertos_device_example ${EXAMPLE_SOURCES})

# === 編譯定義 ===

target_compile_definitions(freertos_device_example PRIVATE
    # FreeRTOS 設定
    FREERTOS
    RTK_PLATFORM_FREERTOS
    
    # 網路和 MQTT 設定
    RTK_USE_PUBSUBCLIENT
    RTK_USE_LIGHTWEIGHT_JSON
    
    # 除錯設定
    $<$<CONFIG:Debug>:DEBUG>
    $<$<CONFIG:Debug>:RTK_DEBUG>
)

# === 包含目錄 ===

target_include_directories(freertos_device_example PRIVATE
    # 當前目錄 (包含 FreeRTOSConfig.h)
    ${CMAKE_CURRENT_SOURCE_DIR}
    
    # RTK Framework 標頭檔
    ${CMAKE_CURRENT_SOURCE_DIR}/../../include
    
    # FreeRTOS 標頭檔
    ${FREERTOS_PATH}/include
    ${FREERTOS_PATH}/portable/GCC/ARM_CM4F  # 針對 Cortex-M4F
    
    # PubSubClient 標頭檔
    ${PUBSUBCLIENT_PATH}/src
)

# === 連結庫 ===

target_link_libraries(freertos_device_example
    rtk_mqtt_framework  # 主要框架庫
)

# === FreeRTOS 源碼 ===

# 查找 FreeRTOS 核心源碼
file(GLOB FREERTOS_CORE_SOURCES
    "${FREERTOS_PATH}/*.c"
    "${FREERTOS_PATH}/portable/GCC/ARM_CM4F/*.c"
    "${FREERTOS_PATH}/portable/MemMang/heap_4.c"  # 使用 heap_4 記憶體管理
)

# 新增 FreeRTOS 源碼到專案
target_sources(freertos_device_example PRIVATE ${FREERTOS_CORE_SOURCES})

# === PubSubClient 源碼 (如果需要) ===

if(EXISTS "${PUBSUBCLIENT_PATH}/src/PubSubClient.cpp")
    target_sources(freertos_device_example PRIVATE
        "${PUBSUBCLIENT_PATH}/src/PubSubClient.cpp"
    )
    # 啟用 C++ 支援
    enable_language(CXX)
    set_property(TARGET freertos_device_example PROPERTY LINKER_LANGUAGE CXX)
endif()

# === 編譯後處理 ===

if(DEFINED ARM_TOOLCHAIN_PREFIX)
    # 產生 .hex 和 .bin 檔案
    add_custom_command(TARGET freertos_device_example POST_BUILD
        COMMAND ${CMAKE_OBJCOPY} -O ihex $<TARGET_FILE:freertos_device_example> ${CMAKE_CURRENT_BINARY_DIR}/freertos_device_example.hex
        COMMAND ${CMAKE_OBJCOPY} -O binary $<TARGET_FILE:freertos_device_example> ${CMAKE_CURRENT_BINARY_DIR}/freertos_device_example.bin
        COMMENT "Generating HEX and BIN files"
    )
    
    # 顯示記憶體使用量
    add_custom_command(TARGET freertos_device_example POST_BUILD
        COMMAND ${CMAKE_SIZE} $<TARGET_FILE:freertos_device_example>
        COMMENT "Memory usage:"
    )
    
    # 產生反組譯檔案 (除錯用)
    add_custom_command(TARGET freertos_device_example POST_BUILD
        COMMAND ${CMAKE_OBJDUMP} -h -S $<TARGET_FILE:freertos_device_example> > ${CMAKE_CURRENT_BINARY_DIR}/freertos_device_example.list
        COMMENT "Generating disassembly listing"
    )
endif()

# === 安裝設定 ===

install(TARGETS freertos_device_example
    DESTINATION bin
)

if(DEFINED ARM_TOOLCHAIN_PREFIX)
    install(FILES 
        ${CMAKE_CURRENT_BINARY_DIR}/freertos_device_example.hex
        ${CMAKE_CURRENT_BINARY_DIR}/freertos_device_example.bin
        DESTINATION bin
    )
endif()

# === 建置資訊 ===

message(STATUS "")
message(STATUS "=== FreeRTOS Device Example Configuration ===")
message(STATUS "  Project: ${PROJECT_NAME} v${PROJECT_VERSION}")
message(STATUS "  FreeRTOS Path: ${FREERTOS_PATH}")
message(STATUS "  PubSubClient Path: ${PUBSUBCLIENT_PATH}")
message(STATUS "  Compiler: ${CMAKE_C_COMPILER}")
if(DEFINED ARM_TOOLCHAIN_PREFIX)
    message(STATUS "  Target: ARM Cortex-M4F")
    message(STATUS "  Cross-compile: YES")
else()
    message(STATUS "  Target: Host platform (simulation)")
    message(STATUS "  Cross-compile: NO")
endif()
message(STATUS "  Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "============================================")

# === 建置選項說明 ===

message(STATUS "")
message(STATUS "Build Instructions:")
message(STATUS "  For simulation (host): cmake -B build && make -C build")
message(STATUS "  For ARM target: cmake -B build -DARM_TOOLCHAIN_PREFIX=arm-none-eabi- && make -C build")
message(STATUS "  With custom FreeRTOS: cmake -B build -DFREERTOS_PATH=/path/to/freertos")
message(STATUS "")

# === 額外目標 ===

# 清理目標
add_custom_target(clean-all
    COMMAND ${CMAKE_COMMAND} -E remove_directory ${CMAKE_CURRENT_BINARY_DIR}
    COMMENT "Cleaning all build files"
)

# 快速建置目標 (僅編譯，不連結)
add_custom_target(compile-check
    COMMAND ${CMAKE_COMMAND} --build ${CMAKE_CURRENT_BINARY_DIR} --target freertos_device_example --config Debug -- -j 4
    COMMENT "Quick compile check"
)