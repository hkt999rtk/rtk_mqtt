package devices

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"rtk_simulation/pkg/config"
	"rtk_simulation/pkg/devices/base"
	"rtk_simulation/pkg/devices/client"
	"rtk_simulation/pkg/devices/iot"
	"rtk_simulation/pkg/devices/network"
)

type DeviceManager struct {
	devices     map[string]base.Device
	deviceTypes map[string]DeviceFactory
	mu          sync.RWMutex
	logger      *logrus.Logger
}

type DeviceFactory func(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error)

type DeviceRegistration struct {
	DeviceType string
	Factory    DeviceFactory
}

func NewDeviceManager(logger *logrus.Logger) *DeviceManager {
	dm := &DeviceManager{
		devices:     make(map[string]base.Device),
		deviceTypes: make(map[string]DeviceFactory),
		logger:      logger,
	}

	dm.registerBuiltInDevices()
	return dm
}

func (dm *DeviceManager) registerBuiltInDevices() {
	registrations := []DeviceRegistration{
		{"router", createRouter},
		{"switch", createSwitch},
		{"access_point", createAccessPoint},
		{"mesh_node", createMeshNode},
		{"smart_bulb", createSmartBulb},
		{"air_conditioner", createAirConditioner},
		{"environmental_sensor", createEnvironmentalSensor},
		{"security_camera", createSecurityCamera},
		{"smart_thermostat", createSmartThermostat},
		{"smart_plug", createSmartPlug},
		{"smartphone", createSmartphone},
		{"laptop", createLaptop},
		{"tablet", createTablet},
		{"smart_tv", createSmartTV},
	}

	for _, reg := range registrations {
		dm.RegisterDeviceType(reg.DeviceType, reg.Factory)
	}
}

func (dm *DeviceManager) RegisterDeviceType(deviceType string, factory DeviceFactory) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.deviceTypes[deviceType] = factory
	dm.logger.Infof("Registered device type: %s", deviceType)
}

func (dm *DeviceManager) CreateDevice(deviceConfig config.DeviceConfig) (base.Device, error) {
	dm.mu.RLock()
	factory, exists := dm.deviceTypes[deviceConfig.Type]
	dm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown device type: %s", deviceConfig.Type)
	}

	device, err := factory(deviceConfig, dm.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create device %s (%s): %v", deviceConfig.ID, deviceConfig.Type, err)
	}

	dm.mu.Lock()
	dm.devices[deviceConfig.ID] = device
	dm.mu.Unlock()

	dm.logger.Infof("Created device: %s (%s)", deviceConfig.ID, deviceConfig.Type)
	return device, nil
}

func (dm *DeviceManager) GetDevice(deviceID string) (base.Device, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	device, exists := dm.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}
	return device, nil
}

func (dm *DeviceManager) ListDevices() []base.Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	devices := make([]base.Device, 0, len(dm.devices))
	for _, device := range dm.devices {
		devices = append(devices, device)
	}
	return devices
}

func (dm *DeviceManager) ListDevicesByType(deviceType string) []base.Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var devices []base.Device
	for _, device := range dm.devices {
		if device.GetDeviceType() == deviceType {
			devices = append(devices, device)
		}
	}
	return devices
}

func (dm *DeviceManager) RemoveDevice(deviceID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	device, exists := dm.devices[deviceID]
	if !exists {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	if err := device.Stop(); err != nil {
		dm.logger.Errorf("Failed to stop device %s: %v", deviceID, err)
	}

	delete(dm.devices, deviceID)
	dm.logger.Infof("Removed device: %s", deviceID)
	return nil
}

func (dm *DeviceManager) StartAllDevices(ctx context.Context) error {
	dm.mu.RLock()
	devices := make([]base.Device, 0, len(dm.devices))
	for _, device := range dm.devices {
		devices = append(devices, device)
	}
	dm.mu.RUnlock()

	for _, device := range devices {
		if err := device.Start(ctx); err != nil {
			dm.logger.Errorf("Failed to start device %s: %v", device.GetDeviceID(), err)
			return err
		}
	}

	dm.logger.Infof("Started %d devices", len(devices))
	return nil
}

func (dm *DeviceManager) StopAllDevices() error {
	dm.mu.RLock()
	devices := make([]base.Device, 0, len(dm.devices))
	for _, device := range dm.devices {
		devices = append(devices, device)
	}
	dm.mu.RUnlock()

	var lastErr error
	for _, device := range devices {
		if err := device.Stop(); err != nil {
			dm.logger.Errorf("Failed to stop device %s: %v", device.GetDeviceID(), err)
			lastErr = err
		}
	}

	dm.logger.Infof("Stopped %d devices", len(devices))
	return lastErr
}

func (dm *DeviceManager) GetDeviceStats() map[string]interface{} {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_devices"] = len(dm.devices)

	typeCounts := make(map[string]int)
	for _, device := range dm.devices {
		typeCounts[device.GetDeviceType()]++
	}
	stats["device_types"] = typeCounts

	return stats
}

// 輔助函數：轉換配置
func convertToBaseConfig(config config.DeviceConfig) base.DeviceConfig {
	return base.DeviceConfig{
		ID:             config.ID,
		Type:           config.Type,
		IPAddress:      config.IPAddress,
		Tenant:         config.Tenant,
		Site:           config.Site,
		ConnectionType: config.ConnectionType,
		Firmware:       config.Firmware,
		Protocols:      config.Protocols,
		Extra:          make(map[string]interface{}), // Initialize empty Extra map
	}
}

func createDefaultMQTTConfig(deviceID string) base.MQTTConfig {
	return base.MQTTConfig{
		Broker:   "localhost",
		Port:     1883,
		Username: "",
		Password: "",
		ClientID: deviceID,
	}
}

func createRouter(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return network.NewRouter(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createSwitch(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return network.NewSwitch(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createAccessPoint(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return network.NewAccessPoint(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createSmartBulb(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return iot.NewSmartBulb(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createAirConditioner(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return iot.NewAirConditioner(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createEnvironmentalSensor(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return iot.NewEnvironmentalSensor(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createSecurityCamera(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return iot.NewSecurityCamera(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createSmartThermostat(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return iot.NewSmartThermostat(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createSmartPlug(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return iot.NewSmartPlug(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createSmartphone(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return client.NewSmartphone(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createLaptop(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return client.NewLaptop(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createTablet(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return client.NewTablet(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createSmartTV(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return client.NewSmartTV(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}

func createMeshNode(config config.DeviceConfig, logger *logrus.Logger) (base.Device, error) {
	return network.NewMeshNode(convertToBaseConfig(config), createDefaultMQTTConfig(config.ID))
}
