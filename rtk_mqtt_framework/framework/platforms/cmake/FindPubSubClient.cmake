# FindPubSubClient.cmake
# 
# 尋找 PubSubClient 函式庫
#
# 使用方法:
#   find_package(PubSubClient REQUIRED)
#
# 定義的變數:
#   PubSubClient_FOUND       - 如果找到 PubSubClient 則為 TRUE
#   PubSubClient_INCLUDE_DIRS - PubSubClient 的包含目錄
#   PubSubClient_LIBRARIES    - PubSubClient 的函式庫
#   PubSubClient_VERSION      - PubSubClient 的版本

# 查找標頭檔
find_path(PubSubClient_INCLUDE_DIR
    NAMES PubSubClient.h
    HINTS
        ${PUBSUBCLIENT_ROOT}/src
        ${PUBSUBCLIENT_ROOT}/include
        $ENV{PUBSUBCLIENT_ROOT}/src
        $ENV{PUBSUBCLIENT_ROOT}/include
    PATHS
        /usr/include
        /usr/local/include
        /opt/local/include
        # Arduino 庫路徑
        ~/Arduino/libraries/PubSubClient/src
        ~/Documents/Arduino/libraries/PubSubClient/src
        # PlatformIO 庫路徑
        ~/.platformio/lib/*/PubSubClient/src
        # 常見安裝路徑
        /usr/share/arduino/libraries/PubSubClient/src
        /usr/local/share/arduino/libraries/PubSubClient/src
)

# 查找函式庫 (通常 PubSubClient 是標頭檔庫)
find_library(PubSubClient_LIBRARY
    NAMES PubSubClient pubsubclient
    HINTS
        ${PUBSUBCLIENT_ROOT}/lib
        $ENV{PUBSUBCLIENT_ROOT}/lib
    PATHS
        /usr/lib
        /usr/local/lib
        /opt/local/lib
)

# 嘗試從標頭檔中提取版本資訊
if(PubSubClient_INCLUDE_DIR AND EXISTS "${PubSubClient_INCLUDE_DIR}/PubSubClient.h")
    file(READ "${PubSubClient_INCLUDE_DIR}/PubSubClient.h" _pubsubclient_header)
    
    # 查找版本定義
    string(REGEX MATCH "#define[ \t]+MQTT_VERSION[ \t]+\"([^\"]+)\"" 
           _pubsubclient_version_match "${_pubsubclient_header}")
    
    if(_pubsubclient_version_match)
        set(PubSubClient_VERSION "${CMAKE_MATCH_1}")
    else()
        # 如果沒有版本資訊，使用預設版本
        set(PubSubClient_VERSION "2.8.0")
    endif()
    
    # 檢查是否支援自定義 Client
    string(REGEX MATCH "class PubSubClient" _pubsubclient_class_match "${_pubsubclient_header}")
    if(_pubsubclient_class_match)
        set(PubSubClient_HAS_CLIENT_SUPPORT TRUE)
    else()
        set(PubSubClient_HAS_CLIENT_SUPPORT FALSE)
    endif()
endif()

# 設定包含目錄
set(PubSubClient_INCLUDE_DIRS ${PubSubClient_INCLUDE_DIR})

# 設定函式庫列表
if(PubSubClient_LIBRARY)
    set(PubSubClient_LIBRARIES ${PubSubClient_LIBRARY})
else()
    # PubSubClient 通常是標頭檔庫，不需要連結
    set(PubSubClient_LIBRARIES "")
endif()

# 使用標準模組來處理 REQUIRED 和 QUIET 參數
include(FindPackageHandleStandardArgs)

find_package_handle_standard_args(PubSubClient
    FOUND_VAR PubSubClient_FOUND
    REQUIRED_VARS 
        PubSubClient_INCLUDE_DIR
    VERSION_VAR PubSubClient_VERSION
)

# 建立 IMPORTED 目標 (如果找到)
if(PubSubClient_FOUND AND NOT TARGET PubSubClient::PubSubClient)
    if(PubSubClient_LIBRARY)
        # 如果有函式庫檔案
        add_library(PubSubClient::PubSubClient UNKNOWN IMPORTED)
        set_target_properties(PubSubClient::PubSubClient PROPERTIES
            IMPORTED_LOCATION "${PubSubClient_LIBRARY}"
            INTERFACE_INCLUDE_DIRECTORIES "${PubSubClient_INCLUDE_DIR}"
        )
    else()
        # 如果只有標頭檔
        add_library(PubSubClient::PubSubClient INTERFACE IMPORTED)
        set_target_properties(PubSubClient::PubSubClient PROPERTIES
            INTERFACE_INCLUDE_DIRECTORIES "${PubSubClient_INCLUDE_DIR}"
        )
    endif()
endif()

# 隱藏內部變數
mark_as_advanced(
    PubSubClient_INCLUDE_DIR
    PubSubClient_LIBRARY
)

# 除錯資訊
if(PubSubClient_FOUND)
    message(STATUS "Found PubSubClient: ${PubSubClient_INCLUDE_DIR}")
    message(STATUS "PubSubClient version: ${PubSubClient_VERSION}")
    message(STATUS "PubSubClient library: ${PubSubClient_LIBRARY}")
    message(STATUS "PubSubClient has client support: ${PubSubClient_HAS_CLIENT_SUPPORT}")
endif()