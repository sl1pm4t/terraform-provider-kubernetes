package kubernetes

import (
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func flattenDaemonSetSpec(in appsv1.DaemonSetSpec, d *schema.ResourceData) ([]interface{}, error) {
	att := make(map[string]interface{})
	att["min_ready_seconds"] = in.MinReadySeconds

	att["selector"] = in.Selector.MatchLabels
	att["strategy"] = flattenDaemonSetStrategy(in.UpdateStrategy)
	// podSpec, err := flattenPodSpec(in.Template.Spec)
	// if err != nil {
	// 	return nil, err
	// }
	// att["template"] = podSpec

	templateMetadata := flattenMetadata(in.Template.ObjectMeta, d)
	podSpec, err := flattenPodSpec(in.Template.Spec)
	if err != nil {
		return nil, err
	}
	template := make(map[string]interface{})
	template["metadata"] = templateMetadata
	template["spec"] = podSpec
	att["template"] = []interface{}{template}

	return []interface{}{att}, nil
}

func flattenDaemonSetStrategy(in appsv1.DaemonSetUpdateStrategy) []interface{} {
	att := make(map[string]interface{})
	if in.Type != "" {
		att["type"] = in.Type
	}
	if in.RollingUpdate != nil {
		att["rolling_update"] = flattenDaemonSetStrategyRollingUpdate(in.RollingUpdate)
	}
	return []interface{}{att}
}

func flattenDaemonSetStrategyRollingUpdate(in *appsv1.RollingUpdateDaemonSet) []interface{} {
	att := make(map[string]interface{})
	if in.MaxUnavailable != nil {
		att["max_unavailable"] = in.MaxUnavailable.String()
	}
	return []interface{}{att}
}

func expandDaemonSetSpec(deployment []interface{}) (appsv1.DaemonSetSpec, error) {
	obj := appsv1.DaemonSetSpec{}
	if len(deployment) == 0 || deployment[0] == nil {
		return obj, nil
	}
	in := deployment[0].(map[string]interface{})
	obj.MinReadySeconds = int32(in["min_ready_seconds"].(int))
	if v, ok := in["selector"]; ok {
		obj.Selector = &metav1.LabelSelector{
			MatchLabels: expandStringMap(v.(map[string]interface{})),
		}
	}
	obj.UpdateStrategy = expandDaemonSetStrategy(in["strategy"].([]interface{}))

	for _, v := range in["template"].([]interface{}) {
		template := v.(map[string]interface{})
		podSpec, err := expandPodSpec(template["spec"].([]interface{}))
		if err != nil {
			return obj, err
		}
		obj.Template = v1.PodTemplateSpec{
			Spec: podSpec,
		}

		if metaCfg, ok := template["metadata"]; ok {
			metadata := expandMetadata(metaCfg.([]interface{}))
			obj.Template.ObjectMeta = metadata
		}
	}

	return obj, nil
}

func expandDaemonSetStrategy(p []interface{}) appsv1.DaemonSetUpdateStrategy {
	obj := appsv1.DaemonSetUpdateStrategy{}

	if len(p) == 0 || p[0] == nil {
		obj.Type = appsv1.RollingUpdateDaemonSetStrategyType
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["type"]; ok {
		obj.Type = appsv1.DaemonSetUpdateStrategyType(v.(string))
	}
	if v, ok := in["rolling_update"]; ok {
		obj.RollingUpdate = expandRollingUpdateDaemonSet(v.([]interface{}))
	}
	return obj
}

func expandRollingUpdateDaemonSet(p []interface{}) *appsv1.RollingUpdateDaemonSet {
	obj := appsv1.RollingUpdateDaemonSet{}
	if len(p) == 0 || p[0] == nil {
		return &obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["max_surge"]; ok {
		obj.MaxSurge = expandRollingUpdateDaemonSetIntOrString(v.(string))
	}
	if v, ok := in["max_unavailable"]; ok {
		obj.MaxUnavailable = expandRollingUpdateDaemonSetIntOrString(v.(string))
	}
	return &obj
}

func expandRollingUpdateDaemonSetIntOrString(v string) *intstr.IntOrString {
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
