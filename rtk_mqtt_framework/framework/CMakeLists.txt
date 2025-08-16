cmake_minimum_required(VERSION 3.10)
project(rtk_mqtt_framework VERSION 1.0.0 LANGUAGES C)

# 設置 C 標準
set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)

# === 平台和後端選項 ===
# RTK Framework now uses PubSubClient exclusively
set(RTK_USE_PUBSUBCLIENT ON CACHE BOOL "Use PubSubClient as MQTT backend" FORCE)
option(RTK_TARGET_FREERTOS "Target FreeRTOS platform" OFF)
option(RTK_TARGET_WINDOWS "Target Windows platform" OFF)
option(RTK_ENABLE_FREERTOS_HEAP "Use FreeRTOS heap management" OFF)
option(RTK_ENABLE_WINDOWS_SERVICE "Enable Windows Service support" OFF)
option(RTK_USE_LIGHTWEIGHT_JSON "Use lightweight JSON processing" OFF)

# FreeRTOS 網路堆疊選項
option(RTK_USE_FREERTOS_TCP "Use FreeRTOS+TCP network stack" OFF)
option(RTK_USE_LWIP "Use lwIP network stack" ON)

# MQTT 客戶端後端選項
option(RTK_ENABLE_PAHO_ADAPTER "Enable Paho MQTT C adapter (optional)" OFF)

# 平台檢測
if(WIN32 OR RTK_TARGET_WINDOWS)
    set(RTK_PLATFORM_WINDOWS TRUE)
    set(RTK_TARGET_WINDOWS ON CACHE BOOL "Target Windows platform" FORCE)
elseif(RTK_TARGET_FREERTOS)
    set(RTK_PLATFORM_FREERTOS TRUE)
else()
    set(RTK_PLATFORM_POSIX TRUE)
endif()

# 根據平台設定語言需求
if(RTK_USE_PUBSUBCLIENT)
    enable_language(CXX)
    set(CMAKE_CXX_STANDARD 11)
    set(CMAKE_CXX_STANDARD_REQUIRED ON)
endif()

# === 編譯選項設定 ===
if(RTK_PLATFORM_WINDOWS)
    # Windows 特定編譯選項
    if(MSVC)
        set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} /W3")
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} /W3")
    else()
        # MinGW
        set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wall -Wextra")
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wall -Wextra")
    endif()
elseif(RTK_PLATFORM_FREERTOS)
    # FreeRTOS 特定編譯選項
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wall -Wextra -ffunction-sections -fdata-sections")
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wall -Wextra -ffunction-sections -fdata-sections")
else()
    # POSIX 編譯選項
    if(CMAKE_COMPILER_IS_GNUCC)
        set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wall -Wextra -Werror")
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wall -Wextra -Werror")
    endif()
endif()

# Debug 和 Release 配置
if(RTK_PLATFORM_WINDOWS AND MSVC)
    set(CMAKE_C_FLAGS_DEBUG "/Od /Zi /DDEBUG")
    set(CMAKE_C_FLAGS_RELEASE "/O2 /DNDEBUG")
    set(CMAKE_CXX_FLAGS_DEBUG "/Od /Zi /DDEBUG")
    set(CMAKE_CXX_FLAGS_RELEASE "/O2 /DNDEBUG")
else()
    set(CMAKE_C_FLAGS_DEBUG "-g -O0 -DDEBUG")
    set(CMAKE_C_FLAGS_RELEASE "-O2 -DNDEBUG")
    set(CMAKE_CXX_FLAGS_DEBUG "-g -O0 -DDEBUG")
    set(CMAKE_CXX_FLAGS_RELEASE "-O2 -DNDEBUG")
endif()

# === 依賴庫管理 ===
# All dependencies are now bundled in external/ directory - NO EXTERNAL DEPENDENCIES REQUIRED!

# 載入平台特定的 CMake 模組
list(APPEND CMAKE_MODULE_PATH "${CMAKE_CURRENT_SOURCE_DIR}/platforms/cmake")

