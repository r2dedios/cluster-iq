package inventory

import "strings"

// ClusterType defines the type of OpenShift cluster based on how it was provisioned
type ClusterType string // @name ClusterType

const (
	// SelfManaged covers IPI, UPI, and SNO clusters — deletable via openshift-install
	SelfManaged ClusterType = "SelfManaged"
	// Rosa is a Red Hat OpenShift Service on AWS cluster — managed, NOT deletable
	Rosa ClusterType = "Rosa"
	// Osd is an OpenShift Dedicated cluster — managed, NOT deletable
	Osd ClusterType = "Osd"
)

// AsClusterType converts the incoming argument into a ClusterType
func AsClusterType(clusterType string) ClusterType {
	switch strings.ToLower(clusterType) {
	case "selfmanaged":
		return SelfManaged
	case "rosa":
		return Rosa
	case "osd":
		return Osd
	default:
		return SelfManaged
	}
}

// IsDeletable returns true if the cluster type supports deletion via openshift-install
func (ct ClusterType) IsDeletable() bool {
	return ct == SelfManaged
}
