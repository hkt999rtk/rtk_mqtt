#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <getopt.h>
#include <signal.h>
#include <time.h>
#include <rtk_device_plugin.h>
#include <rtk_schema_validator.h>
#include <rtk_topic_builder.h>
#include <rtk_message_codec.h>
// Note: Temporarily disable paho adapter for minimal build
// #include <rtk_paho_adapter.h>

/**
 * @file rtk_cli.c
 * @brief RTK MQTT Framework 命令列工具
 * 
 * 提供插件管理、設備控制、診斷測試等功能
 */

// === 命令定義 ===

typedef enum {
    CMD_HELP = 0,
    CMD_VERSION,
    CMD_LIST_PLUGINS,
    CMD_LOAD_PLUGIN,
    CMD_UNLOAD_PLUGIN,
    CMD_PLUGIN_INFO,
    CMD_CREATE_INSTANCE,
    CMD_START_INSTANCE,
    CMD_STOP_INSTANCE,
    CMD_GET_STATE,
    CMD_GET_TELEMETRY,
    CMD_SEND_COMMAND,
    CMD_VALIDATE_SCHEMA,
    CMD_TEST_MQTT,
    CMD_MONITOR,
    CMD_UNKNOWN
} rtk_cli_command_t;

typedef struct {
    const char* name;
    const char* description;
    rtk_cli_command_t cmd_id;
} command_info_t;

static const command_info_t commands[] = {
    {"help", "顯示說明資訊", CMD_HELP},
    {"version", "顯示版本資訊", CMD_VERSION},
    {"list-plugins", "列出已載入的插件", CMD_LIST_PLUGINS},
    {"load-plugin", "載入插件", CMD_LOAD_PLUGIN},
    {"unload-plugin", "卸載插件", CMD_UNLOAD_PLUGIN},
    {"plugin-info", "顯示插件資訊", CMD_PLUGIN_INFO},
    {"create-instance", "建立插件實例", CMD_CREATE_INSTANCE},
    {"start-instance", "啟動插件實例", CMD_START_INSTANCE},
    {"stop-instance", "停止插件實例", CMD_STOP_INSTANCE},
    {"get-state", "獲取設備狀態", CMD_GET_STATE},
    {"get-telemetry", "獲取遙測資料", CMD_GET_TELEMETRY},
    {"send-command", "發送命令到設備", CMD_SEND_COMMAND},
    {"validate-schema", "驗證 JSON Schema", CMD_VALIDATE_SCHEMA},
    {"test-mqtt", "測試 MQTT 連線", CMD_TEST_MQTT},
    {"monitor", "監控設備狀態", CMD_MONITOR},
    {NULL, NULL, CMD_UNKNOWN}
};

// === 全域狀態 ===

static int g_running = 1;
static int g_verbose = 0;
static char g_config_file[256] = "";
static rtk_plugin_instance_t* g_current_instance = NULL;

// === 輔助函式 ===

static void signal_handler(int sig) {
    printf("\n[RTK-CLI] 收到信號 %d，正在關閉...\n", sig);
    g_running = 0;
}

static rtk_cli_command_t parse_command(const char* cmd_str) {
    for (int i = 0; commands[i].name != NULL; i++) {
        if (strcmp(cmd_str, commands[i].name) == 0) {
            return commands[i].cmd_id;
        }
    }
    return CMD_UNKNOWN;
}

static void print_usage(const char* program_name) {
    printf("RTK MQTT Framework CLI 工具\n");
    printf("用法: %s [選項] <命令> [參數...]\n\n", program_name);
    
    printf("全域選項:\n");
    printf("  -v, --verbose           詳細輸出\n");
    printf("  -c, --config <file>     指定配置檔案\n");
    printf("  -h, --help              顯示說明\n\n");
    
    printf("可用命令:\n");
    for (int i = 0; commands[i].name != NULL; i++) {
        printf("  %-18s %s\n", commands[i].name, commands[i].description);
    }
    
    printf("\n範例:\n");
    printf("  %s load-plugin ./wifi_router_plugin.so\n", program_name);
    printf("  %s create-instance wifi_router router1 -c config.json\n", program_name);
    printf("  %s get-state router1\n", program_name);
    printf("  %s send-command router1 '{\"op\":\"diagnosis.get\"}'\n", program_name);
    printf("  %s monitor router1\n", program_name);
}