# Build bundled external dependencies first
add_subdirectory("${CMAKE_CURRENT_SOURCE_DIR}/../external" external-build)

# 條件依賴管理
set(RTK_LINK_LIBRARIES "")
set(RTK_INCLUDE_DIRS "")

# Use bundled cJSON (always available, no external dependency needed)
if(CJSON_FOUND)
    list(APPEND RTK_LINK_LIBRARIES ${CJSON_LIBRARIES})
    list(APPEND RTK_INCLUDE_DIRS ${CJSON_INCLUDE_DIRS})
    message(STATUS "Using bundled cJSON: ${CJSON_INCLUDE_DIRS}")
else()
    if(NOT RTK_USE_LIGHTWEIGHT_JSON)
        message(FATAL_ERROR "Bundled cJSON not found - this should not happen")
    endif()
endif()

# Use bundled PubSubClient (always available, no external dependency needed)
if(PUBSUBCLIENT_FOUND)
    list(APPEND RTK_LINK_LIBRARIES ${PUBSUBCLIENT_LIBRARIES})
    list(APPEND RTK_INCLUDE_DIRS ${PUBSUBCLIENT_INCLUDE_DIRS})
    message(STATUS "Using bundled PubSubClient: ${PUBSUBCLIENT_INCLUDE_DIRS}")
else()
    message(FATAL_ERROR "Bundled PubSubClient not found - this should not happen")
endif()

# Use bundled Paho MQTT C (optional)
if(PAHO_MQTT_C_FOUND AND RTK_ENABLE_PAHO_ADAPTER)
    list(APPEND RTK_LINK_LIBRARIES ${PAHO_MQTT_C_LIBRARIES})
    list(APPEND RTK_INCLUDE_DIRS ${PAHO_MQTT_C_INCLUDE_DIRS})
    message(STATUS "Using bundled Paho MQTT C: ${PAHO_MQTT_C_INCLUDE_DIRS}")
endif()

# 平台特定依賴
if(RTK_PLATFORM_WINDOWS)
    list(APPEND RTK_LINK_LIBRARIES ws2_32 advapi32)
    if(NOT MSVC)
        list(APPEND RTK_LINK_LIBRARIES pthread)
    endif()
elseif(RTK_PLATFORM_FREERTOS)
    # FreeRTOS 路徑設定
    if(DEFINED FREERTOS_PATH)
        find_path(FREERTOS_INCLUDE_DIR 
            NAMES FreeRTOS.h 
            PATHS ${FREERTOS_PATH}/include
        )
        if(FREERTOS_INCLUDE_DIR)
            list(APPEND RTK_INCLUDE_DIRS ${FREERTOS_INCLUDE_DIR})
            message(STATUS "Found FreeRTOS: ${FREERTOS_INCLUDE_DIR}")
        else()
            message(FATAL_ERROR "FreeRTOS.h not found in ${FREERTOS_PATH}/include")
        endif()
    else()
        message(WARNING "FREERTOS_PATH not defined, assuming FreeRTOS headers are in system path")
    endif()
else()
    # POSIX 依賴
    list(APPEND RTK_LINK_LIBRARIES dl m pthread)
endif()

# Bundled cJSON is already included above - no need for duplicate linking

# === 包含目錄設定 ===
include_directories(include)
foreach(include_dir ${RTK_INCLUDE_DIRS})
    include_directories(${include_dir})
endforeach()

# === 源碼檔案設定 ===

# 核心源碼檔案
set(RTK_FRAMEWORK_SOURCES
    src/mqtt_client.c
    src/topic_builder.c
    src/message_codec.c
    src/schema_validator.c
    src/plugin_manager.c
)

# 核心標頭檔案
set(RTK_FRAMEWORK_HEADERS
    include/rtk_mqtt_client.h
    include/rtk_network_interface.h
    include/rtk_platform_compat.h
    include/rtk_topic_builder.h
    include/rtk_message_codec.h
    include/rtk_schema_validator.h
    include/rtk_device_plugin.h
)

