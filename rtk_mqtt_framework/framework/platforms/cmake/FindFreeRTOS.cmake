# FindFreeRTOS.cmake
# 
# 尋找 FreeRTOS 實時作業系統
#
# 使用方法:
#   set(FREERTOS_PATH "/path/to/freertos")  # 可選
#   find_package(FreeRTOS REQUIRED)
#
# 定義的變數:
#   FreeRTOS_FOUND          - 如果找到 FreeRTOS 則為 TRUE
#   FreeRTOS_INCLUDE_DIRS   - FreeRTOS 的包含目錄
#   FreeRTOS_SOURCES        - FreeRTOS 的源碼檔案
#   FreeRTOS_VERSION        - FreeRTOS 的版本
#   FreeRTOS_PORT_DIR       - FreeRTOS 移植層目錄

# 查找 FreeRTOS 主目錄
find_path(FreeRTOS_ROOT_DIR
    NAMES include/FreeRTOS.h
    HINTS
        ${FREERTOS_PATH}
        $ENV{FREERTOS_PATH}
        ${CMAKE_SOURCE_DIR}/third_party/FreeRTOS
        ${CMAKE_SOURCE_DIR}/external/FreeRTOS
        ${CMAKE_SOURCE_DIR}/lib/FreeRTOS
    PATHS
        /usr/src/FreeRTOS
        /opt/FreeRTOS
        # 常見的 ARM 開發環境路徑
        "C:/FreeRTOS"
        "C:/Program Files/FreeRTOS"
        "C:/Program Files (x86)/FreeRTOS"
)

# 查找包含目錄
find_path(FreeRTOS_INCLUDE_DIR
    NAMES FreeRTOS.h
    HINTS
        ${FreeRTOS_ROOT_DIR}/include
        ${FREERTOS_PATH}/include
        $ENV{FREERTOS_PATH}/include
    PATHS
        /usr/include/freertos
        /usr/local/include/freertos
)

# 查找源碼目錄
find_path(FreeRTOS_SOURCE_DIR
    NAMES tasks.c
    HINTS
        ${FreeRTOS_ROOT_DIR}/source
        ${FreeRTOS_ROOT_DIR}/Source
        ${FREERTOS_PATH}/source
        ${FREERTOS_PATH}/Source
        $ENV{FREERTOS_PATH}/source
        $ENV{FREERTOS_PATH}/Source
)

# 查找移植層目錄
find_path(FreeRTOS_PORT_ROOT_DIR
    NAMES GCC
    HINTS
        ${FreeRTOS_ROOT_DIR}/source/portable
        ${FreeRTOS_ROOT_DIR}/Source/portable
        ${FREERTOS_PATH}/source/portable
        ${FREERTOS_PATH}/Source/portable
        $ENV{FREERTOS_PATH}/source/portable
        $ENV{FREERTOS_PATH}/Source/portable
)

# 根據目標處理器自動選擇移植層
if(FreeRTOS_PORT_ROOT_DIR)
    if(CMAKE_SYSTEM_PROCESSOR MATCHES "arm")
        # ARM Cortex-M 處理器
        if(CMAKE_SYSTEM_PROCESSOR MATCHES "cortex-m")
            find_path(FreeRTOS_PORT_DIR
                NAMES port.c portmacro.h
                HINTS
                    ${FreeRTOS_PORT_ROOT_DIR}/GCC/ARM_CM4F
                    ${FreeRTOS_PORT_ROOT_DIR}/GCC/ARM_CM3
                    ${FreeRTOS_PORT_ROOT_DIR}/GCC/ARM_CM0
            )
        else()
            # 其他 ARM 處理器
            find_path(FreeRTOS_PORT_DIR
                NAMES port.c portmacro.h
                HINTS
                    ${FreeRTOS_PORT_ROOT_DIR}/GCC/ARM7_LPC2000
                    ${FreeRTOS_PORT_ROOT_DIR}/GCC/ARM9_LPC2000
            )
        endif()
    elseif(CMAKE_SYSTEM_PROCESSOR MATCHES "x86")
        # x86 處理器 (通常用於模擬)
        find_path(FreeRTOS_PORT_DIR
            NAMES port.c portmacro.h
            HINTS
                ${FreeRTOS_PORT_ROOT_DIR}/GCC/POSIX
                ${FreeRTOS_PORT_ROOT_DIR}/MSVC-MingW
        )
    else()
        # 讓使用者手動設定
        message(STATUS "Unknown processor: ${CMAKE_SYSTEM_PROCESSOR}, please set FreeRTOS_PORT_DIR manually")
    endif()
endif()

# 允許使用者手動指定移植層
if(FREERTOS_PORT AND FreeRTOS_PORT_ROOT_DIR)
    find_path(FreeRTOS_PORT_DIR
        NAMES port.c portmacro.h
        HINTS ${FreeRTOS_PORT_ROOT_DIR}/${FREERTOS_PORT}
    )
endif()