static void print_version(void) {
    printf("RTK CLI 版本 1.0.0\n");
    printf("RTK MQTT Framework 版本 1.0.0\n");
    printf("Copyright (c) 2024 RTK Technologies\n");
}

// === 命令實作 ===

static int cmd_list_plugins(void) {
    const rtk_plugin_info_t* plugins[16];
    int count = rtk_plugin_list_all(plugins, 16);
    
    if (count == 0) {
        printf("目前沒有載入任何插件\n");
        return 0;
    }
    
    printf("已載入的插件 (%d 個):\n", count);
    printf("%-20s %-10s %s\n", "名稱", "版本", "描述");
    printf("%-20s %-10s %s\n", "----", "----", "----");
    
    for (int i = 0; i < count; i++) {
        printf("%-20s %-10s %s\n", 
               plugins[i]->name, 
               plugins[i]->version,
               plugins[i]->description);
    }
    
    return 0;
}

static int cmd_load_plugin(const char* plugin_path) {
    if (!plugin_path) {
        printf("錯誤: 需要指定插件路徑\n");
        return 1;
    }
    
    printf("載入插件: %s\n", plugin_path);
    
    int ret = rtk_plugin_load(plugin_path);
    if (ret != RTK_PLUGIN_SUCCESS) {
        printf("載入失敗: %s\n", rtk_plugin_get_error_string(ret));
        return 1;
    }
    
    printf("插件載入成功\n");
    return 0;
}

static int cmd_unload_plugin(const char* plugin_name) {
    if (!plugin_name) {
        printf("錯誤: 需要指定插件名稱\n");
        return 1;
    }
    
    printf("卸載插件: %s\n", plugin_name);
    
    int ret = rtk_plugin_unload(plugin_name);
    if (ret != RTK_PLUGIN_SUCCESS) {
        printf("卸載失敗: %s\n", rtk_plugin_get_error_string(ret));
        return 1;
    }
    
    printf("插件卸載成功\n");
    return 0;
}

static int cmd_plugin_info(const char* plugin_name) {
    if (!plugin_name) {
        printf("錯誤: 需要指定插件名稱\n");
        return 1;
    }
    
    const rtk_plugin_info_t* plugin = rtk_plugin_find(plugin_name);
    if (!plugin) {
        printf("找不到插件: %s\n", plugin_name);
        return 1;
    }
    
    printf("插件資訊:\n");
    printf("  名稱: %s\n", plugin->name);
    printf("  版本: %s\n", plugin->version);
    printf("  描述: %s\n", plugin->description);
    
    // 嘗試取得設備資訊和能力
    if (plugin->vtable && plugin->vtable->get_capabilities) {
        char* capabilities;
        int cap_count;
        if (plugin->vtable->get_capabilities(&capabilities, &cap_count) == RTK_PLUGIN_SUCCESS) {
            printf("  能力: ");
            for (int i = 0; i < cap_count; i++) {
                printf("%s%s", (i > 0) ? ", " : "", capabilities + i * 32);
            }
            printf("\n");
        }
    }
    
    return 0;
}

static int cmd_create_instance(const char* plugin_name, const char* instance_name) {
    if (!plugin_name || !instance_name) {
        printf("錯誤: 需要指定插件名稱和實例名稱\n");
        return 1;
    }
    
    // 載入配置
    rtk_plugin_config_t config;
    if (strlen(g_config_file) > 0) {
        int ret = rtk_plugin_load_config_from_file(g_config_file, &config);
        if (ret != RTK_PLUGIN_SUCCESS) {
            printf("警告: 載入配置失敗，使用預設配置\n");
            rtk_plugin_get_default_config(&config);
        }
    } else {
        rtk_plugin_get_default_config(&config);
    }
    
    // 設定設備 ID
    strncpy(config.device_id, instance_name, sizeof(config.device_id) - 1);
    
    printf("建立插件實例: %s (插件: %s)\n", instance_name, plugin_name);
    
    rtk_plugin_instance_t* instance = rtk_plugin_create_instance(
        plugin_name, instance_name, &config
    );
    
    if (!instance) {
        printf("建立實例失敗\n");
        return 1;
    }
    
    g_current_instance = instance;
    printf("實例建立成功\n");
    
    return 0;
}