# PubSubClient 源碼檔案 (統一使用)
# Note: Temporarily disabled for minimal build
if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/src/pubsub_adapter.cpp")
    list(APPEND RTK_FRAMEWORK_SOURCES 
        src/pubsub_adapter.cpp
    )
    list(APPEND RTK_FRAMEWORK_HEADERS 
        include/rtk_pubsub_adapter.h
    )
endif()

if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/src/arduino_compat.cpp")
    list(APPEND RTK_FRAMEWORK_SOURCES 
        src/arduino_compat.cpp
    )
    list(APPEND RTK_FRAMEWORK_HEADERS 
        include/arduino_compat.h
    )
endif()

# Optional Paho MQTT C adapter (if enabled)
if(PAHO_MQTT_C_FOUND AND RTK_ENABLE_PAHO_ADAPTER)
    list(APPEND RTK_FRAMEWORK_SOURCES src/paho_adapter.c)
    list(APPEND RTK_FRAMEWORK_HEADERS include/rtk_paho_adapter.h)
    message(STATUS "Paho MQTT C adapter enabled with bundled library")
endif()

# 平台特定源碼檔案
if(RTK_PLATFORM_FREERTOS)
    list(APPEND RTK_FRAMEWORK_SOURCES 
        src/freertos_compat.c
        src/network_freertos.c
    )
elseif(RTK_PLATFORM_WINDOWS)
    list(APPEND RTK_FRAMEWORK_SOURCES 
        src/windows_compat.c
        src/network_windows.c
    )
    if(RTK_ENABLE_WINDOWS_SERVICE)
        list(APPEND RTK_FRAMEWORK_SOURCES src/windows_service.c)
    endif()
else()
    # POSIX
    list(APPEND RTK_FRAMEWORK_SOURCES 
        src/posix_compat.c
    )
    # Add network_posix.c if it exists
    if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/src/network_posix.c")
        list(APPEND RTK_FRAMEWORK_SOURCES src/network_posix.c)
    endif()
endif()
# 注意: 暫時禁用有問題的網路層實現

# JSON 處理源碼
if(RTK_USE_LIGHTWEIGHT_JSON OR RTK_PLATFORM_FREERTOS)
    list(APPEND RTK_FRAMEWORK_SOURCES src/json_pool.c)
    list(APPEND RTK_FRAMEWORK_HEADERS include/rtk_json_config.h)
endif()

# === 建立靜態庫 ===
add_library(rtk_mqtt_framework STATIC ${RTK_FRAMEWORK_SOURCES})

# 設定編譯定義
target_compile_definitions(rtk_mqtt_framework PRIVATE
    RTK_MQTT_FRAMEWORK_VERSION="${PROJECT_VERSION}"
)

# 平台特定編譯定義
if(RTK_PLATFORM_FREERTOS)
    target_compile_definitions(rtk_mqtt_framework PRIVATE FREERTOS RTK_PLATFORM_FREERTOS)
    
    # FreeRTOS 網路堆疊選項
    if(RTK_USE_FREERTOS_TCP)
        target_compile_definitions(rtk_mqtt_framework PRIVATE RTK_USE_FREERTOS_TCP)
    endif()
    if(RTK_USE_LWIP)
        target_compile_definitions(rtk_mqtt_framework PRIVATE RTK_USE_LWIP)
    endif()
elseif(RTK_PLATFORM_WINDOWS)
    target_compile_definitions(rtk_mqtt_framework PRIVATE 
        _WIN32_WINNT=0x0601 RTK_PLATFORM_WINDOWS)
    if(RTK_ENABLE_WINDOWS_SERVICE)
        target_compile_definitions(rtk_mqtt_framework PRIVATE RTK_WINDOWS_SERVICE)
    endif()
else()
    target_compile_definitions(rtk_mqtt_framework PRIVATE RTK_PLATFORM_POSIX)
endif()

