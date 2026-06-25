package inventory

import (
	"regexp"
	"strings"
)

// TODO: The infraID suffix is assumed to be exactly 5 characters. Some OpenShift
// versions or configurations may use longer suffixes. Validate against real
// clusters (IPI, UPI, SNO, ROSA, OSD) and adjust the regex if needed.
const (
	clusterNameRegexp = "kubernetes.io/cluster/(.*?)-.{5}$" // RegExp to get the Cluster Name configured by `openshift-installer` from Tags
	infraIDRegexp     = "kubernetes.io/cluster/.*-(.{5}?)$" // RegExp to get the InfrastructureID configured by `openshift-installer` from Tags
	clusterIDRegexp   = "kubernetes.io/cluster/(.+)$"       // RegExp to get the ClusterID (ClusterName + InfraID) configured by `openshift-installer` from Tags
)

const (
	RedHatClusterTypeTag  = "red-hat-clustertype"
	RedHatManagedTag      = "red-hat-managed"
	OpenshiftClusterIDTag = "openshiftClusterID"
)

// Tag model generic tags as a Key-Value object
type Tag struct {
	// Tag's key
	Key string `db:"key"`

	// Tag's Value
	Value string `db:"value"`

	// InstanceName reference
	InstanceID string `db:"instance_id"`
}

// NewTag returns a new generic tag struct
func NewTag(key string, value string, instanceID string) *Tag {
	return &Tag{Key: key, Value: value, InstanceID: instanceID}
}

// lookForTagByKey looks for a Tag based on its Key and returns a pointer to it
func LookForTagByKey(key string, tags []Tag) *Tag {
	for _, tag := range tags {
		if tag.Key == key {
			return &tag
		}
	}
	return nil
}

// parseClusterName parses a Tag key to obtain the clusterName
func parseClusterName(key string) string {
	re := regexp.MustCompile(clusterNameRegexp)
	res := re.FindAllStringSubmatch(key, 1)

	// if there are no results, return empty string, if there are, return first match
	if len(res) == 0 {
		return UnknownClusterNameCode
	}
	return res[0][1]
}

// parseClusterName parses a Tag key to obtain the clusterID
func parseClusterID(key string) string {
	re := regexp.MustCompile(clusterIDRegexp)
	res := re.FindAllStringSubmatch(key, 1)

	// if there are no results, return empty string, if there are, return first match
	if len(res) == 0 {
		return UnknownClusterIDCode
	}
	return res[0][1]
}

// parseInfraID parses a Tag key to obtain the InfraID
func parseInfraID(key string) string {
	re := regexp.MustCompile(infraIDRegexp)
	res := re.FindAllStringSubmatch(key, 1)

	// if there are no results, return empty string, if there are, return first match
	if len(res) == 0 {
		return ""
	}
	return res[0][1]
}

// GetOwnerFromTags looks for a tag with the key "Owner" and returns its value
func GetOwnerFromTags(tags []Tag) string {
	result := LookForTagByKey("Owner", tags)
	if result != nil {
		return result.Value
	}
	return ""
}

func GetInstanceNameFromTags(tags []Tag) string {
	result := LookForTagByKey("Name", tags)
	if result != nil {
		return result.Value
	}
	return ""
}

func GetClusterIDFromTags(tags []Tag) string {
	for _, tag := range tags {
		if strings.Contains(tag.Key, ClusterTagKey) {
			return parseClusterID(tag.Key)
		}
	}
	return UnknownClusterNameCode
}

func GetClusterNameFromTags(tags []Tag) string {
	for _, tag := range tags {
		if strings.Contains(tag.Key, ClusterTagKey) {
			return parseClusterName(tag.Key)
		}
	}
	return UnknownClusterNameCode
}

func GetInfraIDFromTags(tags []Tag) string {
	for _, tag := range tags {
		if strings.Contains(tag.Key, ClusterTagKey) {
			return parseInfraID(tag.Key)
		}
	}
	return UnknownClusterNameCode
}

// GetClusterTypeFromTags detects the OpenShift cluster type from AWS instance tags.
// ROSA and OSD clusters have reserved tags (red-hat-clustertype, red-hat-managed)
// that cannot be set manually. Self-managed clusters (IPI/UPI/SNO) lack these tags.
func GetClusterTypeFromTags(tags []Tag) ClusterType {
	clusterTypeTag := LookForTagByKey(RedHatClusterTypeTag, tags)
	if clusterTypeTag != nil {
		switch strings.ToLower(clusterTypeTag.Value) {
		case "rosa":
			return Rosa
		case "osd":
			return Osd
		}
	}

	managedTag := LookForTagByKey(RedHatManagedTag, tags)
	if managedTag != nil && strings.ToLower(managedTag.Value) == "true" {
		return Rosa
	}

	return SelfManaged
}

// GetOpenshiftClusterIDFromTags extracts the OpenShift cluster UUID from instance tags.
// This UUID is needed to reconstruct metadata.json for openshift-install destroy.
func GetOpenshiftClusterIDFromTags(tags []Tag) string {
	result := LookForTagByKey(OpenshiftClusterIDTag, tags)
	if result != nil {
		return result.Value
	}
	return ""
}
