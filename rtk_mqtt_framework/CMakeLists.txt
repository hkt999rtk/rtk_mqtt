cmake_minimum_required(VERSION 3.10)

# RTK MQTT Framework 主專案 - Zero External Dependencies Edition
project(rtk_mqtt_framework_project VERSION 1.0.0 LANGUAGES C)

# 設置 C 標準
set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)

# 編譯選項
if(CMAKE_COMPILER_IS_GNUCC)
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wall -Wextra -Werror")
endif()

# Debug 和 Release 配置
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE "Release")
endif()

set(CMAKE_C_FLAGS_DEBUG "-g -O0 -DDEBUG -DRTK_DEBUG")
set(CMAKE_C_FLAGS_RELEASE "-O2 -DNDEBUG")

# 包含 GNUInstallDirs 用於標準安裝路徑
include(GNUInstallDirs)

# 專案資訊
message(STATUS "RTK MQTT Framework Project Configuration:")
message(STATUS "  Version: ${PROJECT_VERSION}")
message(STATUS "  Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "  C Compiler: ${CMAKE_C_COMPILER}")
message(STATUS "  C Flags: ${CMAKE_C_FLAGS}")
message(STATUS "  Install prefix: ${CMAKE_INSTALL_PREFIX}")
message(STATUS "  *** ZERO EXTERNAL DEPENDENCIES - All libraries bundled locally ***")

# 選項配置
option(BUILD_SHARED_LIBS "Build shared libraries" OFF)
option(BUILD_EXAMPLES "Build example programs and plugins" ON)
option(BUILD_TOOLS "Build development tools" ON)
option(BUILD_TESTS "Build test programs" OFF)
option(INSTALL_EXAMPLES "Install example plugins" ON)

message(STATUS "Build Options:")
message(STATUS "  Build shared libs: ${BUILD_SHARED_LIBS}")
message(STATUS "  Build examples: ${BUILD_EXAMPLES}")
message(STATUS "  Build tools: ${BUILD_TOOLS}")
message(STATUS "  Build tests: ${BUILD_TESTS}")
message(STATUS "  Install examples: ${INSTALL_EXAMPLES}")

# 建立核心 framework
message(STATUS "Configuring RTK MQTT Framework Core...")
add_subdirectory(framework)

# 建立範例 (如果啟用)
if(BUILD_EXAMPLES)
    message(STATUS "Configuring RTK MQTT Framework Examples...")
    add_subdirectory(examples)
endif()

# 建立開發工具 (如果啟用)
if(BUILD_TOOLS)
    message(STATUS "Configuring RTK MQTT Framework Tools...")
    add_subdirectory(tools)
endif()

# 建立測試 (如果啟用)
if(BUILD_TESTS)
    message(STATUS "Configuring RTK MQTT Framework Tests...")
    enable_testing()
    add_subdirectory(tests)
endif()

# 安裝 README 和文檔
if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/README.md")
    install(FILES README.md
        DESTINATION ${CMAKE_INSTALL_DOCDIR}
    )
endif()

if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/UNIVERSAL_PLAN.md")
    install(FILES UNIVERSAL_PLAN.md
        DESTINATION ${CMAKE_INSTALL_DOCDIR}
    )
endif()

# 產生並安裝 CMake 配置檔案 (用於其他專案尋找此 framework)
include(CMakePackageConfigHelpers)

# 產生版本檔案
write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/rtk-mqtt-framework-config-version.cmake"
    VERSION ${PROJECT_VERSION}
    COMPATIBILITY SameMajorVersion
)

# 產生配置檔案
configure_package_config_file(
    "${CMAKE_CURRENT_SOURCE_DIR}/cmake/rtk-mqtt-framework-config.cmake.in"
    "${CMAKE_CURRENT_BINARY_DIR}/rtk-mqtt-framework-config.cmake"
    INSTALL_DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/rtk-mqtt-framework
)

# 安裝 CMake 檔案
install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/rtk-mqtt-framework-config.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/rtk-mqtt-framework-config-version.cmake"
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/rtk-mqtt-framework
)

# 建立安裝套件 (CPack)
set(CPACK_PACKAGE_NAME "rtk-mqtt-framework")
set(CPACK_PACKAGE_VERSION ${PROJECT_VERSION})
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "High-performance MQTT diagnostic communication framework for RTK devices")
set(CPACK_PACKAGE_VENDOR "RTK Technologies")
set(CPACK_PACKAGE_CONTACT "support@example.com")

set(CPACK_GENERATOR "TGZ;DEB")
set(CPACK_DEBIAN_PACKAGE_DEPENDS "libc6")
set(CPACK_PACKAGE_DESCRIPTION "Self-contained MQTT framework with zero external dependencies. All required libraries (Eclipse Paho MQTT C, PubSubClient, cJSON) are bundled locally.")
set(CPACK_DEBIAN_PACKAGE_SECTION "devel")
set(CPACK_DEBIAN_PACKAGE_PRIORITY "optional")

include(CPack)

# 最終資訊
message(STATUS "=================================")
message(STATUS "RTK MQTT Framework Configuration Complete")
message(STATUS "=================================")
message(STATUS "To build (No external dependencies required!):")
message(STATUS "  mkdir build && cd build")
message(STATUS "  cmake ..")
message(STATUS "  make -j$(nproc)")
message(STATUS "")
message(STATUS "To install:")
message(STATUS "  sudo make install")
message(STATUS "")
message(STATUS "Note: All MQTT libraries (Paho, PubSubClient) and JSON")
message(STATUS "processing (cJSON) are bundled - no apt/brew installs needed!")
message(STATUS "")
message(STATUS "To run examples:")
message(STATUS "  cd examples")
message(STATUS "  ./plugin_demo -p ./wifi_router/wifi_router_plugin.so")
message(STATUS "=================================")

# 除錯資訊 (Debug 模式下顯示)
if(CMAKE_BUILD_TYPE STREQUAL "Debug")
    message(STATUS "Debug Information:")
    message(STATUS "  CMake version: ${CMAKE_VERSION}")
    message(STATUS "  System name: ${CMAKE_SYSTEM_NAME}")
    message(STATUS "  System processor: ${CMAKE_SYSTEM_PROCESSOR}")
    message(STATUS "  Binary directory: ${CMAKE_BINARY_DIR}")
    message(STATUS "  Source directory: ${CMAKE_SOURCE_DIR}")
endif()