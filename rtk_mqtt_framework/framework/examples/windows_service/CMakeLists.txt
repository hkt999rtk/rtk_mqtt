cmake_minimum_required(VERSION 3.10)
project(rtk_windows_service_example VERSION 1.0.0 LANGUAGES C)

# === 平台檢查 ===

if(NOT WIN32)
    message(FATAL_ERROR "This example requires Windows platform")
endif()

# === 編譯器設定 ===

set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)

# 強制啟用 Windows 平台
set(RTK_TARGET_WINDOWS ON CACHE BOOL "Target Windows platform" FORCE)
set(RTK_ENABLE_WINDOWS_SERVICE ON CACHE BOOL "Enable Windows Service support" FORCE)

# Windows 特定編譯選項
if(MSVC)
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} /W3")
    # 啟用結構化例外處理
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} /EHsc")
    # Unicode 支援
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} /DUNICODE /D_UNICODE")
else()
    # MinGW 編譯器
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wall -Wextra")
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -DUNICODE -D_UNICODE")
endif()

# === 依賴設定 ===

# 包含父目錄的 CMake 設定 (RTK Framework)
add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/../../ rtk_framework)

# === 原始碼檔案 ===

set(EXAMPLE_SOURCES
    main.c
)

# === 建立可執行檔 ===

add_executable(windows_service_example ${EXAMPLE_SOURCES})

# === 編譯定義 ===

target_compile_definitions(windows_service_example PRIVATE
    # Windows 平台
    RTK_PLATFORM_WINDOWS
    RTK_WINDOWS_SERVICE
    _WIN32_WINNT=0x0601  # Windows 7+
    WINVER=0x0601
    
    # Unicode 支援
    UNICODE
    _UNICODE
    
    # 除錯設定
    $<$<CONFIG:Debug>:DEBUG>
    $<$<CONFIG:Debug>:RTK_DEBUG>
    $<$<CONFIG:Debug>:_DEBUG>
)

# === 包含目錄 ===

target_include_directories(windows_service_example PRIVATE
    # RTK Framework 標頭檔
    ${CMAKE_CURRENT_SOURCE_DIR}/../../include
)

# === 連結庫 ===

target_link_libraries(windows_service_example
    rtk_mqtt_framework  # 主要框架庫
    
    # Windows 系統庫
    ws2_32      # Winsock
    advapi32    # Service 控制和註冊表
    user32      # 使用者介面
    kernel32    # 核心功能
    ole32       # COM 支援
    shell32     # Shell 功能
)

# === 建置後處理 ===

# 產生除錯符號檔案
if(MSVC)
    set_target_properties(windows_service_example PROPERTIES
        LINK_FLAGS "/DEBUG"
    )
endif()

# 複製到輸出目錄
add_custom_command(TARGET windows_service_example POST_BUILD
    COMMAND ${CMAKE_COMMAND} -E copy_if_different
        $<TARGET_FILE:windows_service_example>
        ${CMAKE_CURRENT_BINARY_DIR}/RTKMqttCollector.exe
    COMMENT "Creating service executable"
)

# === 安裝設定 ===

install(TARGETS windows_service_example
    DESTINATION bin
    RENAME RTKMqttCollector.exe
)

# 安裝服務配置檔案
configure_file(
    "${CMAKE_CURRENT_SOURCE_DIR}/service_config.xml.in"
    "${CMAKE_CURRENT_BINARY_DIR}/service_config.xml"
    @ONLY
)

install(FILES "${CMAKE_CURRENT_BINARY_DIR}/service_config.xml"
    DESTINATION etc
)

# === 服務管理腳本 ===

# 建立服務安裝腳本
configure_file(
    "${CMAKE_CURRENT_SOURCE_DIR}/install_service.bat.in"
    "${CMAKE_CURRENT_BINARY_DIR}/install_service.bat"
    @ONLY
)

# 建立服務移除腳本
configure_file(
    "${CMAKE_CURRENT_SOURCE_DIR}/uninstall_service.bat.in"
    "${CMAKE_CURRENT_BINARY_DIR}/uninstall_service.bat"
    @ONLY
)

install(FILES 
    "${CMAKE_CURRENT_BINARY_DIR}/install_service.bat"
    "${CMAKE_CURRENT_BINARY_DIR}/uninstall_service.bat"
    DESTINATION bin
)

# === 測試目標 ===

# 建立控制台測試目標
add_custom_target(test-console
    COMMAND $<TARGET_FILE:windows_service_example> console
    DEPENDS windows_service_example
    COMMENT "Running service in console mode for testing"
    WORKING_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}
)

# 建立服務安裝測試目標 (需要管理員權限)
add_custom_target(install-service
    COMMAND ${CMAKE_CURRENT_BINARY_DIR}/install_service.bat
    DEPENDS windows_service_example
    COMMENT "Installing Windows Service (requires administrator privileges)"
    WORKING_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}
)

# 建立服務移除測試目標 (需要管理員權限)
add_custom_target(uninstall-service
    COMMAND ${CMAKE_CURRENT_BINARY_DIR}/uninstall_service.bat
    COMMENT "Uninstalling Windows Service (requires administrator privileges)"
    WORKING_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}
)

# === 建置資訊 ===

message(STATUS "")
message(STATUS "=== Windows Service Example Configuration ===")
message(STATUS "  Project: ${PROJECT_NAME} v${PROJECT_VERSION}")
message(STATUS "  Service Name: RTKMqttCollector")
message(STATUS "  Platform: Windows")
message(STATUS "  Compiler: ${CMAKE_C_COMPILER_ID}")
message(STATUS "  Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "=============================================")

# === 使用說明 ===

message(STATUS "")
message(STATUS "Build Instructions:")
message(STATUS "  1. Build: cmake -B build && cmake --build build")
message(STATUS "  2. Test console mode: cmake --build build --target test-console")
message(STATUS "  3. Install service: cmake --build build --target install-service (as Administrator)")
message(STATUS "  4. Uninstall service: cmake --build build --target uninstall-service (as Administrator)")
message(STATUS "")
message(STATUS "Manual Service Management:")
message(STATUS "  Install: sc create RTKMqttCollector binPath= \"[full_path]\\RTKMqttCollector.exe\"")
message(STATUS "  Start: net start RTKMqttCollector")
message(STATUS "  Stop: net stop RTKMqttCollector")
message(STATUS "  Uninstall: sc delete RTKMqttCollector")
message(STATUS "")