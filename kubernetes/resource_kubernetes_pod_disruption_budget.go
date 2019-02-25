package kubernetes

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
							ForceNew:    true,
						},
						"selector": {
							Type:        schema.TypeList,
							Description: "A label query over volumes to consider for binding.",
							Optional:    true,
							ForceNew:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"match_expressions": {
										Type:        schema.TypeList,
										Description: "A list of label selector requirements. The requirements are ANDed.",
										Optional:    true,
										ForceNew:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:        schema.TypeString,
													Description: "The label key that the selector applies to.",
													Optional:    true,
													ForceNew:    true,
												},
												"operator": {
													Type:        schema.TypeString,
													Description: "A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists` and `DoesNotExist`.",
													Optional:    true,
													ForceNew:    true,
												},
												"values": {
													Type:        schema.TypeSet,
													Description: "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.",
													Optional:    true,
													ForceNew:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
													Set:         schema.HashString,
												},
											},
										},
									},
									"match_labels": {
										Type:        schema.TypeMap,
										Description: "A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.",
										Optional:    true,
										ForceNew:    true,
									},
								},
							},
						},
						"max_unavailable": {
							Type:        schema.TypeString,
							Description: "This is the number pf Pods that should be unavailable after an eviction. Setting this to 0 would prevent any voluntary evictions. This is a mutually exclusive setting with MinAvailable. Only available in Kubernetes 1.7 and higher.",
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesPodDisruptionBudgetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetesProvider).conn
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPodDisruptionBudgetSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	log.Printf("[INFO] Before Creating pod disruption budget: %#v", d)

	podDisruptionBudget := v1beta1.PodDisruptionBudget{
		ObjectMeta: metadata,
		Spec:       spec,
	}
	log.Printf("[INFO] Creating pod disruption budget: %#v", podDisruptionBudget)

	createdPDB, err := conn.PolicyV1beta1().PodDisruptionBudgets(metadata.Namespace).Create(&podDisruptionBudget)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new pod disruption budget: %#v", createdPDB)
	d.SetId(buildId(createdPDB.ObjectMeta))

	return resourceKubernetesPodDisruptionBudgetRead(d, meta)
}

func resourceKubernetesPodDisruptionBudgetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetesProvider).conn

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading pod disruption budget %s", name)
	pdb, err := conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received pod disruption budget: %#v", pdb)

	err = d.Set("metadata", flattenMetadata(pdb.ObjectMeta, d))
	if err != nil {
		return err
	}

	spec, err := flattenPodDisruptionBudgetSpec(pdb.Spec)
	if err != nil {
		return err
	}

	err = d.Set("spec", spec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesPodDisruptionBudgetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetesProvider).conn

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting pod disruption budget: %#v", name)
	err = conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Pod disruption budget %s deleted", name)

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")
	return nil
}
