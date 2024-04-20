package common

type ClusterOperation[T InstanceAttributes] struct {
	InstanceAttributes T
	ResourceClient     ResourceClient
}

func NewClusterOperation(ia InstanceAttributes, resourceClient ResourceClient) *ClusterOperation[InstanceAttributes] {
	return &ClusterOperation[InstanceAttributes]{
		InstanceAttributes: ia,
		ResourceClient:     resourceClient,
	}
}

func (c *ClusterOperation[T]) ReconciliationPaused() bool {
	return c.InstanceAttributes.GetClusterOperation() != nil && c.InstanceAttributes.GetClusterOperation().ReconciliationPaused
}

func (c *ClusterOperation[T]) ClusterStop() bool {
	return c.InstanceAttributes.GetClusterOperation() != nil && c.InstanceAttributes.GetClusterOperation().Stopped
}
