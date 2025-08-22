package device

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// MockStorage implements storage interface for testing
type MockStorage struct {
	mock.Mock
	data map[string]interface{}
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string]interface{}),
	}
}

func (m *MockStorage) Set(key string, value interface{}) error {
	args := m.Called(key, value)
	if args.Error(0) == nil {
		m.data[key] = value
	}
	return args.Error(0)
}

func (m *MockStorage) Get(key string, result interface{}) error {
	args := m.Called(key, result)
	return args.Error(0)
}

func (m *MockStorage) Delete(key string) error {
	args := m.Called(key)
	if args.Error(0) == nil {
		delete(m.data, key)
	}
	return args.Error(0)
}

func (m *MockStorage) List(pattern string) ([]string, error) {
	args := m.Called(pattern)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorage) CreateIndex(name, pattern string, less func(a, b string) bool) error {
	args := m.Called(name, pattern, less)
	return args.Error(0)
}

func (m *MockStorage) View(fn func(storage.Transaction) error) error {
	args := m.Called(fn)
	return args.Error(0)
}

func (m *MockStorage) Update(fn func(storage.Transaction) error) error {
	args := m.Called(fn)
	return args.Error(0)
}

func TestNewManager(t *testing.T) {
	mockStorage := NewMockStorage()

	manager := NewManager(mockStorage)

	assert.NotNil(t, manager)
	assert.Equal(t, mockStorage, manager.storage)
	assert.NotNil(t, manager.devices)
	assert.NotNil(t, manager.stats)
}

