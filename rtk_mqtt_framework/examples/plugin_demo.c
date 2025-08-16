#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <signal.h>
#include <rtk_device_plugin.h>
#include <rtk_schema_validator.h>

/**
 * @file plugin_demo.c
 * @brief RTK MQTT Framework 插件系統示範應用
 * 
 * 展示如何載入插件、建立實例、執行診斷功能
 */

static int g_running = 1;

// 信號處理
void signal_handler(int sig) {
    printf("\n[DEMO] Received signal %d, shutting down...\n", sig);
    g_running = 0;
}

void print_usage(const char* program_name) {
    printf("RTK MQTT Framework Plugin Demo\n");
    printf("Usage: %s [options]\n", program_name);
    printf("Options:\n");
    printf("  -p <plugin_path>  插件動態庫路徑\n");
    printf("  -c <config_file>  配置檔案路徑\n");
    printf("  -n <instance_name> 實例名稱\n");
    printf("  -h                顯示說明\n");
    printf("\n");
    printf("範例:\n");
    printf("  %s -p ./wifi_router_plugin.so -c wifi_router_config.json -n router1\n", program_name);
}

void print_plugin_info(const rtk_plugin_info_t* info) {
    printf("插件資訊:\n");
    printf("  名稱: %s\n", info->name);
    printf("  版本: %s\n", info->version);
    printf("  描述: %s\n", info->description);
    printf("\n");
}

void print_device_info(rtk_plugin_instance_t* instance) {
    rtk_device_info_t device_info;
    
    int ret = RTK_PLUGIN_CALL(instance, get_device_info, 
                              instance->plugin_info->vtable->get_device_info(&device_info));
    
    if (ret == RTK_PLUGIN_SUCCESS) {
        printf("設備資訊:\n");
        printf("  ID: %s\n", device_info.id);
        printf("  型號: %s %s\n", device_info.type, device_info.model);
        printf("  序號: %s\n", device_info.serial_number);
        printf("  硬體版本: %s\n", device_info.hw_version);
        printf("  韌體版本: %s\n", device_info.fw_version);
        printf("\n");
    } else {
        printf("獲取設備資訊失敗: %s\n", rtk_plugin_get_error_string(ret));
    }
}

void demonstrate_state_reporting(rtk_plugin_instance_t* instance) {
    printf("=== 狀態回報示範 ===\n");
    
    char* state_json = NULL;
    size_t state_len = 0;
    
    int ret = RTK_PLUGIN_CALL(instance, get_state,
                              instance->plugin_info->vtable->get_state(&state_json, &state_len));
    
    if (ret == RTK_PLUGIN_SUCCESS && state_json) {
        printf("設備狀態 JSON (%zu bytes):\n%s\n\n", state_len, state_json);
        
        // 驗證 JSON Schema
        rtk_validation_result_t validation;
        if (rtk_schema_auto_validate_json(state_json, &validation) == RTK_SCHEMA_SUCCESS) {
            if (validation.is_valid) {
                printf("✓ Schema 驗證通過\n");
            } else {
                printf("✗ Schema 驗證失敗: %s\n", validation.error_message);
            }
        }
        
        rtk_plugin_safe_free_json(instance, state_json);
    } else {
        printf("取得狀態失敗: %s\n", rtk_plugin_get_error_string(ret));
    }
    printf("\n");
}

void demonstrate_telemetry(rtk_plugin_instance_t* instance) {
    printf("=== 遙測資料示範 ===\n");
    
    char* telemetry_json = NULL;
    size_t telemetry_len = 0;
    
    int ret = RTK_PLUGIN_CALL(instance, get_telemetry,
                              instance->plugin_info->vtable->get_telemetry("wifi.scan_result", 
                                                                           &telemetry_json, &telemetry_len));
    
    if (ret == RTK_PLUGIN_SUCCESS && telemetry_json) {
        printf("WiFi 掃描遙測 JSON (%zu bytes):\n%s\n\n", telemetry_len, telemetry_json);
        rtk_plugin_safe_free_json(instance, telemetry_json);
    } else if (ret == RTK_PLUGIN_ERROR_NOT_FOUND) {
        printf("遙測指標 'wifi.scan_result' 不支援\n");
    } else {
        printf("取得遙測資料失敗: %s\n", rtk_plugin_get_error_string(ret));
    }
    printf("\n");
}

void demonstrate_command_handling(rtk_plugin_instance_t* instance) {
    printf("=== 命令處理示範 ===\n");
    
    // 測試診斷命令
    const char* diagnosis_cmd = 
        "{"
        "\"id\":\"demo_diagnosis_001\","
        "\"op\":\"diagnosis.get\","
        "\"schema\":\"cmd.diagnosis.get/1.0\","
        "\"args\":{\"type\":\"wifi\",\"detail_level\":\"basic\"},"
        "\"ts\":1640995200000"
        "}";
    
    char* response_json = NULL;
    size_t response_len = 0;
    
    printf("發送診斷命令:\n%s\n\n", diagnosis_cmd);
    
    int ret = RTK_PLUGIN_CALL(instance, handle_command,
                              instance->plugin_info->vtable->handle_command(diagnosis_cmd, 
                                                                           &response_json, &response_len));
    
    if (ret == RTK_PLUGIN_SUCCESS && response_json) {
        printf("診斷回應 JSON (%zu bytes):\n%s\n", response_len, response_json);
        rtk_plugin_safe_free_json(instance, response_json);
    } else {
        printf("命令處理失敗: %s\n", rtk_plugin_get_error_string(ret));
    }
    
    printf("\n");
    
    // 測試重新啟動命令
    const char* reboot_cmd = 
        "{"
        "\"id\":\"demo_reboot_001\","
        "\"op\":\"device.reboot\","
        "\"ts\":1640995260000"
        "}";
    
    printf("發送重新啟動命令:\n%s\n\n", reboot_cmd);
    
    ret = RTK_PLUGIN_CALL(instance, handle_command,
                          instance->plugin_info->vtable->handle_command(reboot_cmd, 
                                                                       &response_json, &response_len));
    
    if (ret == RTK_PLUGIN_SUCCESS && response_json) {
        printf("重新啟動回應 JSON (%zu bytes):\n%s\n", response_len, response_json);
        rtk_plugin_safe_free_json(instance, response_json);
    } else {
        printf("重新啟動命令失敗: %s\n", rtk_plugin_get_error_string(ret));
    }
    
    printf("\n");
}

