/**
 * @file main.cpp
 * @brief Windows C++ Demo for RTK MQTT Framework Go DLL
 * 
 * This example demonstrates how to use the RTK MQTT Framework Go DLL
 * from Windows C++ code. It shows basic client management, configuration,
 * and message publishing functionality.
 */

#include <iostream>
#include <string>
#include <cstring>
#include <ctime>
#include <thread>
#include <chrono>

// Include RTK MQTT Framework DLL header
#include "rtk_mqtt_framework.h"

#ifdef _WIN32
    #include <windows.h>
    #define LIBRARY_HANDLE HMODULE
    #define LOAD_LIBRARY(name) LoadLibraryA(name)
    #define GET_FUNCTION(lib, name) GetProcAddress(lib, name)
    #define CLOSE_LIBRARY(lib) FreeLibrary(lib)
#else
    #include <dlfcn.h>
    #define LIBRARY_HANDLE void*
    #define LOAD_LIBRARY(name) dlopen(name, RTLD_LAZY)
    #define GET_FUNCTION(lib, name) dlsym(lib, name)
    #define CLOSE_LIBRARY(lib) dlclose(lib)
#endif

// Function pointer types for DLL functions  
typedef rtk_client_handle_t (*rtk_create_client_func)();
typedef int (*rtk_destroy_client_func)(rtk_client_handle_t);
typedef int (*rtk_configure_mqtt_func)(rtk_client_handle_t, const rtk_mqtt_config_t*);
typedef int (*rtk_set_device_info_func)(rtk_client_handle_t, const rtk_device_info_t*);
typedef int (*rtk_connect_func)(rtk_client_handle_t);
typedef int (*rtk_disconnect_func)(rtk_client_handle_t);
typedef int (*rtk_publish_state_func)(rtk_client_handle_t, const rtk_device_state_t*);
typedef int (*rtk_is_connected_func)(rtk_client_handle_t);
typedef int (*rtk_get_client_count_func)();
typedef const char* (*rtk_get_version_func)();
typedef const char* (*rtk_get_last_error_func)();

/**
 * @class RTKMQTTClient
 * @brief C++ wrapper class for RTK MQTT Framework Go DLL
 */
class RTKMQTTClient {
private:
    LIBRARY_HANDLE dll_handle;
    rtk_client_handle_t client_handle;
    
    // Function pointers
    rtk_create_client_func create_client;
    rtk_destroy_client_func destroy_client;
    rtk_configure_mqtt_func configure_mqtt;
    rtk_set_device_info_func set_device_info;
    rtk_connect_func connect;
    rtk_disconnect_func disconnect;
    rtk_publish_state_func publish_state;
    rtk_is_connected_func is_connected;
    rtk_get_client_count_func get_client_count;
    rtk_get_version_func get_version;
    rtk_get_last_error_func get_last_error;

    bool LoadDLL(const std::string& dll_path) {
        dll_handle = LOAD_LIBRARY(dll_path.c_str());
        if (!dll_handle) {
            std::cerr << "Failed to load DLL: " << dll_path << std::endl;
            return false;
        }

        // Load function pointers
        create_client = (rtk_create_client_func)GET_FUNCTION(dll_handle, "rtk_create_client");
        destroy_client = (rtk_destroy_client_func)GET_FUNCTION(dll_handle, "rtk_destroy_client");
        configure_mqtt = (rtk_configure_mqtt_func)GET_FUNCTION(dll_handle, "rtk_configure_mqtt");
        set_device_info = (rtk_set_device_info_func)GET_FUNCTION(dll_handle, "rtk_set_device_info");
        connect = (rtk_connect_func)GET_FUNCTION(dll_handle, "rtk_connect");
        disconnect = (rtk_disconnect_func)GET_FUNCTION(dll_handle, "rtk_disconnect");
        publish_state = (rtk_publish_state_func)GET_FUNCTION(dll_handle, "rtk_publish_state");
        is_connected = (rtk_is_connected_func)GET_FUNCTION(dll_handle, "rtk_is_connected");
        get_client_count = (rtk_get_client_count_func)GET_FUNCTION(dll_handle, "rtk_get_client_count");
        get_version = (rtk_get_version_func)GET_FUNCTION(dll_handle, "rtk_get_version");
        get_last_error = (rtk_get_last_error_func)GET_FUNCTION(dll_handle, "rtk_get_last_error");

        // Check if all functions were loaded
        if (!create_client || !destroy_client || !configure_mqtt || !connect) {
            std::cerr << "Failed to load DLL functions" << std::endl;
            CLOSE_LIBRARY(dll_handle);
            return false;
        }

        return true;
    }

public:
    RTKMQTTClient() : dll_handle(nullptr), client_handle(0) {}

    ~RTKMQTTClient() {
        if (client_handle) {
            Disconnect();
            destroy_client(client_handle);
        }
        if (dll_handle) {
            CLOSE_LIBRARY(dll_handle);
        }
    }

