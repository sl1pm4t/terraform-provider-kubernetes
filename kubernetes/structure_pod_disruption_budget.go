package kubernetes

import (
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
)

// Expander

func expandPodDisruptionBudgetSpec(podDisruptionBudgetSpec []interface{}) (v1beta1.PodDisruptionBudgetSpec, error) {
	obj := v1beta1.PodDisruptionBudgetSpec{}

	if len(podDisruptionBudgetSpec) == 0 || podDisruptionBudgetSpec[0] == nil {
		return obj, nil
	}
	in := podDisruptionBudgetSpec[0].(map[string]interface{})

	//MinAvailable
	if v, ok := in["min_available"]; ok {
		obj.MinAvailable = expandPodDisruptionBudgetSpecIntOrString(v.(string))
	}

	// Selector
	obj.Selector = &metav1.LabelSelector{
		MatchLabels: expandStringMap(in["selector"].(map[string]interface{})),
	}

	//MaxUnavailable
	if v, ok := in["max_unavailable"]; ok {
		obj.MaxUnavailable = expandPodDisruptionBudgetSpecIntOrString(v.(string))
	}

	return obj, nil
}

func expandPodDisruptionBudgetSpecIntOrString(v string) *intstr.IntOrString {
	i, err := strconv.Atoi(v)
	if err != nil {
		return &intstr.IntOrString{
			Type:   intstr.String,
			StrVal: v,
		}
	}
	return &intstr.IntOrString{
		Type:   intstr.Int,
		IntVal: int32(i),
	}
}