void run_plugin_health_monitor(rtk_plugin_instance_t* instance) {
    printf("=== 插件健康監控 (按 Ctrl+C 停止) ===\n");
    
    int check_count = 0;
    while (g_running) {
        int health = rtk_plugin_health_check(instance);
        
        printf("\r[%d] 健康狀態: %s   ", ++check_count, 
               (health > 0) ? "正常" : ((health == 0) ? "異常" : "錯誤"));
        fflush(stdout);
        
        sleep(2);  // 每 2 秒檢查一次
    }
    
    printf("\n健康監控已停止\n\n");
}

int main(int argc, char* argv[]) {
    printf("RTK MQTT Framework Plugin Demo v1.0\n");
    printf("=====================================\n\n");
    
    // 預設參數
    char* plugin_path = "./examples/wifi_router/wifi_router_plugin.so";
    char* config_file = "./examples/wifi_router/wifi_router_config.json";
    char* instance_name = "demo_router";
    
    // 解析命令列參數
    int opt;
    while ((opt = getopt(argc, argv, "p:c:n:h")) != -1) {
        switch (opt) {
            case 'p':
                plugin_path = optarg;
                break;
            case 'c':
                config_file = optarg;
                break;
            case 'n':
                instance_name = optarg;
                break;
            case 'h':
                print_usage(argv[0]);
                return 0;
            default:
                print_usage(argv[0]);
                return 1;
        }
    }
    
    printf("配置:\n");
    printf("  插件路徑: %s\n", plugin_path);
    printf("  配置檔案: %s\n", config_file);
    printf("  實例名稱: %s\n\n", instance_name);
    
    // 設定信號處理
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    
    // 初始化插件管理器
    printf("初始化插件管理器...\n");
    int ret = rtk_plugin_manager_init();
    if (ret != RTK_PLUGIN_SUCCESS) {
        printf("插件管理器初始化失敗: %s\n", rtk_plugin_get_error_string(ret));
        return 1;
    }
    
    // 初始化 Schema 驗證器
    printf("初始化 Schema 驗證器...\n");
    ret = rtk_schema_validator_init();
    if (ret != RTK_SCHEMA_SUCCESS) {
        printf("Schema 驗證器初始化失敗: %s\n", rtk_schema_get_error_string(ret));
        rtk_plugin_manager_cleanup();
        return 1;
    }
    
    // 載入插件
    printf("載入插件: %s\n", plugin_path);
    ret = rtk_plugin_load(plugin_path);
    if (ret != RTK_PLUGIN_SUCCESS) {
        printf("插件載入失敗: %s\n", rtk_plugin_get_error_string(ret));
        rtk_schema_validator_cleanup();
        rtk_plugin_manager_cleanup();
        return 1;
    }
    
    // 顯示載入的插件資訊
    const rtk_plugin_info_t* plugin_info = rtk_plugin_find("wifi_router");
    if (!plugin_info) {
        printf("找不到插件 'wifi_router'\n");
        rtk_schema_validator_cleanup();
        rtk_plugin_manager_cleanup();
        return 1;
    }
    
    print_plugin_info(plugin_info);
    
    // 載入配置
    rtk_plugin_config_t config;
    ret = rtk_plugin_load_config_from_file(config_file, &config);
    if (ret != RTK_PLUGIN_SUCCESS) {
        printf("載入配置失敗: %s，使用預設配置\n", rtk_plugin_get_error_string(ret));
        rtk_plugin_get_default_config(&config);
        strcpy(config.device_id, "demo_wifi_router_001");
    }
    
    // 建立插件實例
    printf("建立插件實例: %s\n", instance_name);
    rtk_plugin_instance_t* instance = rtk_plugin_create_instance(
        "wifi_router", instance_name, &config
    );
    
    if (!instance) {
        printf("插件實例建立失敗\n");
        rtk_schema_validator_cleanup();
        rtk_plugin_manager_cleanup();
        return 1;
    }
    
    // 啟動插件實例
    printf("啟動插件實例...\n");
    ret = rtk_plugin_start_instance(instance);
    if (ret != RTK_PLUGIN_SUCCESS) {
        printf("插件實例啟動失敗: %s\n", rtk_plugin_get_error_string(ret));
        rtk_plugin_destroy_instance(instance);
        rtk_schema_validator_cleanup();
        rtk_plugin_manager_cleanup();
        return 1;
    }
    
    printf("插件實例啟動成功!\n\n");
    
    // 顯示設備資訊
    print_device_info(instance);
    
    // 示範各種功能
    demonstrate_state_reporting(instance);
    demonstrate_telemetry(instance);
    demonstrate_command_handling(instance);
    
    // 執行健康監控
    run_plugin_health_monitor(instance);
    
    // 清理
    printf("清理插件實例...\n");
    rtk_plugin_stop_instance(instance);
    rtk_plugin_destroy_instance(instance);
    
    printf("清理系統資源...\n");
    rtk_schema_validator_cleanup();
    rtk_plugin_manager_cleanup();
    
    printf("程式結束\n");
    return 0;
}