static int cmd_start_instance(const char* instance_name) {
    if (!g_current_instance) {
        printf("錯誤: 沒有可用的實例\n");
        return 1;
    }
    
    if (instance_name && strcmp(g_current_instance->name, instance_name) != 0) {
        printf("錯誤: 實例名稱不符: %s\n", instance_name);
        return 1;
    }
    
    printf("啟動實例: %s\n", g_current_instance->name);
    
    int ret = rtk_plugin_start_instance(g_current_instance);
    if (ret != RTK_PLUGIN_SUCCESS) {
        printf("啟動失敗: %s\n", rtk_plugin_get_error_string(ret));
        return 1;
    }
    
    printf("實例啟動成功\n");
    return 0;
}

static int cmd_stop_instance(const char* instance_name) {
    if (!g_current_instance) {
        printf("錯誤: 沒有可用的實例\n");
        return 1;
    }
    
    if (instance_name && strcmp(g_current_instance->name, instance_name) != 0) {
        printf("錯誤: 實例名稱不符: %s\n", instance_name);
        return 1;
    }
    
    printf("停止實例: %s\n", g_current_instance->name);
    
    int ret = rtk_plugin_stop_instance(g_current_instance);
    if (ret != RTK_PLUGIN_SUCCESS) {
        printf("停止失敗: %s\n", rtk_plugin_get_error_string(ret));
        return 1;
    }
    
    printf("實例已停止\n");
    return 0;
}

static int cmd_get_state(const char* instance_name) {
    if (!g_current_instance) {
        printf("錯誤: 沒有可用的實例\n");
        return 1;
    }
    
    if (instance_name && strcmp(g_current_instance->name, instance_name) != 0) {
        printf("錯誤: 實例名稱不符: %s\n", instance_name);
        return 1;
    }
    
    char* json_state = NULL;
    size_t json_len = 0;
    
    int ret = RTK_PLUGIN_CALL(g_current_instance, get_state,
                              g_current_instance->plugin_info->vtable->get_state(&json_state, &json_len));
    
    if (ret != RTK_PLUGIN_SUCCESS) {
        printf("取得狀態失敗: %s\n", rtk_plugin_get_error_string(ret));
        return 1;
    }
    
    if (json_state) {
        printf("設備狀態 (%zu bytes):\n", json_len);
        printf("%s\n", json_state);
        
        // 驗證 Schema
        if (g_verbose) {
            rtk_validation_result_t validation;
            if (rtk_schema_auto_validate_json(json_state, &validation) == RTK_SCHEMA_SUCCESS) {
                printf("\nSchema 驗證: %s\n", validation.is_valid ? "通過" : "失敗");
                if (!validation.is_valid) {
                    printf("錯誤: %s\n", validation.error_message);
                }
            }
        }
        
        rtk_plugin_safe_free_json(g_current_instance, json_state);
    }
    
    return 0;
}

static int cmd_get_telemetry(const char* instance_name, const char* metric) {
    if (!g_current_instance) {
        printf("錯誤: 沒有可用的實例\n");
        return 1;
    }
    
    if (!metric) {
        printf("錯誤: 需要指定遙測指標\n");
        return 1;
    }
    
    char* json_data = NULL;
    size_t json_len = 0;
    
    int ret = RTK_PLUGIN_CALL(g_current_instance, get_telemetry,
                              g_current_instance->plugin_info->vtable->get_telemetry(metric, &json_data, &json_len));
    
    if (ret != RTK_PLUGIN_SUCCESS) {
        if (ret == RTK_PLUGIN_ERROR_NOT_FOUND) {
            printf("不支援的遙測指標: %s\n", metric);
        } else {
            printf("取得遙測失敗: %s\n", rtk_plugin_get_error_string(ret));
        }
        return 1;
    }
    
    if (json_data) {
        printf("遙測資料 '%s' (%zu bytes):\n", metric, json_len);
        printf("%s\n", json_data);
        rtk_plugin_safe_free_json(g_current_instance, json_data);
    }
    
    return 0;
}

static int cmd_send_command(const char* instance_name, const char* cmd_json) {
    if (!g_current_instance) {
        printf("錯誤: 沒有可用的實例\n");
        return 1;
    }
    
    if (!cmd_json) {
        printf("錯誤: 需要指定命令 JSON\n");
        return 1;
    }
    
    char* response_json = NULL;
    size_t response_len = 0;
    
    printf("發送命令: %s\n", cmd_json);
    
    int ret = RTK_PLUGIN_CALL(g_current_instance, handle_command,
                              g_current_instance->plugin_info->vtable->handle_command(cmd_json, &response_json, &response_len));
    
    if (ret != RTK_PLUGIN_SUCCESS) {
        printf("命令執行失敗: %s\n", rtk_plugin_get_error_string(ret));
        return 1;
    }
    
    if (response_json) {
        printf("命令回應 (%zu bytes):\n", response_len);
        printf("%s\n", response_json);
        rtk_plugin_safe_free_json(g_current_instance, response_json);
    }
    
    return 0;
}