func TestManager_Start(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	// Mock the List call for loading existing devices
	mockStorage.On("List", "devices:*").Return([]string{}, nil)

	ctx := context.Background()
	err := manager.Start(ctx)

	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestManager_RegisterDevice(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	deviceInfo := types.ExtendedDeviceInfo{
		DeviceInfo: types.DeviceInfo{
			Tenant:   "test-tenant",
			Site:     "test-site",
			DeviceID: "test-device",
			Type:     "wifi_router",
		},
		Location: "Test Location",
		Model:    "Test Model",
		Firmware: "1.0.0",
	}

	// Mock storage operations
	expectedKey := "devices:test-tenant:test-site:test-device"
	mockStorage.On("Set", expectedKey, mock.AnythingOfType("types.DeviceState")).Return(nil)

	err := manager.RegisterDevice(deviceInfo)

	assert.NoError(t, err)

	// Verify device was added to in-memory store
	key := "test-tenant:test-site:test-device"
	device, exists := manager.devices[key]
	assert.True(t, exists)
	assert.Equal(t, deviceInfo.DeviceInfo, device.DeviceInfo)
	assert.Equal(t, types.DeviceStatusOnline, device.Status)

	mockStorage.AssertExpectations(t)
}

func TestManager_UpdateDeviceState(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	// Register a device first
	deviceInfo := types.ExtendedDeviceInfo{
		DeviceInfo: types.DeviceInfo{
			Tenant:   "test-tenant",
			Site:     "test-site",
			DeviceID: "test-device",
			Type:     "wifi_router",
		},
	}

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.DeviceState")).Return(nil)
	err := manager.RegisterDevice(deviceInfo)
	require.NoError(t, err)

	// Update device state
	stateData := map[string]interface{}{
		"cpu_usage":    75.5,
		"memory_usage": 60.2,
		"uptime":       3600,
	}

	err = manager.UpdateDeviceState(deviceInfo.DeviceInfo, stateData)

	assert.NoError(t, err)

	// Verify state was updated
	key := "test-tenant:test-site:test-device"
	device, exists := manager.devices[key]
	assert.True(t, exists)
	assert.Equal(t, stateData, device.State)
	assert.WithinDuration(t, time.Now(), device.LastSeen, time.Second)

	mockStorage.AssertExpectations(t)
}

func TestManager_GetDevice(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	deviceInfo := types.ExtendedDeviceInfo{
		DeviceInfo: types.DeviceInfo{
			Tenant:   "test-tenant",
			Site:     "test-site",
			DeviceID: "test-device",
			Type:     "wifi_router",
		},
	}

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.DeviceState")).Return(nil)
	err := manager.RegisterDevice(deviceInfo)
	require.NoError(t, err)

	// Test getting existing device
	device, err := manager.GetDevice(deviceInfo.DeviceInfo)
	assert.NoError(t, err)
	assert.NotNil(t, device)
	assert.Equal(t, deviceInfo.DeviceInfo, device.DeviceInfo)

	// Test getting non-existent device
	nonExistentDevice := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "non-existent",
		Type:     "wifi_router",
	}

	device, err = manager.GetDevice(nonExistentDevice)
	assert.Error(t, err)
	assert.Nil(t, device)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_ListDevices(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	// Register multiple devices
	devices := []types.ExtendedDeviceInfo{
		{
			DeviceInfo: types.DeviceInfo{
				Tenant:   "tenant1",
				Site:     "site1",
				DeviceID: "device1",
				Type:     "wifi_router",
			},
		},
		{
			DeviceInfo: types.DeviceInfo{
				Tenant:   "tenant1",
				Site:     "site1",
				DeviceID: "device2",
				Type:     "smart_switch",
			},
		},
		{
			DeviceInfo: types.DeviceInfo{
				Tenant:   "tenant2",
				Site:     "site1",
				DeviceID: "device3",
				Type:     "iot_sensor",
			},
		},
	}

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.DeviceState")).Return(nil)
	for _, device := range devices {
		err := manager.RegisterDevice(device)
		require.NoError(t, err)
	}

	// Test listing all devices
	allDevices := manager.ListDevices("")
	assert.Len(t, allDevices, 3)

	// Test filtering by tenant
	tenant1Devices := manager.ListDevices("tenant1")
	assert.Len(t, tenant1Devices, 2)
	for _, device := range tenant1Devices {
		assert.Equal(t, "tenant1", device.Tenant)
	}

	// Test filtering by non-existent tenant
	nonExistentDevices := manager.ListDevices("non-existent")
	assert.Len(t, nonExistentDevices, 0)
}

func TestManager_DeleteDevice(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	deviceInfo := types.ExtendedDeviceInfo{
		DeviceInfo: types.DeviceInfo{
			Tenant:   "test-tenant",
			Site:     "test-site",
			DeviceID: "test-device",
			Type:     "wifi_router",
		},
	}

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.DeviceState")).Return(nil)
	err := manager.RegisterDevice(deviceInfo)
	require.NoError(t, err)

	// Verify device exists
	_, err = manager.GetDevice(deviceInfo.DeviceInfo)
	assert.NoError(t, err)

	// Mock delete operation
	expectedKey := "devices:test-tenant:test-site:test-device"
	mockStorage.On("Delete", expectedKey).Return(nil)

	// Delete device
	err = manager.DeleteDevice(deviceInfo.DeviceInfo)
	assert.NoError(t, err)

	// Verify device is gone
	_, err = manager.GetDevice(deviceInfo.DeviceInfo)
	assert.Error(t, err)

	mockStorage.AssertExpectations(t)
}

func TestManager_SetDeviceStatus(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	deviceInfo := types.ExtendedDeviceInfo{
		DeviceInfo: types.DeviceInfo{
			Tenant:   "test-tenant",
			Site:     "test-site",
			DeviceID: "test-device",
			Type:     "wifi_router",
		},
	}

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.DeviceState")).Return(nil)
	err := manager.RegisterDevice(deviceInfo)
	require.NoError(t, err)

	tests := []struct {
		name   string
		status types.DeviceStatus
	}{
		{
			name:   "set online",
			status: types.DeviceStatusOnline,
		},
		{
			name:   "set offline",
			status: types.DeviceStatusOffline,
		},
		{
			name:   "set error",
			status: types.DeviceStatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SetDeviceStatus(deviceInfo.DeviceInfo, tt.status)
			assert.NoError(t, err)

			device, err := manager.GetDevice(deviceInfo.DeviceInfo)
			assert.NoError(t, err)
			assert.Equal(t, tt.status, device.Status)
		})
	}
}

