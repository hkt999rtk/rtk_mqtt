/**
 * @file smart_thermostat_plugin.c
 * @brief Smart Thermostat Plugin Example for RTK MQTT Framework Go DLL
 * 
 * This plugin demonstrates a smart thermostat device that uses the
 * RTK MQTT Framework Go DLL for communication. It simulates:
 * - Temperature monitoring
 * - Heating/cooling control
 * - Energy consumption tracking
 * - Smart scheduling events
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <math.h>

// Include RTK MQTT Framework DLL header
#include "rtk_mqtt_framework.h"

#ifdef _WIN32
    #include <windows.h>
    #define sleep(x) Sleep((x) * 1000)
#else
    #include <unistd.h>
    #include <dlfcn.h>
#endif

// Smart Thermostat Configuration
typedef struct {
    double target_temperature;
    double current_temperature;
    double tolerance;
    int heating_enabled;
    int cooling_enabled;
    double power_consumption;
    time_t last_adjustment;
    char schedule_mode[32]; // "auto", "manual", "eco", "comfort"
} smart_thermostat_state_t;

// Plugin State
typedef struct {
    rtk_client_handle_t rtk_client;
    smart_thermostat_state_t thermostat;
    int running;
    time_t start_time;
    int message_count;
} plugin_state_t;

// Global plugin state
static plugin_state_t g_plugin_state = {0};

/**
 * @brief Initialize smart thermostat with default values
 */
void init_thermostat(smart_thermostat_state_t* thermostat) {
    thermostat->target_temperature = 22.0;  // 22°C target
    thermostat->current_temperature = 20.5; // Starting temperature
    thermostat->tolerance = 0.5;             // ±0.5°C tolerance
    thermostat->heating_enabled = 0;
    thermostat->cooling_enabled = 0;
    thermostat->power_consumption = 0.0;
    thermostat->last_adjustment = time(NULL);
    strcpy(thermostat->schedule_mode, "auto");
}

/**
 * @brief Simulate temperature changes based on heating/cooling
 */
void simulate_temperature_control(smart_thermostat_state_t* thermostat) {
    double ambient_temp = 18.0; // Ambient temperature
    double time_factor = 0.1;   // Rate of temperature change
    
    // Calculate target vs current difference
    double temp_diff = thermostat->target_temperature - thermostat->current_temperature;
    
    // Decide heating/cooling action
    if (temp_diff > thermostat->tolerance) {
        // Need heating
        thermostat->heating_enabled = 1;
        thermostat->cooling_enabled = 0;
        thermostat->power_consumption = 1500.0; // 1.5kW heating
        thermostat->current_temperature += time_factor * 0.8;
    } else if (temp_diff < -thermostat->tolerance) {
        // Need cooling
        thermostat->heating_enabled = 0;
        thermostat->cooling_enabled = 1;
        thermostat->power_consumption = 800.0; // 0.8kW cooling
        thermostat->current_temperature -= time_factor * 0.6;
    } else {
        // In comfort zone
        thermostat->heating_enabled = 0;
        thermostat->cooling_enabled = 0;
        thermostat->power_consumption = 25.0; // Standby power
        // Drift towards ambient temperature
        double drift = (ambient_temp - thermostat->current_temperature) * 0.02;
        thermostat->current_temperature += drift;
    }
    
    // Add some random variation
    double noise = ((double)rand() / RAND_MAX - 0.5) * 0.2;
    thermostat->current_temperature += noise;
    
    // Update last adjustment time if action was taken
    if (thermostat->heating_enabled || thermostat->cooling_enabled) {
        thermostat->last_adjustment = time(NULL);
    }
}

/**
 * @brief Publish device state to MQTT
 */
int publish_thermostat_state(rtk_client_handle_t client, const smart_thermostat_state_t* thermostat) {
    rtk_device_state_t state = {0};
    
    // Determine overall status
    if (thermostat->heating_enabled) {
        strcpy(state.status, "heating");
    } else if (thermostat->cooling_enabled) {
        strcpy(state.status, "cooling");
    } else {
        strcpy(state.status, "idle");
    }
    
    // Determine health status
    double temp_diff = fabs(thermostat->target_temperature - thermostat->current_temperature);
    if (temp_diff <= thermostat->tolerance) {
        strcpy(state.health, "optimal");
    } else if (temp_diff <= thermostat->tolerance * 2) {
        strcpy(state.health, "adjusting");
    } else {
        strcpy(state.health, "warning");
    }
    
    state.uptime = time(NULL) - g_plugin_state.start_time;
    state.last_seen = time(NULL);
    
    return rtk_publish_state(client, &state);
}

/**
 * @brief Log telemetry data (simplified - no actual publishing)
 */
void log_telemetry(const smart_thermostat_state_t* thermostat) {
    printf("  Temperature: %.1f°C (target: %.1f°C)\n", 
           thermostat->current_temperature, thermostat->target_temperature);
    printf("  Power consumption: %.0fW\n", thermostat->power_consumption);
    printf("  Mode: %s\n", thermostat->schedule_mode);
}

/**
 * @brief Main plugin operation loop
 */