# 嘗試從標頭檔中提取版本資訊
if(FreeRTOS_INCLUDE_DIR AND EXISTS "${FreeRTOS_INCLUDE_DIR}/FreeRTOS.h")
    file(READ "${FreeRTOS_INCLUDE_DIR}/FreeRTOS.h" _freertos_header)
    
    # 查找版本定義
    string(REGEX MATCH "#define[ \t]+tskKERNEL_VERSION_NUMBER[ \t]+\"([^\"]+)\"" 
           _freertos_version_match "${_freertos_header}")
    
    if(_freertos_version_match)
        set(FreeRTOS_VERSION "${CMAKE_MATCH_1}")
    else()
        # 嘗試其他版本格式
        string(REGEX MATCH "#define[ \t]+tskKERNEL_VERSION_MAJOR[ \t]+([0-9]+)" 
               _freertos_major_match "${_freertos_header}")
        string(REGEX MATCH "#define[ \t]+tskKERNEL_VERSION_MINOR[ \t]+([0-9]+)" 
               _freertos_minor_match "${_freertos_header}")
        
        if(_freertos_major_match AND _freertos_minor_match)
            set(FreeRTOS_VERSION "${CMAKE_MATCH_1}.${CMAKE_MATCH_1}")
        else()
            set(FreeRTOS_VERSION "Unknown")
        endif()
    endif()
endif()

# 收集 FreeRTOS 源碼檔案
if(FreeRTOS_SOURCE_DIR)
    file(GLOB FreeRTOS_CORE_SOURCES
        "${FreeRTOS_SOURCE_DIR}/*.c"
    )
    
    # 記憶體管理檔案 (通常選擇其中一個)
    file(GLOB FreeRTOS_HEAP_SOURCES
        "${FreeRTOS_SOURCE_DIR}/portable/MemMang/heap_*.c"
    )
    
    # 移植層檔案
    if(FreeRTOS_PORT_DIR)
        file(GLOB FreeRTOS_PORT_SOURCES
            "${FreeRTOS_PORT_DIR}/*.c"
        )
    endif()
    
    set(FreeRTOS_SOURCES 
        ${FreeRTOS_CORE_SOURCES}
        ${FreeRTOS_PORT_SOURCES}
    )
    
    # 如果沒有指定 heap 實作，使用 heap_4.c (推薦)
    if(NOT FREERTOS_HEAP_IMPLEMENTATION)
        set(FREERTOS_HEAP_IMPLEMENTATION "heap_4")
    endif()
    
    foreach(heap_file ${FreeRTOS_HEAP_SOURCES})
        get_filename_component(heap_name ${heap_file} NAME_WE)
        if(heap_name STREQUAL ${FREERTOS_HEAP_IMPLEMENTATION})
            list(APPEND FreeRTOS_SOURCES ${heap_file})
            break()
        endif()
    endforeach()
endif()

# 設定包含目錄
set(FreeRTOS_INCLUDE_DIRS "")
if(FreeRTOS_INCLUDE_DIR)
    list(APPEND FreeRTOS_INCLUDE_DIRS ${FreeRTOS_INCLUDE_DIR})
endif()

if(FreeRTOS_PORT_DIR)
    list(APPEND FreeRTOS_INCLUDE_DIRS ${FreeRTOS_PORT_DIR})
endif()

# 使用標準模組來處理 REQUIRED 和 QUIET 參數
include(FindPackageHandleStandardArgs)

find_package_handle_standard_args(FreeRTOS
    FOUND_VAR FreeRTOS_FOUND
    REQUIRED_VARS 
        FreeRTOS_INCLUDE_DIR
        FreeRTOS_SOURCE_DIR
    VERSION_VAR FreeRTOS_VERSION
)

# 建立 IMPORTED 目標 (如果找到)
if(FreeRTOS_FOUND AND NOT TARGET FreeRTOS::FreeRTOS)
    add_library(FreeRTOS::FreeRTOS INTERFACE IMPORTED)
    
    set_target_properties(FreeRTOS::FreeRTOS PROPERTIES
        INTERFACE_INCLUDE_DIRECTORIES "${FreeRTOS_INCLUDE_DIRS}"
        INTERFACE_SOURCES "${FreeRTOS_SOURCES}"
    )
    
    # 建立核心組件目標
    if(FreeRTOS_CORE_SOURCES)
        add_library(FreeRTOS::Core STATIC ${FreeRTOS_CORE_SOURCES})
        target_include_directories(FreeRTOS::Core PUBLIC ${FreeRTOS_INCLUDE_DIRS})
    endif()
    
    # 建立移植層組件目標
    if(FreeRTOS_PORT_SOURCES)
        add_library(FreeRTOS::Port STATIC ${FreeRTOS_PORT_SOURCES})
        target_include_directories(FreeRTOS::Port PUBLIC ${FreeRTOS_INCLUDE_DIRS})
    endif()
endif()

# 隱藏內部變數
mark_as_advanced(
    FreeRTOS_ROOT_DIR
    FreeRTOS_INCLUDE_DIR
    FreeRTOS_SOURCE_DIR
    FreeRTOS_PORT_ROOT_DIR
    FreeRTOS_PORT_DIR
)

# 除錯資訊
if(FreeRTOS_FOUND)
    message(STATUS "Found FreeRTOS: ${FreeRTOS_ROOT_DIR}")
    message(STATUS "FreeRTOS version: ${FreeRTOS_VERSION}")
    message(STATUS "FreeRTOS include: ${FreeRTOS_INCLUDE_DIR}")
    message(STATUS "FreeRTOS source: ${FreeRTOS_SOURCE_DIR}")
    message(STATUS "FreeRTOS port: ${FreeRTOS_PORT_DIR}")
    message(STATUS "FreeRTOS heap: ${FREERTOS_HEAP_IMPLEMENTATION}")
    message(STATUS "FreeRTOS sources: ${FreeRTOS_SOURCES}")
endif()