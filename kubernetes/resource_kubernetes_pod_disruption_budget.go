package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/api/policy/v1beta1"
	"log"
)

func resourceKubernetesPodDisruptionBudget() *schema.Resource {

	return &schema.Resource{
		Create: resourceKubernetesPodDisruptionBudgetCreate,
		Read:   resourceKubernetesPodDisruptionBudgetRead,
		Delete: resourceKubernetesPodDisruptionBudgetDelete,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("pod disruption budget", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the specification of the Pod Disruption Budget.`",
				Required:    true,
				MaxItems:    1,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"min_available": {
							Type:        schema.TypeString,
							Description: "Minimum amount of pods that need to stay available. You can set it to 100% to prevent all voluntary evictions. This is a mutually exclusive setting with MaxUnavailable.",
							Optional:    true,
						},
						"selector": {
							Type:        schema.TypeMap,
							Description: "A label query over pods that should match the Pod Disruption Budget.",
							Required:    true,
							ForceNew:    true,
						},
						"max_unavailable": {
							Type:        schema.TypeString,
							Description: "This is the number pf Pods that should be unavailable after an eviction. Setting this to 0 would prevent any voluntary evictions. This is a mutually exclusive setting with MinAvailable. Only available in Kubernetes 1.7 and higher.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesPodDisruptionBudgetCreate(d *schema.ResourceData, meta interface{}) error {
	// get metadata properties from spec
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPodDisruptionBudgetSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	podDisruptionBudget := v1beta1.PodDisruptionBudget{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating Pod Disruption Budget: %#v", podDisruptionBudget)

	return resourceKubernetesPodDisruptionBudgetRead(d, meta)
}

func resourceKubernetesPodDisruptionBudgetRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceKubernetesPodDisruptionBudgetDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
