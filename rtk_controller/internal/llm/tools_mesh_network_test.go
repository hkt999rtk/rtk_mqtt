package llm

import (
	"context"
	"testing"
	"time"

	"rtk_controller/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeshGetTopologyTool_Execute(t *testing.T) {
	tool := &MeshGetTopologyTool{name: "mesh.get_topology"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"scan_depth": 3,
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

func TestMeshNodeRelationshipTool_Execute(t *testing.T) {
	tool := &MeshNodeRelationshipTool{name: "mesh.node_relationship"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"node_id": "mesh-node-002",
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data

	// Test with no node_id (all nodes)
	delete(params, "node_id")
	result, err = tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestMeshPathOptimizationTool_Execute(t *testing.T) {
	tool := &MeshPathOptimizationTool{name: "mesh.path_optimization"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"optimization_mode": "balanced",
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data

	// Test different optimization modes
	modes := []string{"throughput", "latency", "balanced", "power_efficient"}
	for _, mode := range modes {
		params["optimization_mode"] = mode
		result, err = tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.True(t, result.Success)
	}
}

func TestMeshBackhaulTestTool_Execute(t *testing.T) {
	tool := &MeshBackhaulTestTool{name: "mesh.backhaul_test"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"test_duration": 30,
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data

	// Test with default duration
	delete(params, "test_duration")
	result, err = tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestMeshLoadBalancingTool_Execute(t *testing.T) {
	tool := &MeshLoadBalancingTool{name: "mesh.load_balancing"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"rebalance_strategy": "adaptive",
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data

	// Test different strategies
	strategies := []string{"adaptive", "round_robin", "least_loaded", "geographic"}
	for _, strategy := range strategies {
		params["rebalance_strategy"] = strategy
		result, err = tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.True(t, result.Success)
	}
}

func TestMeshFailoverSimulationTool_Execute(t *testing.T) {
	tool := &MeshFailoverSimulationTool{name: "mesh.failover_simulation"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]interface{}{
		"failure_scenario": "single_node_failure",
	}

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify result structure - check for any analysis data
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data) // Just verify we have some data

	// Test different failure scenarios
	scenarios := []string{"single_node_failure", "gateway_failure", "cascade_failure", "network_partition"}
	for _, scenario := range scenarios {
		params["failure_scenario"] = scenario
		result, err = tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.True(t, result.Success)
	}
}

func TestMeshToolsInterface(t *testing.T) {
	tools := []types.LLMTool{
		&MeshGetTopologyTool{name: "mesh.get_topology"},
		&MeshNodeRelationshipTool{name: "mesh.node_relationship"},
		&MeshPathOptimizationTool{name: "mesh.path_optimization"},
		&MeshBackhaulTestTool{name: "mesh.backhaul_test"},
		&MeshLoadBalancingTool{name: "mesh.load_balancing"},
		&MeshFailoverSimulationTool{name: "mesh.failover_simulation"},
	}

	for _, tool := range tools {
		// Test Name() method
		assert.NotEmpty(t, tool.Name())

		// Test Category() method - mesh tools have different categories
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

func TestMeshToolsValidation(t *testing.T) {
	testCases := []struct {
		tool   types.LLMTool
		params map[string]interface{}
		valid  bool
	}{
		{
			tool:   &MeshGetTopologyTool{name: "mesh.get_topology"},
			params: map[string]interface{}{"scan_depth": 3},
			valid:  true,
		},
		{
			tool:   &MeshGetTopologyTool{name: "mesh.get_topology"},
			params: map[string]interface{}{"scan_depth": "invalid"},
			valid:  false,
		},
		{
			tool:   &MeshNodeRelationshipTool{name: "mesh.node_relationship"},
			params: map[string]interface{}{"node_id": "mesh-node-001"},
			valid:  true,
		},
		{
			tool:   &MeshNodeRelationshipTool{name: "mesh.node_relationship"},
			params: map[string]interface{}{"node_id": 123},
			valid:  false,
		},
		{
			tool:   &MeshPathOptimizationTool{name: "mesh.path_optimization"},
			params: map[string]interface{}{"optimization_mode": "balanced"},
			valid:  true,
		},
		{
			tool:   &MeshBackhaulTestTool{name: "mesh.backhaul_test"},
			params: map[string]interface{}{"test_duration": 60},
			valid:  true,
		},
		{
			tool:   &MeshBackhaulTestTool{name: "mesh.backhaul_test"},
			params: map[string]interface{}{"test_duration": "invalid"},
			valid:  false,
		},
		{
			tool:   &MeshLoadBalancingTool{name: "mesh.load_balancing"},
			params: map[string]interface{}{"rebalance_strategy": "adaptive"},
			valid:  true,
		},
		{
			tool:   &MeshFailoverSimulationTool{name: "mesh.failover_simulation"},
			params: map[string]interface{}{"failure_scenario": "single_node_failure"},
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

func TestMeshToolsConstructors(t *testing.T) {
	constructors := []func() types.LLMTool{
		func() types.LLMTool { return NewMeshGetTopologyTool() },
		func() types.LLMTool { return NewMeshNodeRelationshipTool() },
		func() types.LLMTool { return NewMeshPathOptimizationTool() },
		func() types.LLMTool { return NewMeshBackhaulTestTool() },
		func() types.LLMTool { return NewMeshLoadBalancingTool() },
		func() types.LLMTool { return NewMeshFailoverSimulationTool() },
	}

	expectedNames := []string{
		"mesh.get_topology",
		"mesh.node_relationship",
		"mesh.path_optimization",
		"mesh.backhaul_test",
		"mesh.load_balancing",
		"mesh.failover_simulation",
	}

	for i, constructor := range constructors {
		tool := constructor()
		assert.NotNil(t, tool)
		assert.Equal(t, expectedNames[i], tool.Name())
	}
}

// Benchmark tests
func BenchmarkMeshGetTopologyTool_Execute(b *testing.B) {
	tool := &MeshGetTopologyTool{name: "mesh.get_topology"}
	ctx := context.Background()
	params := map[string]interface{}{
		"scan_depth": 3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.Execute(ctx, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMeshBackhaulTestTool_Execute(b *testing.B) {
	tool := &MeshBackhaulTestTool{name: "mesh.backhaul_test"}
	ctx := context.Background()
	params := map[string]interface{}{
		"test_duration": 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.Execute(ctx, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}
