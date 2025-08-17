#!/bin/bash

# Comment out unimplemented storage methods
sed -i '' 's/if err := dim.storage.DeleteDeviceIdentity/\/\/ TODO: Implement DeleteDeviceIdentity\n\t\/\/ if err := dim.storage.DeleteDeviceIdentity/g' device_identity_manager.go
sed -i '' 's/allIdentities, err := dim.storage.GetAllDeviceIdentities/\/\/ TODO: Implement GetAllDeviceIdentities\n\tallIdentities, err := \/\*dim.storage.\*\/GetAllDeviceIdentities/g' device_identity_manager.go
sed -i '' 's/if err := dim.storage.SetDeviceGroup/\/\/ TODO: Implement SetDeviceGroup\n\t\/\/ if err := dim.storage.SetDeviceGroup/g' device_identity_manager.go
sed -i '' 's/group, err := dim.storage.GetDeviceGroup/\/\/ TODO: Implement GetDeviceGroup\n\tvar group \*DeviceGroup\n\t_ = \/\*dim.storage.GetDeviceGroup/g' device_identity_manager.go
sed -i '' 's/groups, err := dim.storage.GetAllDeviceGroups/\/\/ TODO: Implement GetAllDeviceGroups\n\tvar groups []\*DeviceGroup\n\t_ = \/\*dim.storage.GetAllDeviceGroups/g' device_identity_manager.go
sed -i '' 's/if err := dim.storage.DeleteDeviceGroup/\/\/ TODO: Implement DeleteDeviceGroup\n\t\/\/ if err := dim.storage.DeleteDeviceGroup/g' device_identity_manager.go
sed -i '' 's/if err := dim.storage.SetDeviceTag/\/\/ TODO: Implement SetDeviceTag\n\t\/\/ if err := dim.storage.SetDeviceTag/g' device_identity_manager.go
sed -i '' 's/tag, err := dim.storage.GetDeviceTag/\/\/ TODO: Implement GetDeviceTag\n\tvar tag \*DeviceTag\n\t_ = \/\*dim.storage.GetDeviceTag/g' device_identity_manager.go
sed -i '' 's/tags, err := dim.storage.GetAllDeviceTags/\/\/ TODO: Implement GetAllDeviceTags\n\tvar tags []\*DeviceTag\n\t_ = \/\*dim.storage.GetAllDeviceTags/g' device_identity_manager.go
sed -i '' 's/if err := dim.storage.DeleteDeviceTag/\/\/ TODO: Implement DeleteDeviceTag\n\t\/\/ if err := dim.storage.DeleteDeviceTag/g' device_identity_manager.go
sed -i '' 's/identities, err := dim.storage.GetAllDeviceIdentities/\/\/ TODO: Implement GetAllDeviceIdentities\n\tidentities, err := \/\*dim.storage.\*\/GetAllDeviceIdentities/g' device_identity_manager.go
