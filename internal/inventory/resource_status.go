package inventory

import "strings"

// ResourceStatus defines the status of the instance
type ResourceStatus string // @name ResourceStatus

const (
	// Running Instance status
	Running ResourceStatus = "Running"
	// Stopped Instance status
	Stopped ResourceStatus = "Stopped"
	// Terminated Instance status
	Terminated ResourceStatus = "Terminated"
	// DeleteFailed indicates a cluster deletion was attempted but did not complete
	DeleteFailed ResourceStatus = "DeleteFailed"
)

// AsResourceStatus converts the incoming argument into a ResourceStatus type
func AsResourceStatus(status string) ResourceStatus {
	switch strings.ToLower(status) {
	case "running":
		return Running
	case "stop":
		return Stopped
	case "stopped":
		return Stopped
	case "terminated":
		return Terminated
	case "deletefailed":
		return DeleteFailed
	default:
		return Running
	}
}
