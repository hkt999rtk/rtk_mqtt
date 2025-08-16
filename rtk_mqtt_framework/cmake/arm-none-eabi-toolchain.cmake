# ARM None EABI 工具鏈文件
# 用於交叉編譯到 ARM Cortex-M 微控制器 (FreeRTOS)

set(CMAKE_SYSTEM_NAME Generic)
set(CMAKE_SYSTEM_PROCESSOR arm)

# 工具鏈設置
set(TOOLCHAIN_PREFIX arm-none-eabi-)

# 編譯器
set(CMAKE_C_COMPILER ${TOOLCHAIN_PREFIX}gcc)
set(CMAKE_CXX_COMPILER ${TOOLCHAIN_PREFIX}g++)
set(CMAKE_ASM_COMPILER ${TOOLCHAIN_PREFIX}gcc)
set(CMAKE_AR ${TOOLCHAIN_PREFIX}ar)
set(CMAKE_OBJCOPY ${TOOLCHAIN_PREFIX}objcopy)
set(CMAKE_OBJDUMP ${TOOLCHAIN_PREFIX}objdump)
set(CMAKE_SIZE ${TOOLCHAIN_PREFIX}size)

# 不搜索程序和庫的根路徑
set(CMAKE_FIND_ROOT_PATH_MODE_PROGRAM NEVER)
set(CMAKE_FIND_ROOT_PATH_MODE_LIBRARY ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_INCLUDE ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_PACKAGE ONLY)

# ARM CPU 配置 (可通過 -DARM_CPU 覆蓋)
if(NOT DEFINED ARM_CPU)
    set(ARM_CPU "cortex-m4")
endif()

# FPU 配置
if(ARM_CPU STREQUAL "cortex-m0")
    set(CPU_FLAGS "-mcpu=cortex-m0 -mthumb")
    set(FPU_FLAGS "")
elseif(ARM_CPU STREQUAL "cortex-m3")
    set(CPU_FLAGS "-mcpu=cortex-m3 -mthumb")
    set(FPU_FLAGS "")
elseif(ARM_CPU STREQUAL "cortex-m4")
    set(CPU_FLAGS "-mcpu=cortex-m4 -mthumb")
    set(FPU_FLAGS "-mfloat-abi=hard -mfpu=fpv4-sp-d16")
elseif(ARM_CPU STREQUAL "cortex-m7")
    set(CPU_FLAGS "-mcpu=cortex-m7 -mthumb")
    set(FPU_FLAGS "-mfloat-abi=hard -mfpu=fpv5-d16")
elseif(ARM_CPU STREQUAL "cortex-m33")
    set(CPU_FLAGS "-mcpu=cortex-m33 -mthumb")
    set(FPU_FLAGS "-mfloat-abi=hard -mfpu=fpv5-sp-d16")
else()
    message(FATAL_ERROR "不支持的 ARM CPU: ${ARM_CPU}")
endif()

# 編譯器標志
set(COMMON_FLAGS "${CPU_FLAGS} ${FPU_FLAGS}")
set(COMMON_FLAGS "${COMMON_FLAGS} -ffunction-sections -fdata-sections")
set(COMMON_FLAGS "${COMMON_FLAGS} -Wall -Wextra")
# 簡化的 bare-metal 支持 - 只構建靜態庫
set(COMMON_FLAGS "${COMMON_FLAGS} -ffreestanding")

# C 標志
set(CMAKE_C_FLAGS_INIT "${COMMON_FLAGS} -std=c99")
set(CMAKE_C_FLAGS_DEBUG_INIT "-O0 -g3 -DDEBUG")
set(CMAKE_C_FLAGS_RELEASE_INIT "-O2 -DNDEBUG")

# C++ 標志
set(CMAKE_CXX_FLAGS_INIT "${COMMON_FLAGS} -std=c++11")
set(CMAKE_CXX_FLAGS_DEBUG_INIT "-O0 -g3 -DDEBUG")
set(CMAKE_CXX_FLAGS_RELEASE_INIT "-O2 -DNDEBUG")

# 鏈接器標志
set(CMAKE_EXE_LINKER_FLAGS_INIT "${CPU_FLAGS} ${FPU_FLAGS}")
set(CMAKE_EXE_LINKER_FLAGS_INIT "${CMAKE_EXE_LINKER_FLAGS_INIT} -Wl,--gc-sections")
set(CMAKE_EXE_LINKER_FLAGS_INIT "${CMAKE_EXE_LINKER_FLAGS_INIT} -Wl,--print-memory-usage")

# 禁用共享庫 (嵌入式系統不支持)
set(BUILD_SHARED_LIBS OFF CACHE BOOL "Build shared libraries" FORCE)

# 平台宏定義
add_definitions(-DRTK_PLATFORM_FREERTOS=1)
add_definitions(-DRTK_TARGET_ARM=1)

# 根據 CPU 添加特定宏
if(ARM_CPU MATCHES "cortex-m[47]")
    add_definitions(-DARM_MATH_CM4=1)
endif()

# FreeRTOS 相關設置
if(RTK_TARGET_FREERTOS)
    add_definitions(-DFREERTOS=1)
    add_definitions(-DRTK_USE_FREERTOS=1)
    
    # 如果指定了 FreeRTOS 路徑
    if(DEFINED FREERTOS_PATH)
        set(FREERTOS_INCLUDE_DIRS
            ${FREERTOS_PATH}/Source/include
            ${FREERTOS_PATH}/Source/portable/GCC/ARM_CM4F
        )
        include_directories(${FREERTOS_INCLUDE_DIRS})
    endif()
endif()

# 內存優化設置
add_definitions(-DRTK_USE_LIGHTWEIGHT_JSON=1)
add_definitions(-DRTK_MINIMAL_MEMORY=1)
add_definitions(-DRTK_NO_DYNAMIC_ALLOCATION=1)

# 顯示配置信息
message(STATUS "ARM 工具鏈配置:")
message(STATUS "  CPU: ${ARM_CPU}")
message(STATUS "  編譯器標志: ${COMMON_FLAGS}")
message(STATUS "  構建類型: ${CMAKE_BUILD_TYPE}")

# 跳過編譯器測試（靜態庫不需要執行檔測試）
set(CMAKE_C_COMPILER_WORKS TRUE)
set(CMAKE_CXX_COMPILER_WORKS TRUE)
set(CMAKE_ASM_COMPILER_WORKS TRUE)

# 添加連結器腳本 - 適用於靜態庫構建
set(CMAKE_EXE_LINKER_FLAGS_INIT "${CMAKE_EXE_LINKER_FLAGS_INIT} -nostartfiles -nodefaultlibs")

# 檢查工具鏈是否可用
execute_process(
    COMMAND ${CMAKE_C_COMPILER} --version
    OUTPUT_QUIET ERROR_QUIET
    RESULT_VARIABLE TOOLCHAIN_CHECK
)

if(TOOLCHAIN_CHECK)
    message(FATAL_ERROR "ARM 工具鏈不可用。請安裝 arm-none-eabi-gcc")
endif()

message(STATUS "✅ ARM 工具鏈檢查通過")