# PubSubClient 編譯定義 (統一使用)
target_compile_definitions(rtk_mqtt_framework PRIVATE RTK_USE_PUBSUBCLIENT)
# 設定為 C++ 庫以支持 PubSubClient
set_property(TARGET rtk_mqtt_framework PROPERTY LINKER_LANGUAGE CXX)

if(RTK_USE_LIGHTWEIGHT_JSON)
    target_compile_definitions(rtk_mqtt_framework PRIVATE RTK_USE_LIGHTWEIGHT_JSON)
endif()

# 連結庫
target_link_libraries(rtk_mqtt_framework ${RTK_LINK_LIBRARIES})

# 設置庫的屬性
set_target_properties(rtk_mqtt_framework PROPERTIES
    VERSION ${PROJECT_VERSION}
    SOVERSION 1
    PUBLIC_HEADER "${RTK_FRAMEWORK_HEADERS}"
)

# === 建立共享庫 (可選) ===
option(BUILD_SHARED_LIBS "Build shared libraries" OFF)
if(BUILD_SHARED_LIBS AND NOT RTK_PLATFORM_FREERTOS)
    add_library(rtk_mqtt_framework_shared SHARED ${RTK_FRAMEWORK_SOURCES})
    
    # 套用相同的編譯定義
    target_compile_definitions(rtk_mqtt_framework_shared PRIVATE
        RTK_MQTT_FRAMEWORK_VERSION="${PROJECT_VERSION}"
    )
    
    if(RTK_PLATFORM_WINDOWS)
        target_compile_definitions(rtk_mqtt_framework_shared PRIVATE 
            _WIN32_WINNT=0x0601 RTK_PLATFORM_WINDOWS)
        if(RTK_ENABLE_WINDOWS_SERVICE)
            target_compile_definitions(rtk_mqtt_framework_shared PRIVATE RTK_WINDOWS_SERVICE)
        endif()
    else()
        target_compile_definitions(rtk_mqtt_framework_shared PRIVATE RTK_PLATFORM_POSIX)
    endif()
    
    # PubSubClient 編譯定義 (共享庫)
    target_compile_definitions(rtk_mqtt_framework_shared PRIVATE RTK_USE_PUBSUBCLIENT)
    set_property(TARGET rtk_mqtt_framework_shared PROPERTY LINKER_LANGUAGE CXX)
    
    if(RTK_USE_LIGHTWEIGHT_JSON)
        target_compile_definitions(rtk_mqtt_framework_shared PRIVATE RTK_USE_LIGHTWEIGHT_JSON)
    endif()
    
    target_link_libraries(rtk_mqtt_framework_shared ${RTK_LINK_LIBRARIES})
    
    set_target_properties(rtk_mqtt_framework_shared PROPERTIES
        OUTPUT_NAME rtk_mqtt_framework
        VERSION ${PROJECT_VERSION}
        SOVERSION 1
    )
endif()

# 安裝設定
include(GNUInstallDirs)

# 安裝標頭檔
install(FILES ${RTK_FRAMEWORK_HEADERS}
    DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}/rtk_mqtt_framework
)

# 安裝庫檔案
install(TARGETS rtk_mqtt_framework
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
    PUBLIC_HEADER DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}/rtk_mqtt_framework
)

if(BUILD_SHARED_LIBS)
    install(TARGETS rtk_mqtt_framework_shared
        LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
        ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
    )
endif()

# 產生 pkg-config 檔案
configure_file(
    "${CMAKE_CURRENT_SOURCE_DIR}/rtk_mqtt_framework.pc.in"
    "${CMAKE_CURRENT_BINARY_DIR}/rtk_mqtt_framework.pc"
    @ONLY
)

install(FILES "${CMAKE_CURRENT_BINARY_DIR}/rtk_mqtt_framework.pc"
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/pkgconfig
)

# 建立 framework 內部範例 (可選)
option(BUILD_FRAMEWORK_EXAMPLES "Build framework internal examples" OFF)
if(BUILD_FRAMEWORK_EXAMPLES)
    add_subdirectory(examples)
endif()