static int cmd_validate_schema(const char* json_file, const char* schema_name) {
    if (!json_file) {
        printf("錯誤: 需要指定 JSON 檔案\n");
        return 1;
    }
    
    FILE* file = fopen(json_file, "r");
    if (!file) {
        printf("無法開啟檔案: %s\n", json_file);
        return 1;
    }
    
    // 讀取檔案內容
    fseek(file, 0, SEEK_END);
    long file_size = ftell(file);
    fseek(file, 0, SEEK_SET);
    
    char* json_content = malloc(file_size + 1);
    if (!json_content) {
        fclose(file);
        printf("記憶體配置失敗\n");
        return 1;
    }
    
    fread(json_content, 1, file_size, file);
    json_content[file_size] = '\0';
    fclose(file);
    
    // 驗證 Schema
    rtk_validation_result_t result;
    int ret;
    
    if (schema_name) {
        ret = rtk_schema_validate_json(json_content, schema_name, &result);
    } else {
        ret = rtk_schema_auto_validate_json(json_content, &result);
    }
    
    printf("Schema 驗證結果:\n");
    printf("  檔案: %s\n", json_file);
    if (schema_name) {
        printf("  Schema: %s\n", schema_name);
    }
    printf("  狀態: %s\n", result.is_valid ? "通過" : "失敗");
    
    if (!result.is_valid) {
        printf("  錯誤: %s\n", result.error_message);
        if (strlen(result.error_path) > 0) {
            printf("  路徑: %s\n", result.error_path);
        }
    }
    
    free(json_content);
    return result.is_valid ? 0 : 1;
}

static int cmd_monitor(const char* instance_name) {
    if (!g_current_instance) {
        printf("錯誤: 沒有可用的實例\n");
        return 1;
    }
    
    printf("監控實例: %s (按 Ctrl+C 停止)\n", g_current_instance->name);
    printf("%-20s %-10s %-15s %s\n", "時間", "健康狀態", "運行狀態", "備註");
    printf("%-20s %-10s %-15s %s\n", "----", "----", "----", "----");
    
    int check_count = 0;
    while (g_running) {
        time_t now = time(NULL);
        struct tm* tm_info = localtime(&now);
        char time_str[32];
        strftime(time_str, sizeof(time_str), "%H:%M:%S", tm_info);
        
        int health = rtk_plugin_health_check(g_current_instance);
        const char* health_str = (health > 0) ? "正常" : ((health == 0) ? "異常" : "錯誤");
        const char* running_str = g_current_instance->is_running ? "運行中" : "已停止";
        
        printf("%-20s %-10s %-15s #%d\n", time_str, health_str, running_str, ++check_count);
        
        // 每 10 次檢查輸出一次狀態
        if (check_count % 10 == 0) {
            printf("  -> 取得狀態資料...\n");
            cmd_get_state(instance_name);
            printf("\n");
        }
        
        sleep(3);  // 每 3 秒檢查一次
    }
    
    printf("\n監控已停止\n");
    return 0;
}

// === 主程式 ===