func TestManager_GetStats(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	// Initially, stats should be zero
	stats := manager.GetStats()
	assert.Equal(t, int64(0), stats.TotalDevices)
	assert.Equal(t, int64(0), stats.OnlineDevices)
	assert.Equal(t, int64(0), stats.OfflineDevices)

	// Register devices with different statuses
	devices := []struct {
		info   types.ExtendedDeviceInfo
		status types.DeviceStatus
	}{
		{
			info: types.ExtendedDeviceInfo{
				DeviceInfo: types.DeviceInfo{
					Tenant:   "tenant1",
					Site:     "site1",
					DeviceID: "device1",
					Type:     "wifi_router",
				},
			},
			status: types.DeviceStatusOnline,
		},
		{
			info: types.ExtendedDeviceInfo{
				DeviceInfo: types.DeviceInfo{
					Tenant:   "tenant1",
					Site:     "site1",
					DeviceID: "device2",
					Type:     "smart_switch",
				},
			},
			status: types.DeviceStatusOffline,
		},
		{
			info: types.ExtendedDeviceInfo{
				DeviceInfo: types.DeviceInfo{
					Tenant:   "tenant1",
					Site:     "site1",
					DeviceID: "device3",
					Type:     "iot_sensor",
				},
			},
			status: types.DeviceStatusOnline,
		},
	}

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.DeviceState")).Return(nil)
	for _, device := range devices {
		err := manager.RegisterDevice(device.info)
		require.NoError(t, err)

		err = manager.SetDeviceStatus(device.info.DeviceInfo, device.status)
		require.NoError(t, err)
	}

	// Check updated stats
	stats = manager.GetStats()
	assert.Equal(t, int64(3), stats.TotalDevices)
	assert.Equal(t, int64(2), stats.OnlineDevices)
	assert.Equal(t, int64(1), stats.OfflineDevices)
}

func TestManager_Stop(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	// Start the manager
	mockStorage.On("List", "devices:*").Return([]string{}, nil)
	ctx := context.Background()
	err := manager.Start(ctx)
	require.NoError(t, err)

	// Stop the manager
	manager.Stop()

	// Verify that operations still work after stop (manager should be graceful)
	deviceInfo := types.ExtendedDeviceInfo{
		DeviceInfo: types.DeviceInfo{
			Tenant:   "test-tenant",
			Site:     "test-site",
			DeviceID: "test-device",
			Type:     "wifi_router",
		},
	}

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.DeviceState")).Return(nil)
	err = manager.RegisterDevice(deviceInfo)
	assert.NoError(t, err) // Should still work after stop
}

func TestManager_ConcurrentAccess(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	const numGoroutines = 10
	const numDevices = 10

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.DeviceState")).Return(nil)

	// Test concurrent device registration
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < numDevices; j++ {
				deviceInfo := types.ExtendedDeviceInfo{
					DeviceInfo: types.DeviceInfo{
						Tenant:   "tenant1",
						Site:     "site1",
						DeviceID: string(rune('A' + id*numDevices + j)), // Unique device IDs
						Type:     "wifi_router",
					},
				}

				err := manager.RegisterDevice(deviceInfo)
				assert.NoError(t, err)

				// Test concurrent status updates
				err = manager.SetDeviceStatus(deviceInfo.DeviceInfo, types.DeviceStatusOnline)
				assert.NoError(t, err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all devices were registered
	stats := manager.GetStats()
	assert.Equal(t, int64(numGoroutines*numDevices), stats.TotalDevices)
	assert.Equal(t, int64(numGoroutines*numDevices), stats.OnlineDevices)
}

func TestManager_DeviceTimeout(t *testing.T) {
	mockStorage := NewMockStorage()
	manager := NewManager(mockStorage)

	deviceInfo := types.ExtendedDeviceInfo{
		DeviceInfo: types.DeviceInfo{
			Tenant:   "test-tenant",
			Site:     "test-site",
			DeviceID: "test-device",
			Type:     "wifi_router",
		},
	}

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.DeviceState")).Return(nil)
	err := manager.RegisterDevice(deviceInfo)
	require.NoError(t, err)

	// Get device and verify it's online
	device, err := manager.GetDevice(deviceInfo.DeviceInfo)
	assert.NoError(t, err)
	assert.Equal(t, types.DeviceStatusOnline, device.Status)

	// Manually set last seen to old time to simulate timeout
	key := "test-tenant:test-site:test-device"
	manager.mu.Lock()
	if dev, exists := manager.devices[key]; exists {
		dev.LastSeen = time.Now().Add(-10 * time.Minute) // 10 minutes ago
		manager.devices[key] = dev
	}
	manager.mu.Unlock()

	// Check if device would be considered offline (this would be done by a background process)
	device, err = manager.GetDevice(deviceInfo.DeviceInfo)
	assert.NoError(t, err)

	// The device should still be marked as online until the timeout checker runs
	// This test verifies the structure is in place for timeout checking
	assert.True(t, device.LastSeen.Before(time.Now().Add(-5*time.Minute)))
}