void* plugin_main_loop(void* arg) {
    plugin_state_t* state = (plugin_state_t*)arg;
    int cycle_count = 0;
    time_t last_state_publish = 0;
    time_t last_telemetry_publish = 0;
    
    printf("Smart Thermostat Plugin: Main loop started\n");
    
    while (state->running) {
        time_t now = time(NULL);
        
        // Simulate thermostat operation
        simulate_temperature_control(&state->thermostat);
        
        // Publish state every 30 seconds
        if (now - last_state_publish >= 30) {
            if (publish_thermostat_state(state->rtk_client, &state->thermostat) == RTK_SUCCESS) {
                printf("Thermostat state published (Temp: %.1f°C, Target: %.1f°C, Status: %s%s)\n",
                       state->thermostat.current_temperature,
                       state->thermostat.target_temperature,
                       state->thermostat.heating_enabled ? "heating" : 
                       (state->thermostat.cooling_enabled ? "cooling" : "idle"),
                       state->thermostat.heating_enabled || state->thermostat.cooling_enabled ? 
                       " [ACTIVE]" : "");
                last_state_publish = now;
                state->message_count++;
            }
        }
        
        // Log telemetry every 60 seconds  
        if (now - last_telemetry_publish >= 60) {
            log_telemetry(&state->thermostat);
            last_telemetry_publish = now;
        }
        
        cycle_count++;
        
        sleep(3); // 3 second cycle
    }
    
    printf("Smart Thermostat Plugin: Main loop ended\n");
    return NULL;
}

/**
 * @brief Initialize plugin
 */
int plugin_init(const char* device_id, const char* broker_host, int broker_port) {
    printf("=== Smart Thermostat Plugin Initialization ===\n");
    
    // Initialize random seed
    srand(time(NULL));
    
    // Create RTK client
    g_plugin_state.rtk_client = rtk_create_client();
    if (g_plugin_state.rtk_client == 0) {
        printf("ERROR: Failed to create RTK MQTT client\n");
        return -1;
    }
    
    // Configure MQTT
    rtk_mqtt_config_t mqtt_config = {0};
    strncpy(mqtt_config.broker_host, broker_host, sizeof(mqtt_config.broker_host) - 1);
    mqtt_config.broker_port = broker_port;
    snprintf(mqtt_config.client_id, sizeof(mqtt_config.client_id), "smart_thermostat_%s", device_id);
    
    if (rtk_configure_mqtt(g_plugin_state.rtk_client, &mqtt_config) != RTK_SUCCESS) {
        printf("ERROR: Failed to configure MQTT\n");
        return -1;
    }
    
    // Set device info
    rtk_device_info_t device_info = {0};
    strncpy(device_info.id, device_id, sizeof(device_info.id) - 1);
    strcpy(device_info.device_type, "smart_thermostat");
    strcpy(device_info.name, "Smart Thermostat Pro");
    strcpy(device_info.version, "2.1.0");
    
    if (rtk_set_device_info(g_plugin_state.rtk_client, &device_info) != RTK_SUCCESS) {
        printf("ERROR: Failed to set device info\n");
        return -1;
    }
    
    // Connect to MQTT broker
    if (rtk_connect(g_plugin_state.rtk_client) != RTK_SUCCESS) {
        printf("ERROR: Failed to connect to MQTT broker\n");
        return -1;
    }
    
    // Initialize thermostat
    init_thermostat(&g_plugin_state.thermostat);
    
    // Initialize plugin state
    g_plugin_state.running = 1;
    g_plugin_state.start_time = time(NULL);
    g_plugin_state.message_count = 0;
    
    printf("✓ Smart Thermostat Plugin initialized successfully\n");
    printf("  Device ID: %s\n", device_id);
    printf("  MQTT Broker: %s:%d\n", broker_host, broker_port);
    printf("  RTK Framework Version: %s\n", rtk_get_version());
    
    return 0;
}

/**
 * @brief Cleanup plugin
 */
void plugin_cleanup() {
    printf("Smart Thermostat Plugin: Cleaning up...\n");
    
    g_plugin_state.running = 0;
    
    if (g_plugin_state.rtk_client) {
        rtk_disconnect(g_plugin_state.rtk_client);
        rtk_destroy_client(g_plugin_state.rtk_client);
    }
    
    printf("Smart Thermostat Plugin: Cleanup completed\n");
}

/**
 * @brief Main function - standalone plugin executable
 */
int main(int argc, char* argv[]) {
    printf("=== RTK MQTT Framework - Smart Thermostat Plugin ===\n");
    printf("This plugin simulates a smart thermostat using the RTK MQTT Framework Go DLL\n\n");
    
    // Parse command line arguments
    const char* device_id = (argc > 1) ? argv[1] : "smart_thermostat_001";
    const char* broker_host = (argc > 2) ? argv[2] : "test.mosquitto.org";
    int broker_port = (argc > 3) ? atoi(argv[3]) : 1883;
    
    printf("Configuration:\n");
    printf("  Device ID: %s\n", device_id);
    printf("  MQTT Broker: %s:%d\n", broker_host, broker_port);
    printf("\n");
    
    // Initialize plugin
    if (plugin_init(device_id, broker_host, broker_port) != 0) {
        printf("ERROR: Plugin initialization failed\n");
        return -1;
    }
    
    // Start main operation loop
    printf("\nStarting thermostat operation...\n");
    printf("Press Ctrl+C to stop\n\n");
    
    plugin_main_loop(&g_plugin_state);
    
    // Cleanup
    plugin_cleanup();
    
    printf("\nSmart Thermostat Plugin finished.\n");
    return 0;
}