package auth

import (
	"rtk_mqtt_broker/config"
)

type AuthManager struct {
	users map[string]string
}

func NewAuthManager(config *config.Config) *AuthManager {
	users := make(map[string]string)
	
	for _, user := range config.Security.Users {
		if username, ok := user["username"]; ok {
			if password, ok := user["password"]; ok {
				users[username] = password
			}
		}
	}

	return &AuthManager{
		users: users,
	}
}

func (a *AuthManager) Authenticate(username, password string) bool {
	if storedPassword, exists := a.users[username]; exists {
		return storedPassword == password
	}
	return false
}

func (a *AuthManager) IsEnabled() bool {
	return len(a.users) > 0
}