# 建立測試 (可選)
option(BUILD_TESTS "Build test programs" OFF)
if(BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()

# === Debug 編譯定義 ===
if(CMAKE_BUILD_TYPE STREQUAL "Debug")
    target_compile_definitions(rtk_mqtt_framework PRIVATE RTK_DEBUG)
    if(BUILD_SHARED_LIBS AND TARGET rtk_mqtt_framework_shared)
        target_compile_definitions(rtk_mqtt_framework_shared PRIVATE RTK_DEBUG)
    endif()
endif()

# === 配置資訊輸出 ===
message(STATUS "")
message(STATUS "=== RTK MQTT Framework Configuration ===")
message(STATUS "  Version: ${PROJECT_VERSION}")
message(STATUS "  Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "  Install prefix: ${CMAKE_INSTALL_PREFIX}")
message(STATUS "")
message(STATUS "Platform Configuration:")
if(RTK_PLATFORM_FREERTOS)
    message(STATUS "  Target Platform: FreeRTOS")
    message(STATUS "  FreeRTOS Path: ${FREERTOS_PATH}")
    message(STATUS "  Use FreeRTOS Heap: ${RTK_ENABLE_FREERTOS_HEAP}")
elseif(RTK_PLATFORM_WINDOWS)
    message(STATUS "  Target Platform: Windows")
    message(STATUS "  Windows Service: ${RTK_ENABLE_WINDOWS_SERVICE}")
    message(STATUS "  Compiler: ${CMAKE_C_COMPILER_ID}")
else()
    message(STATUS "  Target Platform: POSIX (Linux/macOS/Unix)")
endif()
message(STATUS "")
message(STATUS "MQTT Backend Configuration:")
message(STATUS "  MQTT Backend: PubSubClient (unified)")
if(PUBSUBCLIENT_SOURCES)
    message(STATUS "  PubSubClient: Bundled source (${PUBSUBCLIENT_SOURCE_DIR})")
else()
    message(STATUS "  PubSubClient Library: ${PUBSUBCLIENT_LIBRARY}")
endif()
message(STATUS "  PubSubClient Headers: ${PUBSUBCLIENT_INCLUDE_DIR}")
message(STATUS "  Language: C++")
message(STATUS "")
message(STATUS "JSON Processing:")
if(RTK_USE_LIGHTWEIGHT_JSON)
    message(STATUS "  JSON Library: Lightweight/Custom")
else()
    message(STATUS "  JSON Library: cJSON")
    message(STATUS "  cJSON Path: ${CJSON_LIBRARIES}")
endif()
message(STATUS "")
message(STATUS "Build Options:")
message(STATUS "  Build shared libs: ${BUILD_SHARED_LIBS}")
message(STATUS "  Build examples: ${BUILD_EXAMPLES}")
message(STATUS "  Build tests: ${BUILD_TESTS}")
message(STATUS "")
message(STATUS "Compiler Configuration:")
message(STATUS "  C Compiler: ${CMAKE_C_COMPILER}")
message(STATUS "  C Flags: ${CMAKE_C_FLAGS}")
message(STATUS "  CXX Compiler: ${CMAKE_CXX_COMPILER}")
message(STATUS "  CXX Flags: ${CMAKE_CXX_FLAGS}")
message(STATUS "")
message(STATUS "")
message(STATUS "Dependencies Summary:")
message(STATUS "  All dependencies bundled locally - NO EXTERNAL DEPENDENCIES REQUIRED")
message(STATUS "  cJSON: Bundled (${CJSON_FOUND})")
message(STATUS "  PubSubClient: Bundled (${PUBSUBCLIENT_FOUND})")
if(RTK_ENABLE_PAHO_ADAPTER)
    message(STATUS "  Paho MQTT C: Bundled (${PAHO_MQTT_C_FOUND})")
else()
    message(STATUS "  Paho MQTT C: Disabled")
endif()
message(STATUS "")
message(STATUS "Link Libraries: ${RTK_LINK_LIBRARIES}")
message(STATUS "Include Directories: ${RTK_INCLUDE_DIRS}")
message(STATUS "=========================================")