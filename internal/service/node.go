package service

import (
	"os"

	"github.com/google/uuid"
)

type NodeInfo interface {
	GetNodeId() string
	GetServiceId() string
}

type nodeInfo struct {
	nodeId    string
	serviceId string
}

func NewNodeInfo() NodeInfo {
	nodeId := generateNodeId()
	serviceId := generateServiceId()
	
	return &nodeInfo{
		nodeId:    nodeId,
		serviceId: serviceId,
	}
}

func (n *nodeInfo) GetNodeId() string {
	return n.nodeId
}

func (n *nodeInfo) GetServiceId() string {
	return n.serviceId
}

func generateNodeId() string {
	// Try to get from environment
	if nodeId := os.Getenv("NODE_ID"); nodeId != "" {
		return nodeId
	}
	
	// Try to get hostname
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	
	// Generate UUID as fallback
	return uuid.New().String()
}

func generateServiceId() string {
	// Try to get from environment
	if serviceId := os.Getenv("SERVICE_ID"); serviceId != "" {
		return serviceId
	}
	
	// Generate UUID
	return uuid.New().String()
}