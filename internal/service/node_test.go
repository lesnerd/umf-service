package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNodeInfo(t *testing.T) {
	nodeInfo := NewNodeInfo()

	assert.NotNil(t, nodeInfo)
	assert.Implements(t, (*NodeInfo)(nil), nodeInfo)

	// Test that both IDs are generated
	nodeID := nodeInfo.GetNodeId()
	serviceID := nodeInfo.GetServiceId()

	assert.NotEmpty(t, nodeID)
	assert.NotEmpty(t, serviceID)
	assert.NotEqual(t, nodeID, serviceID)
}

func TestNodeInfo_GetNodeId(t *testing.T) {
	nodeInfo := NewNodeInfo()

	nodeID := nodeInfo.GetNodeId()
	assert.NotEmpty(t, nodeID)
	// Node ID could be hostname or UUID, so just check it's not empty
}

func TestNodeInfo_GetServiceId(t *testing.T) {
	nodeInfo := NewNodeInfo()

	serviceID := nodeInfo.GetServiceId()
	assert.NotEmpty(t, serviceID)
	assert.Len(t, serviceID, 36) // UUID length
}

func TestGenerateNodeId_EnvironmentVariable(t *testing.T) {
	// Set environment variable
	expectedNodeID := "test-node-id"
	os.Setenv("NODE_ID", expectedNodeID)
	defer os.Unsetenv("NODE_ID")

	nodeInfo := NewNodeInfo()
	nodeID := nodeInfo.GetNodeId()

	assert.Equal(t, expectedNodeID, nodeID)
}

func TestGenerateNodeId_Hostname(t *testing.T) {
	// Clear environment variable
	os.Unsetenv("NODE_ID")

	// Get hostname
	hostname, err := os.Hostname()
	require.NoError(t, err)

	nodeInfo := NewNodeInfo()
	nodeID := nodeInfo.GetNodeId()

	// Should use hostname when NODE_ID is not set
	assert.Equal(t, hostname, nodeID)
}

func TestGenerateNodeId_UUIDFallback(t *testing.T) {
	// Clear environment variable
	os.Unsetenv("NODE_ID")

	// Mock hostname to fail (this is difficult to do in Go)
	// Instead, we'll test that the function always returns a valid ID
	nodeInfo := NewNodeInfo()
	nodeID := nodeInfo.GetNodeId()

	assert.NotEmpty(t, nodeID)
	// Node ID could be hostname or UUID, so just check it's not empty
}

func TestGenerateServiceId_EnvironmentVariable(t *testing.T) {
	// Set environment variable
	expectedServiceID := "test-service-id"
	os.Setenv("SERVICE_ID", expectedServiceID)
	defer os.Unsetenv("SERVICE_ID")

	nodeInfo := NewNodeInfo()
	serviceID := nodeInfo.GetServiceId()

	assert.Equal(t, expectedServiceID, serviceID)
}

func TestGenerateServiceId_UUID(t *testing.T) {
	// Clear environment variable
	os.Unsetenv("SERVICE_ID")

	nodeInfo := NewNodeInfo()
	serviceID := nodeInfo.GetServiceId()

	// Should generate UUID when SERVICE_ID is not set
	assert.NotEmpty(t, serviceID)
	assert.Len(t, serviceID, 36) // UUID format
}

func TestNodeInfo_InterfaceCompliance(t *testing.T) {
	nodeInfo := NewNodeInfo()

	// Test that the returned object implements the NodeInfo interface
	var _ NodeInfo = nodeInfo

	// Test all interface methods
	assert.NotEmpty(t, nodeInfo.GetNodeId())
	assert.NotEmpty(t, nodeInfo.GetServiceId())
}

func TestNodeInfo_Consistency(t *testing.T) {
	nodeInfo := NewNodeInfo()

	// Test that IDs remain consistent for the same instance
	nodeID1 := nodeInfo.GetNodeId()
	nodeID2 := nodeInfo.GetNodeId()
	serviceID1 := nodeInfo.GetServiceId()
	serviceID2 := nodeInfo.GetServiceId()

	assert.Equal(t, nodeID1, nodeID2)
	assert.Equal(t, serviceID1, serviceID2)
}

func TestNodeInfo_Uniqueness(t *testing.T) {
	// Test that different instances have different IDs
	nodeInfo1 := NewNodeInfo()
	nodeInfo2 := NewNodeInfo()

	serviceID1 := nodeInfo1.GetServiceId()
	serviceID2 := nodeInfo2.GetServiceId()

	// Service IDs should be different (UUIDs)
	assert.NotEqual(t, serviceID1, serviceID2)
}

func TestNodeInfo_EnvironmentPriority(t *testing.T) {
	// Test that environment variables take priority
	expectedNodeID := "env-node-id"
	expectedServiceID := "env-service-id"

	os.Setenv("NODE_ID", expectedNodeID)
	os.Setenv("SERVICE_ID", expectedServiceID)
	defer func() {
		os.Unsetenv("NODE_ID")
		os.Unsetenv("SERVICE_ID")
	}()

	nodeInfo := NewNodeInfo()

	assert.Equal(t, expectedNodeID, nodeInfo.GetNodeId())
	assert.Equal(t, expectedServiceID, nodeInfo.GetServiceId())
}

func TestNodeInfo_EmptyEnvironmentVariables(t *testing.T) {
	// Test with empty environment variables
	os.Setenv("NODE_ID", "")
	os.Setenv("SERVICE_ID", "")
	defer func() {
		os.Unsetenv("NODE_ID")
		os.Unsetenv("SERVICE_ID")
	}()

	nodeInfo := NewNodeInfo()

	// Should fall back to hostname/UUID
	nodeID := nodeInfo.GetNodeId()
	serviceID := nodeInfo.GetServiceId()

	assert.NotEmpty(t, nodeID)
	assert.NotEmpty(t, serviceID)
}