int main(int argc, char* argv[]) {
    printf("RTK MQTT Framework CLI v1.0.0\n");
    printf("=============================\n\n");
    
    // 設定信號處理
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    
    // 解析命令列參數
    static struct option long_options[] = {
        {"verbose", no_argument, 0, 'v'},
        {"config", required_argument, 0, 'c'},
        {"help", no_argument, 0, 'h'},
        {0, 0, 0, 0}
    };
    
    int opt;
    while ((opt = getopt_long(argc, argv, "vc:h", long_options, NULL)) != -1) {
        switch (opt) {
            case 'v':
                g_verbose = 1;
                break;
            case 'c':
                strncpy(g_config_file, optarg, sizeof(g_config_file) - 1);
                break;
            case 'h':
                print_usage(argv[0]);
                return 0;
            default:
                print_usage(argv[0]);
                return 1;
        }
    }
    
    if (optind >= argc) {
        print_usage(argv[0]);
        return 1;
    }
    
    const char* command = argv[optind];
    rtk_cli_command_t cmd_id = parse_command(command);
    
    if (cmd_id == CMD_UNKNOWN) {
        printf("未知命令: %s\n", command);
        print_usage(argv[0]);
        return 1;
    }
    
    // 初始化系統 (除了 help 和 version 命令)
    if (cmd_id != CMD_HELP && cmd_id != CMD_VERSION) {
        printf("初始化插件管理器...\n");
        int ret = rtk_plugin_manager_init();
        if (ret != RTK_PLUGIN_SUCCESS) {
            printf("插件管理器初始化失敗: %s\n", rtk_plugin_get_error_string(ret));
            return 1;
        }
        
        printf("初始化 Schema 驗證器...\n");
        ret = rtk_schema_validator_init();
        if (ret != RTK_SCHEMA_SUCCESS) {
            printf("Schema 驗證器初始化失敗: %s\n", rtk_schema_get_error_string(ret));
            rtk_plugin_manager_cleanup();
            return 1;
        }
        
        if (g_verbose) {
            printf("系統初始化完成\n\n");
        }
    }
    
    // 執行命令
    int result = 1;
    
    switch (cmd_id) {
        case CMD_HELP:
            print_usage(argv[0]);
            result = 0;
            break;
            
        case CMD_VERSION:
            print_version();
            result = 0;
            break;
            
        case CMD_LIST_PLUGINS:
            result = cmd_list_plugins();
            break;
            
        case CMD_LOAD_PLUGIN:
            if (optind + 1 < argc) {
                result = cmd_load_plugin(argv[optind + 1]);
            } else {
                printf("錯誤: load-plugin 需要插件路徑參數\n");
            }
            break;
            
        case CMD_UNLOAD_PLUGIN:
            if (optind + 1 < argc) {
                result = cmd_unload_plugin(argv[optind + 1]);
            } else {
                printf("錯誤: unload-plugin 需要插件名稱參數\n");
            }
            break;
            
        case CMD_PLUGIN_INFO:
            if (optind + 1 < argc) {
                result = cmd_plugin_info(argv[optind + 1]);
            } else {
                printf("錯誤: plugin-info 需要插件名稱參數\n");
            }
            break;
            
        case CMD_CREATE_INSTANCE:
            if (optind + 2 < argc) {
                result = cmd_create_instance(argv[optind + 1], argv[optind + 2]);
            } else {
                printf("錯誤: create-instance 需要插件名稱和實例名稱參數\n");
            }
            break;
            
        case CMD_START_INSTANCE:
            result = cmd_start_instance((optind + 1 < argc) ? argv[optind + 1] : NULL);
            break;
            
        case CMD_STOP_INSTANCE:
            result = cmd_stop_instance((optind + 1 < argc) ? argv[optind + 1] : NULL);
            break;
            
        case CMD_GET_STATE:
            result = cmd_get_state((optind + 1 < argc) ? argv[optind + 1] : NULL);
            break;
            
        case CMD_GET_TELEMETRY:
            if (optind + 1 < argc) {
                result = cmd_get_telemetry(NULL, argv[optind + 1]);
            } else {
                printf("錯誤: get-telemetry 需要遙測指標參數\n");
            }
            break;
            
        case CMD_SEND_COMMAND:
            if (optind + 1 < argc) {
                result = cmd_send_command(NULL, argv[optind + 1]);
            } else {
                printf("錯誤: send-command 需要 JSON 命令參數\n");
            }
            break;
            
        case CMD_VALIDATE_SCHEMA:
            if (optind + 1 < argc) {
                const char* schema_name = (optind + 2 < argc) ? argv[optind + 2] : NULL;
                result = cmd_validate_schema(argv[optind + 1], schema_name);
            } else {
                printf("錯誤: validate-schema 需要 JSON 檔案參數\n");
            }
            break;
            
        case CMD_MONITOR:
            result = cmd_monitor((optind + 1 < argc) ? argv[optind + 1] : NULL);
            break;
            
        default:
            printf("命令尚未實作: %s\n", command);
            break;
    }
    
    // 清理
    if (cmd_id != CMD_HELP && cmd_id != CMD_VERSION) {
        if (g_current_instance) {
            rtk_plugin_stop_instance(g_current_instance);
            rtk_plugin_destroy_instance(g_current_instance);
        }
        
        rtk_schema_validator_cleanup();
        rtk_plugin_manager_cleanup();
        
        if (g_verbose) {
            printf("系統清理完成\n");
        }
    }
    
    return result;
}