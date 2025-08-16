# FindWindowsSDK.cmake
# 
# 尋找 Windows SDK 和相關組件
#
# 使用方法:
#   find_package(WindowsSDK REQUIRED)
#
# 定義的變數:
#   WindowsSDK_FOUND         - 如果找到 Windows SDK 則為 TRUE
#   WindowsSDK_INCLUDE_DIRS  - Windows SDK 的包含目錄
#   WindowsSDK_LIBRARIES     - Windows SDK 的函式庫
#   WindowsSDK_VERSION       - Windows SDK 的版本
#   WindowsSDK_ROOT_DIR      - Windows SDK 的根目錄

# 只在 Windows 平台上執行
if(NOT WIN32)
    set(WindowsSDK_FOUND FALSE)
    return()
endif()

# 查找 Windows SDK 根目錄
set(WindowsSDK_SEARCH_PATHS
    "C:/Program Files (x86)/Windows Kits/10"
    "C:/Program Files/Windows Kits/10"
    "C:/Program Files (x86)/Windows Kits/8.1"
    "C:/Program Files/Windows Kits/8.1"
    "C:/Program Files (x86)/Windows Kits/8.0"
    "C:/Program Files/Windows Kits/8.0"
    "C:/Program Files (x86)/Microsoft SDKs/Windows"
    "C:/Program Files/Microsoft SDKs/Windows"
)

find_path(WindowsSDK_ROOT_DIR
    NAMES Include/windows.h Include/Windows.h
    PATHS ${WindowsSDK_SEARCH_PATHS}
    PATH_SUFFIXES 
        ""
        "v10.0"
        "v8.1"
        "v8.0"
        "v7.1A"
        "v7.0A"
)

# 如果沒有找到，嘗試從註冊表獲取
if(NOT WindowsSDK_ROOT_DIR AND WIN32)
    # 嘗試 Windows 10 SDK
    get_filename_component(WindowsSDK_ROOT_DIR 
        "[HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows Kits\\Installed Roots;KitsRoot10]" 
        ABSOLUTE)
    
    if(NOT EXISTS "${WindowsSDK_ROOT_DIR}")
        # 嘗試 Windows 8.1 SDK
        get_filename_component(WindowsSDK_ROOT_DIR 
            "[HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows Kits\\Installed Roots;KitsRoot81]" 
            ABSOLUTE)
    endif()
    
    if(NOT EXISTS "${WindowsSDK_ROOT_DIR}")
        # 嘗試 Windows 8.0 SDK
        get_filename_component(WindowsSDK_ROOT_DIR 
            "[HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows Kits\\Installed Roots;KitsRoot]" 
            ABSOLUTE)
    endif()
endif()

# 查找版本
if(WindowsSDK_ROOT_DIR)
    # 對於 Windows 10 SDK，查找最新版本
    if(EXISTS "${WindowsSDK_ROOT_DIR}/Include")
        file(GLOB WindowsSDK_VERSION_DIRS "${WindowsSDK_ROOT_DIR}/Include/10.*")
        if(WindowsSDK_VERSION_DIRS)
            list(SORT WindowsSDK_VERSION_DIRS)
            list(REVERSE WindowsSDK_VERSION_DIRS)
            list(GET WindowsSDK_VERSION_DIRS 0 WindowsSDK_LATEST_VERSION_DIR)
            get_filename_component(WindowsSDK_VERSION 
                "${WindowsSDK_LATEST_VERSION_DIR}" NAME)
        endif()
    endif()
    
    # 如果沒有找到版本目錄，檢查是否為舊版 SDK
    if(NOT WindowsSDK_VERSION)
        if(EXISTS "${WindowsSDK_ROOT_DIR}/Include/um/windows.h")
            set(WindowsSDK_VERSION "8.1")
        elseif(EXISTS "${WindowsSDK_ROOT_DIR}/Include/windows.h")
            set(WindowsSDK_VERSION "7.1")
        endif()
    endif()
endif()

# 設定包含目錄
set(WindowsSDK_INCLUDE_DIRS "")
if(WindowsSDK_ROOT_DIR)
    if(WindowsSDK_VERSION MATCHES "^10\\.")
        # Windows 10 SDK
        list(APPEND WindowsSDK_INCLUDE_DIRS 
            "${WindowsSDK_ROOT_DIR}/Include/${WindowsSDK_VERSION}/um"
            "${WindowsSDK_ROOT_DIR}/Include/${WindowsSDK_VERSION}/shared"
            "${WindowsSDK_ROOT_DIR}/Include/${WindowsSDK_VERSION}/winrt"
            "${WindowsSDK_ROOT_DIR}/Include/${WindowsSDK_VERSION}/cppwinrt"
        )
    elseif(WindowsSDK_VERSION STREQUAL "8.1")
        # Windows 8.1 SDK
        list(APPEND WindowsSDK_INCLUDE_DIRS 
            "${WindowsSDK_ROOT_DIR}/Include/um"
            "${WindowsSDK_ROOT_DIR}/Include/shared"
            "${WindowsSDK_ROOT_DIR}/Include/winrt"
        )
    else()
        # 舊版 SDK
        list(APPEND WindowsSDK_INCLUDE_DIRS "${WindowsSDK_ROOT_DIR}/Include")
    endif()