    bool Initialize(const std::string& dll_path) {
        if (!LoadDLL(dll_path)) {
            return false;
        }

        client_handle = create_client();
        if (client_handle == 0) {
            std::cerr << "Failed to create RTK MQTT client" << std::endl;
            return false;
        }

        std::cout << "RTK MQTT Framework Version: " << get_version() << std::endl;
        return true;
    }

    bool ConfigureMQTT(const std::string& broker_host, int broker_port, 
                       const std::string& client_id) {
        rtk_mqtt_config_t config = {0};
        strncpy(config.broker_host, broker_host.c_str(), sizeof(config.broker_host) - 1);
        config.broker_port = broker_port;
        strncpy(config.client_id, client_id.c_str(), sizeof(config.client_id) - 1);

        int result = configure_mqtt(client_handle, &config);
        if (result != RTK_SUCCESS) {
            std::cerr << "Failed to configure MQTT: " << result << std::endl;
            return false;
        }
        return true;
    }

    bool SetDeviceInfo(const std::string& id, const std::string& type,
                      const std::string& name, const std::string& version) {
        rtk_device_info_t info = {0};
        strncpy(info.id, id.c_str(), sizeof(info.id) - 1);
        strncpy(info.device_type, type.c_str(), sizeof(info.device_type) - 1);
        strncpy(info.name, name.c_str(), sizeof(info.name) - 1);
        strncpy(info.version, version.c_str(), sizeof(info.version) - 1);

        int result = set_device_info(client_handle, &info);
        if (result != RTK_SUCCESS) {
            std::cerr << "Failed to set device info: " << result << std::endl;
            return false;
        }
        return true;
    }

    bool Connect() {
        int result = connect(client_handle);
        if (result != RTK_SUCCESS) {
            std::cerr << "Failed to connect: " << result << std::endl;
            return false;
        }
        std::cout << "Connected to MQTT broker successfully" << std::endl;
        return true;
    }

    bool Disconnect() {
        int result = disconnect(client_handle);
        if (result != RTK_SUCCESS) {
            std::cerr << "Failed to disconnect: " << result << std::endl;
            return false;
        }
        std::cout << "Disconnected from MQTT broker" << std::endl;
        return true;
    }

    bool PublishState(const std::string& status, const std::string& health,
                     int64_t uptime) {
        rtk_device_state_t state = {0};
        strncpy(state.status, status.c_str(), sizeof(state.status) - 1);
        strncpy(state.health, health.c_str(), sizeof(state.health) - 1);
        state.uptime = uptime;
        state.last_seen = time(nullptr);

        int result = publish_state(client_handle, &state);
        if (result != RTK_SUCCESS) {
            std::cerr << "Failed to publish state: " << result << std::endl;
            return false;
        }
        std::cout << "Published device state successfully" << std::endl;
        return true;
    }
};

int main() {
    std::cout << "=== RTK MQTT Framework Windows C++ DLL Demo ===" << std::endl;

    // Initialize RTK MQTT client
    RTKMQTTClient client;
    
    // Load the DLL (adjust path as needed)
#ifdef _WIN32
    std::string dll_path = "rtk_mqtt_framework.dll";
#else
    std::string dll_path = "./rtk_mqtt_framework_simple.so";
#endif

    if (!client.Initialize(dll_path)) {
        std::cerr << "Failed to initialize RTK MQTT client" << std::endl;
        return -1;
    }

    // Configure MQTT connection
    if (!client.ConfigureMQTT("test.mosquitto.org", 1883, "rtk_cpp_demo_client")) {
        std::cerr << "Failed to configure MQTT" << std::endl;
        return -1;
    }

    // Set device information
    if (!client.SetDeviceInfo("00:11:22:33:44:55", "cpp_demo_device", 
                             "C++ Demo Device", "1.0.0")) {
        std::cerr << "Failed to set device info" << std::endl;
        return -1;
    }

    // Connect to MQTT broker
    if (!client.Connect()) {
        std::cerr << "Failed to connect to MQTT broker" << std::endl;
        return -1;
    }

    // Simulate device operation
    std::cout << "\n=== Simulating Device Operation ===" << std::endl;

    // Publish initial state
    client.PublishState("online", "healthy", 0);

    // Simulate device operation
    for (int i = 0; i < 5; i++) {
        std::this_thread::sleep_for(std::chrono::seconds(2));

        // Update state
        client.PublishState("online", "healthy", (i + 1) * 2);
        
        std::cout << "Cycle " << (i + 1) << " completed" << std::endl;
    }

    std::cout << "\n=== Demo Completed ===" << std::endl;

    // Disconnect and cleanup
    client.Disconnect();

    std::cout << "C++ DLL demo finished successfully!" << std::endl;
    return 0;
}