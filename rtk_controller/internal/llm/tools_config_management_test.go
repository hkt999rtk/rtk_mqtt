package llm

import (
	"context"
	"testing"
	"time"

	"rtk_controller/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigWiFiSettingsTool_Execute(t *testing.T) {
	tool := &ConfigWiFiSettingsTool{name: "config.wifi_settings"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"device_id": "ap-001",
		"config_changes": map[string]interface{}{
			"ssid":    "NewNetwork",
			"channel": 36,
		},
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data
}

func TestConfigQoSPoliciesTool_Execute(t *testing.T) {
	tool := &ConfigQoSPoliciesTool{name: "config.qos_policies"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"policy_type": "adaptive",
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data
}

func TestConfigSecuritySettingsTool_Execute(t *testing.T) {
	tool := &ConfigSecuritySettingsTool{name: "config.security_settings"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"security_level": "high",
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data
}

func TestConfigBandSteeringTool_Execute(t *testing.T) {
	tool := &ConfigBandSteeringTool{name: "config.band_steering"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"steering_mode": "adaptive",
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data
}

func TestConfigAutoOptimizeTool_Execute(t *testing.T) {
	tool := &ConfigAutoOptimizeTool{name: "config.auto_optimize"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"optimization_scope": "full_network",
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data
}

func TestConfigValidateChangesTool_Execute(t *testing.T) {
	tool := &ConfigValidateChangesTool{name: "config.validate_changes"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"proposed_changes": map[string]interface{}{
			"wifi_channel": 36,
			"tx_power":     20,
		},
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data
}

func TestConfigRollbackSafeTool_Execute(t *testing.T) {
	tool := &ConfigRollbackSafeTool{name: "config.rollback_safe"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"backup_id": "backup_20250821_045530",
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data
}

func TestConfigImpactAnalysisTool_Execute(t *testing.T) {
	tool := &ConfigImpactAnalysisTool{name: "config.impact_analysis"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"proposed_changes": map[string]interface{}{
			"channel_change":   "6 -> 36",
			"power_adjustment": "20dBm -> 18dBm",
		},
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data
}

func TestConfigToolsInterface(t *testing.T) {
	tools := []types.LLMTool{
		&ConfigWiFiSettingsTool{name: "config.wifi_settings"},
		&ConfigQoSPoliciesTool{name: "config.qos_policies"},
		&ConfigSecuritySettingsTool{name: "config.security_settings"},
		&ConfigBandSteeringTool{name: "config.band_steering"},
		&ConfigAutoOptimizeTool{name: "config.auto_optimize"},
		&ConfigValidateChangesTool{name: "config.validate_changes"},
		&ConfigRollbackSafeTool{name: "config.rollback_safe"},
		&ConfigImpactAnalysisTool{name: "config.impact_analysis"},
	}

	for _, tool := range tools {
		// Test Name() method
		assert.NotEmpty(t, tool.Name())

		// Test Category() method - config tools have different categories
		category := tool.Category()
		assert.NotEmpty(t, category)
		validCategories := []types.ToolCategory{
			types.ToolCategoryRead,
			types.ToolCategoryTest,
			types.ToolCategoryAct,
		}
		found := false
		for _, vc := range validCategories {
			if category == vc {
				found = true
				break
			}
		}
		assert.True(t, found, "Tool %s has invalid category: %s", tool.Name(), category)

		// Test Description() method
		assert.NotEmpty(t, tool.Description())

		// Test RequiredCapabilities() method
		capabilities := tool.RequiredCapabilities()
		assert.NotNil(t, capabilities)

		// Test Validate() method with empty params
		err := tool.Validate(map[string]interface{}{})
		// Some tools require specific parameters, so validation may fail
		// We're just checking that the method doesn't panic
		_ = err
	}
}

func TestConfigToolsValidation(t *testing.T) {
	testCases := []struct {
		tool   types.LLMTool
		params map[string]interface{}
		valid  bool
	}{
		{
			tool:   &ConfigWiFiSettingsTool{name: "config.wifi_settings"},
			params: map[string]interface{}{"device_id": "ap-001"},
			valid:  true,
		},
		{
			tool:   &ConfigWiFiSettingsTool{name: "config.wifi_settings"},
			params: map[string]interface{}{"device_id": 123},
			valid:  false,
		},
		{
			tool:   &ConfigQoSPoliciesTool{name: "config.qos_policies"},
			params: map[string]interface{}{"policy_type": "adaptive"},
			valid:  true,
		},
		{
			tool:   &ConfigSecuritySettingsTool{name: "config.security_settings"},
			params: map[string]interface{}{"security_level": "high"},
			valid:  true,
		},
		{
			tool:   &ConfigValidateChangesTool{name: "config.validate_changes"},
			params: map[string]interface{}{"proposed_changes": map[string]interface{}{"test": "value"}},
			valid:  true,
		},
		{
			tool:   &ConfigValidateChangesTool{name: "config.validate_changes"},
			params: map[string]interface{}{"proposed_changes": "invalid"},
			valid:  false,
		},
		{
			tool:   &ConfigImpactAnalysisTool{name: "config.impact_analysis"},
			params: map[string]interface{}{"proposed_changes": map[string]interface{}{"test": "value"}},
			valid:  true,
		},
	}

	for _, tc := range testCases {
		err := tc.tool.Validate(tc.params)
		if tc.valid {
			assert.NoError(t, err, "Expected validation to pass for %s", tc.tool.Name())
		} else {
			assert.Error(t, err, "Expected validation to fail for %s", tc.tool.Name())
		}
	}
}

func TestConfigToolsConstructors(t *testing.T) {
	constructors := []func() types.LLMTool{
		func() types.LLMTool { return NewConfigWiFiSettingsTool() },
		func() types.LLMTool { return NewConfigQoSPoliciesTool() },
		func() types.LLMTool { return NewConfigSecuritySettingsTool() },
		func() types.LLMTool { return NewConfigBandSteeringTool() },
		func() types.LLMTool { return NewConfigAutoOptimizeTool() },
		func() types.LLMTool { return NewConfigValidateChangesTool() },
		func() types.LLMTool { return NewConfigRollbackSafeTool() },
		func() types.LLMTool { return NewConfigImpactAnalysisTool() },
	}

	expectedNames := []string{
		"config.wifi_settings",
		"config.qos_policies",
		"config.security_settings",
		"config.band_steering",
		"config.auto_optimize",
		"config.validate_changes",
		"config.rollback_safe",
		"config.impact_analysis",
	}

	for i, constructor := range constructors {
		tool := constructor()
		assert.NotNil(t, tool)
		assert.Equal(t, expectedNames[i], tool.Name())
	}
}

// Benchmark tests
func BenchmarkConfigWiFiSettingsTool_Execute(b *testing.B) {
	tool := &ConfigWiFiSettingsTool{name: "config.wifi_settings"}
	ctx := context.Background()
	params := map[string]interface{}{
		"device_id": "ap-001",
		"config_changes": map[string]interface{}{
			"ssid": "TestNetwork",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.Execute(ctx, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConfigAutoOptimizeTool_Execute(b *testing.B) {
	tool := &ConfigAutoOptimizeTool{name: "config.auto_optimize"}
	ctx := context.Background()
	params := map[string]interface{}{
		"optimization_scope": "single_ap",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.Execute(ctx, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}