endif()

# 設定函式庫目錄和函式庫
set(WindowsSDK_LIBRARIES "")
if(WindowsSDK_ROOT_DIR)
    # 決定架構
    if(CMAKE_SIZEOF_VOID_P EQUAL 8)
        set(WindowsSDK_ARCH "x64")
    else()
        set(WindowsSDK_ARCH "x86")
    endif()
    
    # 查找函式庫目錄
    if(WindowsSDK_VERSION MATCHES "^10\\.")
        set(WindowsSDK_LIB_DIR "${WindowsSDK_ROOT_DIR}/Lib/${WindowsSDK_VERSION}/um/${WindowsSDK_ARCH}")
    elseif(WindowsSDK_VERSION STREQUAL "8.1")
        set(WindowsSDK_LIB_DIR "${WindowsSDK_ROOT_DIR}/Lib/winv6.3/um/${WindowsSDK_ARCH}")
    else()
        set(WindowsSDK_LIB_DIR "${WindowsSDK_ROOT_DIR}/Lib/${WindowsSDK_ARCH}")
    endif()
    
    # 常用的 Windows 函式庫
    set(WindowsSDK_COMMON_LIBS
        kernel32 user32 gdi32 winspool comdlg32 advapi32 shell32
        ole32 oleaut32 uuid odbc32 odbccp32 ws2_32 wsock32
    )
    
    foreach(lib ${WindowsSDK_COMMON_LIBS})
        find_library(WindowsSDK_${lib}_LIBRARY
            NAMES ${lib}
            PATHS ${WindowsSDK_LIB_DIR}
            NO_DEFAULT_PATH
        )
        if(WindowsSDK_${lib}_LIBRARY)
            list(APPEND WindowsSDK_LIBRARIES ${WindowsSDK_${lib}_LIBRARY})
        endif()
    endforeach()
endif()

# 使用標準模組來處理 REQUIRED 和 QUIET 參數
include(FindPackageHandleStandardArgs)

find_package_handle_standard_args(WindowsSDK
    FOUND_VAR WindowsSDK_FOUND
    REQUIRED_VARS 
        WindowsSDK_ROOT_DIR
        WindowsSDK_INCLUDE_DIRS
    VERSION_VAR WindowsSDK_VERSION
)

# 建立 IMPORTED 目標 (如果找到)
if(WindowsSDK_FOUND AND NOT TARGET WindowsSDK::WindowsSDK)
    add_library(WindowsSDK::WindowsSDK INTERFACE IMPORTED)
    
    set_target_properties(WindowsSDK::WindowsSDK PROPERTIES
        INTERFACE_INCLUDE_DIRECTORIES "${WindowsSDK_INCLUDE_DIRS}"
        INTERFACE_LINK_LIBRARIES "${WindowsSDK_LIBRARIES}"
    )
    
    # 建立個別函式庫目標
    foreach(lib ${WindowsSDK_COMMON_LIBS})
        if(WindowsSDK_${lib}_LIBRARY AND NOT TARGET WindowsSDK::${lib})
            add_library(WindowsSDK::${lib} UNKNOWN IMPORTED)
            set_target_properties(WindowsSDK::${lib} PROPERTIES
                IMPORTED_LOCATION "${WindowsSDK_${lib}_LIBRARY}"
            )
        endif()
    endforeach()
endif()

# 隱藏內部變數
mark_as_advanced(
    WindowsSDK_ROOT_DIR
    WindowsSDK_LIB_DIR
)

foreach(lib ${WindowsSDK_COMMON_LIBS})
    mark_as_advanced(WindowsSDK_${lib}_LIBRARY)
endforeach()

# 除錯資訊
if(WindowsSDK_FOUND)
    message(STATUS "Found Windows SDK: ${WindowsSDK_ROOT_DIR}")
    message(STATUS "Windows SDK version: ${WindowsSDK_VERSION}")
    message(STATUS "Windows SDK includes: ${WindowsSDK_INCLUDE_DIRS}")
    message(STATUS "Windows SDK libraries: ${WindowsSDK_LIB_DIR}")
    message(STATUS "Windows SDK arch: ${WindowsSDK_ARCH}")
endif()