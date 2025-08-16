# External Dependencies CMakeLists.txt
# This file builds all bundled external libraries

cmake_minimum_required(VERSION 3.10)

# === cJSON Library ===
set(cJSON_ROOT ${CMAKE_CURRENT_SOURCE_DIR}/cjson)

if(EXISTS ${cJSON_ROOT}/CMakeLists.txt)
    message(STATUS "Building bundled cJSON from: ${cJSON_ROOT}")
    
    # Configure cJSON options
    set(ENABLE_CJSON_UTILS ON CACHE BOOL "Enable cJSON utilities")
    set(ENABLE_CJSON_TEST OFF CACHE BOOL "Disable cJSON tests")
    set(BUILD_SHARED_AND_STATIC_LIBS ON CACHE BOOL "Build both shared and static libs")
    set(CJSON_OVERRIDE_BUILD_SHARED_LIBS OFF CACHE BOOL "Don't override shared libs")
    
    # Add cJSON subdirectory
    add_subdirectory(${cJSON_ROOT} cjson-build)
    
    # Create IMPORTED targets for easier linking
    add_library(cjson::cjson ALIAS cjson)
    if(TARGET cjson_utils)
        add_library(cjson::cjson_utils ALIAS cjson_utils)
    endif()
    
    # Set variables for parent CMake
    set(CJSON_LIBRARIES cjson PARENT_SCOPE)
    set(CJSON_INCLUDE_DIRS ${cJSON_ROOT} PARENT_SCOPE)
    set(CJSON_FOUND TRUE PARENT_SCOPE)
    
else()
    message(FATAL_ERROR "cJSON source not found at: ${cJSON_ROOT}")
endif()

# === Eclipse Paho MQTT C Library ===
set(PAHO_MQTT_C_ROOT ${CMAKE_CURRENT_SOURCE_DIR}/paho-mqtt-c)

if(EXISTS ${PAHO_MQTT_C_ROOT}/CMakeLists.txt)
    message(STATUS "Building bundled Paho MQTT C from: ${PAHO_MQTT_C_ROOT}")
    
    # Configure Paho MQTT C options
    set(PAHO_BUILD_STATIC ON CACHE BOOL "Build static library")
    set(PAHO_BUILD_SHARED ON CACHE BOOL "Build shared library") 
    set(PAHO_WITH_SSL OFF CACHE BOOL "Disable SSL/TLS support by default")
    set(PAHO_BUILD_DOCUMENTATION OFF CACHE BOOL "Disable documentation")
    set(PAHO_BUILD_SAMPLES OFF CACHE BOOL "Disable samples")
    
    # Platform-specific SSL configuration
    if(RTK_ENABLE_SSL)
        set(PAHO_WITH_SSL ON CACHE BOOL "Enable SSL/TLS support" FORCE)
    endif()
    
    # Add Paho MQTT C subdirectory
    add_subdirectory(${PAHO_MQTT_C_ROOT} paho-mqtt-c-build)
    
    # Create IMPORTED targets for easier linking
    if(TARGET paho-mqtt3c-static)
        add_library(paho-mqtt::paho-mqtt3c-static ALIAS paho-mqtt3c-static)
    endif()
    if(TARGET paho-mqtt3c)
        add_library(paho-mqtt::paho-mqtt3c ALIAS paho-mqtt3c)
    endif()
    if(TARGET paho-mqtt3a-static)
        add_library(paho-mqtt::paho-mqtt3a-static ALIAS paho-mqtt3a-static)
    endif()
    if(TARGET paho-mqtt3a)
        add_library(paho-mqtt::paho-mqtt3a ALIAS paho-mqtt3a)
    endif()
    
    # Set variables for parent CMake
    set(PAHO_MQTT_C_LIBRARIES paho-mqtt3c-static PARENT_SCOPE)
    set(PAHO_MQTT_C_ASYNC_LIBRARIES paho-mqtt3a-static PARENT_SCOPE)
    set(PAHO_MQTT_C_INCLUDE_DIRS ${PAHO_MQTT_C_ROOT}/src PARENT_SCOPE)
    set(PAHO_MQTT_C_FOUND TRUE PARENT_SCOPE)
    
else()
    message(FATAL_ERROR "Paho MQTT C source not found at: ${PAHO_MQTT_C_ROOT}")
endif()

# === PubSubClient (already available) ===
set(PUBSUBCLIENT_ROOT ${CMAKE_CURRENT_SOURCE_DIR}/pubsubclient)

if(EXISTS ${PUBSUBCLIENT_ROOT}/src/PubSubClient.h)
    message(STATUS "Found bundled PubSubClient at: ${PUBSUBCLIENT_ROOT}")
    
    # Create PubSubClient library target
    add_library(pubsubclient STATIC
        ${PUBSUBCLIENT_ROOT}/src/PubSubClient.cpp
    )
    
    target_include_directories(pubsubclient PUBLIC
        ${PUBSUBCLIENT_ROOT}/src
    )
    
    # Set variables for parent CMake
    set(PUBSUBCLIENT_LIBRARIES pubsubclient PARENT_SCOPE)
    set(PUBSUBCLIENT_INCLUDE_DIRS ${PUBSUBCLIENT_ROOT}/src PARENT_SCOPE)
    set(PUBSUBCLIENT_FOUND TRUE PARENT_SCOPE)
    
else()
    message(WARNING "PubSubClient source not found at: ${PUBSUBCLIENT_ROOT}")
endif()

# === Summary ===
message(STATUS "=== External Dependencies Summary ===")
message(STATUS "cJSON: ${CJSON_FOUND}")
message(STATUS "Paho MQTT C: ${PAHO_MQTT_C_FOUND}")
message(STATUS "PubSubClient: ${PUBSUBCLIENT_FOUND}")
message(STATUS "=====================================")

# Export all targets for installation
export(TARGETS cjson 
       FILE "${CMAKE_CURRENT_BINARY_DIR}/external-targets.cmake")

if(TARGET paho-mqtt3c-static)
    export(TARGETS paho-mqtt3c-static 
           APPEND FILE "${CMAKE_CURRENT_BINARY_DIR}/external-targets.cmake")
endif()

if(TARGET pubsubclient)
    export(TARGETS pubsubclient 
           APPEND FILE "${CMAKE_CURRENT_BINARY_DIR}/external-targets.cmake")
endif()