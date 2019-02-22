package kubernetes

import (
	"errors"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"log"
	"strconv"
)

// Flatten
func flattenPodDisruptionBudgetSpec(in v1beta1.PodDisruptionBudgetSpec) ([]interface{}, error) {
	att := make(map[string]interface{})
	if in.MinAvailable != nil {
		att["min_available"] = in.MinAvailable.String()
	}

	att["selector"] = in.Selector.MatchLabels

	if in.MaxUnavailable != nil {
		att["max_unavailable"] = in.MaxUnavailable.String()
	}

	return []interface{}{att}, nil

}

// Expand
func expandPodDisruptionBudgetSpec(podDisruptionBudgetSpec []interface{}) (v1beta1.PodDisruptionBudgetSpec, error) {
	obj := v1beta1.PodDisruptionBudgetSpec{}
	if len(podDisruptionBudgetSpec) == 0 || podDisruptionBudgetSpec[0] == nil {
		return obj, nil
	}

	in := podDisruptionBudgetSpec[0].(map[string]interface{})
	log.Printf("[DEBUG] This is the instance the map we have extracted in expand method: %#v", in)

	// Selector
	obj.Selector = &metav1.LabelSelector{
		MatchLabels: expandStringMap(in["selector"].(map[string]interface{})),
	}

	// Validate spec here, to chuck us out if we have set both min & max set
	if in["min_available"] != "" && in["max_unavailable"] != "" {
		log.Print("[DEBUG] You cannot set Min Available && Max Unavailable together")
		return obj, errors.New("You cannot set Min Available && Max Unavailable together")
	}

	//MinAvailable
	if v, ok := in["min_available"]; ok {
		if v.(string) != "" {
			obj.MinAvailable = expandPodDisruptionBudgetSpecIntOrString(v.(string))
		}
	}

	//MaxUnavailable
	if v, ok := in["max_unavailable"]; ok {
		if v.(string) != "" {
			log.Printf("[DEBUG] max_unavailable isset here :/ : %#v", in["max_unavailable"])
			obj.MaxUnavailable = expandPodDisruptionBudgetSpecIntOrString(v.(string))
		}
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
