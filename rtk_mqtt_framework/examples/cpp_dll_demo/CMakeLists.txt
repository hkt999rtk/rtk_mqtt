cmake_minimum_required(VERSION 3.10)
project(RTK_MQTT_Framework_CPP_DLL_Demo)

# Set C++ standard
set(CMAKE_CXX_STANDARD 11)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Set C standard for C plugin
set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)

# Platform-specific configurations
if(WIN32)
    # Windows specific settings
    add_definitions(-D_WIN32)
    if(MSVC)
        # Visual Studio specific flags
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} /W3")
        set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} /W3")
    else()
        # MinGW specific flags
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wall -Wextra")
        set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wall -Wextra")
    endif()
else()
    # Unix-like systems (Linux, macOS)
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wall -Wextra")
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wall -Wextra")
endif()

# Include directories
include_directories(${CMAKE_CURRENT_SOURCE_DIR})

# Find RTK MQTT Framework header (look in parent directories)
set(RTK_HEADER_SEARCH_PATHS
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/../../framework-go/cmd/rtk-dll
    ${CMAKE_CURRENT_SOURCE_DIR}/../../framework-go/cmd/rtk-dll-simple
    ${CMAKE_CURRENT_SOURCE_DIR}/../../../rtk_mqtt_framework/framework-go/cmd/rtk-dll
)

foreach(search_path ${RTK_HEADER_SEARCH_PATHS})
    if(EXISTS "${search_path}/rtk_mqtt_framework.h")
        set(RTK_HEADER_PATH "${search_path}/rtk_mqtt_framework.h")
        include_directories(${search_path})
        message(STATUS "Found RTK MQTT Framework header: ${RTK_HEADER_PATH}")
        break()
    elseif(EXISTS "${search_path}/rtk_mqtt_framework_simple.h")
        set(RTK_HEADER_PATH "${search_path}/rtk_mqtt_framework_simple.h")
        include_directories(${search_path})
        message(STATUS "Found RTK MQTT Framework simple header: ${RTK_HEADER_PATH}")
        break()
    endif()
endforeach()

if(NOT RTK_HEADER_PATH)
    message(WARNING "RTK MQTT Framework header not found. Please copy rtk_mqtt_framework.h to this directory.")
endif()

# C++ Demo Executable
add_executable(cpp_dll_demo
    main.cpp
)

# C Plugin Executable  
add_executable(smart_thermostat_plugin
    smart_thermostat_plugin.c
)

# Platform-specific linking
if(WIN32)
    # Windows: Link with system libraries
    target_link_libraries(cpp_dll_demo)
    target_link_libraries(smart_thermostat_plugin)
else()
    # Unix: Link with dl library for dynamic loading
    target_link_libraries(cpp_dll_demo dl)
    target_link_libraries(smart_thermostat_plugin dl)
endif()

# Threading support
find_package(Threads REQUIRED)
target_link_libraries(cpp_dll_demo Threads::Threads)
target_link_libraries(smart_thermostat_plugin Threads::Threads)

# Install targets
install(TARGETS cpp_dll_demo smart_thermostat_plugin
    RUNTIME DESTINATION bin
)

# Copy header file if found
if(RTK_HEADER_PATH)
    configure_file(${RTK_HEADER_PATH} ${CMAKE_CURRENT_BINARY_DIR}/rtk_mqtt_framework.h COPYONLY)
    install(FILES ${RTK_HEADER_PATH} DESTINATION include)
endif()

# Print build information
message(STATUS "RTK MQTT Framework C++ DLL Demo Configuration:")
message(STATUS "  C++ Standard: ${CMAKE_CXX_STANDARD}")
message(STATUS "  C Standard: ${CMAKE_C_STANDARD}")
message(STATUS "  Build Type: ${CMAKE_BUILD_TYPE}")
message(STATUS "  Platform: ${CMAKE_SYSTEM_NAME}")
if(WIN32)
    message(STATUS "  Target: Windows DLL integration")
else()
    message(STATUS "  Target: Unix shared library integration")
endif()

# Custom target to copy DLL/SO files
add_custom_target(copy_dll_files ALL
    COMMENT "Copying RTK MQTT Framework DLL/SO files..."
)

# Look for DLL/SO files to copy
set(DLL_SEARCH_PATHS
    ${CMAKE_CURRENT_SOURCE_DIR}/../../framework-go/cmd/rtk-dll
    ${CMAKE_CURRENT_SOURCE_DIR}/../../framework-go/cmd/rtk-dll-simple
    ${CMAKE_CURRENT_SOURCE_DIR}/../../framework-go/dist
)

foreach(search_path ${DLL_SEARCH_PATHS})
    file(GLOB RTK_DLL_FILES 
        "${search_path}/*.dll"
        "${search_path}/*.so"
        "${search_path}/*.dylib"
    )
    if(RTK_DLL_FILES)
        foreach(dll_file ${RTK_DLL_FILES})
            add_custom_command(TARGET copy_dll_files POST_BUILD
                COMMAND ${CMAKE_COMMAND} -E copy_if_different
                ${dll_file} ${CMAKE_CURRENT_BINARY_DIR}/
            )
            message(STATUS "Will copy DLL/SO: ${dll_file}")
        endforeach()
        break()
    endif()
endforeach()

# Documentation
message(STATUS "")
message(STATUS "Build Instructions:")
message(STATUS "  mkdir build && cd build")
message(STATUS "  cmake ..")
message(STATUS "  make")
message(STATUS "")
message(STATUS "Run Instructions:")
if(WIN32)
    message(STATUS "  .\\cpp_dll_demo.exe")
    message(STATUS "  .\\smart_thermostat_plugin.exe [device_id] [broker_host] [broker_port]")
else()
    message(STATUS "  ./cpp_dll_demo")
    message(STATUS "  ./smart_thermostat_plugin [device_id] [broker_host] [broker_port]")
endif()
message(STATUS "")