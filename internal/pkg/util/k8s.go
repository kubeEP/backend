package util

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	v1Core "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/component-helpers/scheduling/corev1"
	"regexp"
	"strings"
)

func ParseGCPNodePoolByNodeName(projectName, nodeName string) string {
	nodePoolName := strings.ReplaceAll(nodeName, fmt.Sprintf("gke-%s-", projectName), "")
	re := regexp.MustCompile("-[a-z0-9]{8}-[a-z0-9]{4}$")
	return re.ReplaceAllString(nodePoolName, "")
}

func CheckPodNodePoolMatch(
	nodeLabels labels.Set,
	podNodeAffinity *v1.NodeAffinity,
	nodeSelector labels.Selector,
) (res bool, err error) {
	matchNodeSelector := nodeSelector.Matches(nodeLabels)
	nodeData := &v1.Node{ObjectMeta: v1Core.ObjectMeta{Labels: nodeLabels}}
	matchNodeAffinity := true
	if podNodeAffinity != nil {
		if podNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			matchNodeAffinity, err = corev1.MatchNodeSelectorTerms(
				nodeData,
				podNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
			)
			if err != nil {
				return false, err
			}
		}
	}

	return matchNodeSelector && matchNodeAffinity, nil

}
