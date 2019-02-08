package kubernetes

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

const deploymentsResourceGroupName = "deployments"

var deploymentsAPIGroups = []APIGroup{appsV1, appsV1beta2, appsV1beta1, extensionsV1beta1}

var deploymentNotSupportedError = errors.New("could not find Kubernetes API group that supports Deployment resources")

func resourceKubernetesDeployment() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesDeploymentCreate,
		Read:   resourceKubernetesDeploymentRead,
		Exists: resourceKubernetesDeploymentExists,
		Update: resourceKubernetesDeploymentUpdate,
		Delete: resourceKubernetesDeploymentDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		SchemaVersion: 2,
		MigrateState:  resourceKubernetesDeploymentStateUpgrader,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("deployment", true),
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "To better match the Kubernetes API, the name attribute should be configured under the metadata block. Please update your Terraform configuration.",
			},
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the specification of the desired behavior of the deployment. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"min_ready_seconds": {
							Type:        schema.TypeInt,
							Description: "Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready)",
							Optional:    true,
							Default:     0,
						},
						"paused": {
							Type:        schema.TypeBool,
							Description: "Indicates that the deployment is paused.",
							Optional:    true,
							Default:     false,
						},
						"progress_deadline_seconds": {
							Type:        schema.TypeInt,
							Description: "The maximum time in seconds for a deployment to make progress before it is considered to be failed. The deployment controller will continue to process failed deployments and a condition with a ProgressDeadlineExceeded reason will be surfaced in the deployment status. Note that progress will not be estimated during the time a deployment is paused. Defaults to 600s.",
							Optional:    true,
							Default:     600,
						},
						"replicas": {
							Type:        schema.TypeInt,
							Description: "The number of desired replicas. Defaults to 1. More info: http://kubernetes.io/docs/user-guide/replication-controller#what-is-a-replication-controller",
							Optional:    true,
							Default:     1,
						},
						"revision_history_limit": {
							Type:        schema.TypeInt,
							Description: "The number of old ReplicaSets to retain to allow rollback. Defaults to 10.",
							Optional:    true,
							Default:     10,
						},
						"selector": {
							Type:        schema.TypeMap,
							Description: "A label query over pods that should match the Replicas count. If Selector is empty, it is defaulted to the labels present on the Pod template. Label keys and values that must match in order to be controlled by this deployment, if empty defaulted to labels on Pod template. More info: http://kubernetes.io/docs/user-guide/labels#label-selectors",
							Optional:    true,
							Computed:    true,
						},
						"strategy": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: "Update strategy. One of RollingUpdate, Recreate. Defaults to RollingDate",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "Update strategy",
									},
									"rolling_update": {
										Type:        schema.TypeList,
										Description: "rolling update",
										Optional:    true,
										Computed:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"max_surge": {
													Type:        schema.TypeString,
													Description: "The maximum number of pods that can be scheduled above the desired number of pods. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). This can not be 0 if MaxUnavailable is 0. Absolute number is calculated from percentage by rounding up. Defaults to 25%. Example: when this is set to 30%, the new RC can be scaled up immediately when the rolling update starts, such that the total number of old and new pods do not exceed 130% of desired pods. Once old pods have been killed, new RC can be scaled up further, ensuring that total number of pods running at any time during the update is atmost 130% of desired pods.",
													Optional:    true,
													Default:     "25%",
												},
												"max_unavailable": {
													Type:        schema.TypeString,
													Description: "The maximum number of pods that can be unavailable during the update. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). Absolute number is calculated from percentage by rounding down. This can not be 0 if MaxSurge is 0. Defaults to 25%. Example: when this is set to 30%, the old RC can be scaled down to 70% of desired pods immediately when the rolling update starts. Once new pods are ready, old RC can be scaled down further, followed by scaling up the new RC, ensuring that the total number of pods available at all times during the update is at least 70% of desired pods.",
													Optional:    true,
													Default:     "25%",
												},
											},
										},
									},
								},
							},
						},
						"template": {
							Type:        schema.TypeList,
							Description: "Template describes the pods that will be created.",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"metadata": metadataSchema("deploymentSpec", true),
									"spec": &schema.Schema{
										Type:        schema.TypeList,
										Description: "Template describes the pods that will be created.",
										Required:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: podSpecFields(true),
										},
									},
									"active_deadline_seconds":          relocatedAttribute("active_deadline_seconds"),
									"container":                        relocatedAttribute("container"),
									"dns_policy":                       relocatedAttribute("dns_policy"),
									"host_ipc":                         relocatedAttribute("host_ipc"),
									"host_network":                     relocatedAttribute("host_network"),
									"host_pid":                         relocatedAttribute("host_pid"),
									"hostname":                         relocatedAttribute("hostname"),
									"init_container":                   relocatedAttribute("init_container"),
									"node_name":                        relocatedAttribute("node_name"),
									"node_selector":                    relocatedAttribute("node_selector"),
									"restart_policy":                   relocatedAttribute("restart_policy"),
									"security_context":                 relocatedAttribute("security_context"),
									"service_account_name":             relocatedAttribute("service_account_name"),
									"automount_service_account_token":  relocatedAttribute("automount_service_account_token"),
									"subdomain":                        relocatedAttribute("subdomain"),
									"termination_grace_period_seconds": relocatedAttribute("termination_grace_period_seconds"),
									"volume":                           relocatedAttribute("volume"),
								},
							},
						},
					},
				},
			},
		},
	}
}

func relocatedAttribute(name string) *schema.Schema {
	s := &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Removed:  fmt.Sprintf("%s has been relocated to [resource].spec.template.spec.%s. Please update your Terraform config.", name, name),
	}
	return s
}

func resourceKubernetesDeploymentCreate(d *schema.ResourceData, meta interface{}) error {
	kp := meta.(*kubernetesProvider)
	conn := meta.(*kubernetesProvider).conn

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandDeploymentSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}
	if metadata.Namespace == "" {
		metadata.Namespace = "default"
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	outDeploymentV1 := &appsv1.Deployment{}

	log.Printf("[INFO] Creating new deployment: %#v", deployment)
	apiGroup, err := kp.highestSupportedAPIGroup(deploymentsResourceGroupName, deploymentsAPIGroups...)
	if err != nil {
		return err
	}
	switch apiGroup {
	case appsV1:
		// Push deployment to API, and capture resultant object
		outDeploymentV1, err = conn.AppsV1().Deployments(metadata.Namespace).Create(&deployment)

	case appsV1beta2:
		beta := &appsv1beta2.Deployment{}
		err = Convert(&deployment, beta)
		if err != nil {
			break
		}

		out, err2 := conn.AppsV1beta2().Deployments(metadata.Namespace).Create(beta)
		if err2 != nil {
			err = err2
			break
		}

		err = Convert(out, outDeploymentV1)
		if err != nil {
			break
		}

	case appsV1beta1:
		beta := &appsv1beta1.Deployment{}
		err = Convert(&deployment, beta)
		if err != nil {
			break
		}

		var outDeploymentV1beta1 *appsv1beta1.Deployment
		outDeploymentV1beta1, err = conn.AppsV1beta1().Deployments(metadata.Namespace).Create(beta)
		if err != nil {
			break
		}

		err = Convert(outDeploymentV1beta1, outDeploymentV1)
		if err != nil {
			break
		}

	case extensionsV1beta1:
		beta := &extensionsv1beta1.Deployment{}
		err = Convert(&deployment, beta)
		if err != nil {
			break
		}

		var outDeploymentV1beta1 *extensionsv1beta1.Deployment
		outDeploymentV1beta1, err = conn.ExtensionsV1beta1().Deployments(metadata.Namespace).Create(beta)
		if err != nil {
			break
		}

		err = Convert(outDeploymentV1beta1, outDeploymentV1)
		if err != nil {
			break
		}

	default:
		err = deploymentNotSupportedError
	}
	if err != nil {
		return fmt.Errorf("Failed to create deployment: %s", err)
	}

	log.Printf("[INFO] Created deployment: %s", outDeploymentV1.ObjectMeta.SelfLink)

	d.SetId(buildId(outDeploymentV1.ObjectMeta))
	// deployment.ObjectMeta.Labels = reconcileTopLevelLabels(
	// 	deployment.ObjectMeta.Labels,
	// 	expandMetadata(d.Get("metadata").([]interface{})),
	// 	expandMetadata(d.Get("spec.0.template.0.metadata").([]interface{})),
	// )
	// err = d.Set("metadata", flattenMetadata(out.ObjectMeta, d))
	// if err != nil {
	// 	return err
	// }

	log.Printf("[DEBUG] Waiting for deployment %s to schedule %d replicas",
		d.Id(), *outDeploymentV1.Spec.Replicas)
	// 10 mins should be sufficient for scheduling ~10k replicas
	err = resource.Retry(d.Timeout(schema.TimeoutCreate),
		waitForDeploymentReplicasFunc(
			kp,
			outDeploymentV1.GetNamespace(),
			outDeploymentV1.GetName(),
		),
	)
	if err != nil {
		return err
	}
	// We could wait for all pods to actually reach Ready state
	// but that means checking each pod status separately (which can be expensive at scale)
	// as there's no aggregate data available from the API

	log.Printf("[INFO] Submitted new deployment: %#v", outDeploymentV1)

	return resourceKubernetesDeploymentRead(d, meta)
}

func resourceKubernetesDeploymentRead(d *schema.ResourceData, meta interface{}) error {
	kp := meta.(*kubernetesProvider)

	namespace, name, err := idParts(d.Id())
	deployment, err := readDeployment(kp, namespace, name)
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received deployment: %#v", deployment)

	deployment.ObjectMeta.Labels = reconcileTopLevelLabels(
		deployment.ObjectMeta.Labels,
		expandMetadata(d.Get("metadata").([]interface{})),
		expandMetadata(d.Get("spec.0.template.0.metadata").([]interface{})),
	)
	err = d.Set("metadata", flattenMetadata(deployment.ObjectMeta, d))
	if err != nil {
		return err
	}

	spec, err := flattenDeploymentSpec(deployment.Spec, d)
	if err != nil {
		return err
	}

	err = d.Set("spec", spec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesDeploymentUpdate(d *schema.ResourceData, meta interface{}) error {
	kp := meta.(*kubernetesProvider)
	namespace, name, err := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		spec, err := expandDeploymentSpec(d.Get("spec").([]interface{}))
		if err != nil {
			return err
		}

		ops = append(ops, &ReplaceOperation{
			Path:  "/spec",
			Value: spec,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating deployment %q: %v", name, string(data))

	out, err := resourceKubernetesPatchDeployment(d, kp, data)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Submitted updated deployment: %#v", out)

	err = resource.Retry(d.Timeout(schema.TimeoutUpdate),
		waitForDeploymentReplicasFunc(kp, namespace, name))
	if err != nil {
		return err
	}

	return resourceKubernetesDeploymentRead(d, meta)
}

func resourceKubernetesPatchDeployment(d *schema.ResourceData, kp *kubernetesProvider, data []byte) (deployment *appsv1.Deployment, err error) {
	conn := kp.conn
	deployment = &appsv1.Deployment{}

	namespace, name, err := idParts(d.Id())
	apiGroup, err := kp.highestSupportedAPIGroup(deploymentsResourceGroupName, deploymentsAPIGroups...)
	if err != nil {
		return nil, err
	}

	switch apiGroup {
	case appsV1:
		deployment, err = conn.AppsV1().Deployments(namespace).Patch(name, pkgApi.JSONPatchType, data)
		if err != nil {
			return
		}

	case appsV1beta2:
		beta, err := conn.AppsV1beta2().Deployments(namespace).Patch(name, pkgApi.JSONPatchType, data)
		if err != nil {
			return nil, err
		}

		err = Convert(beta, deployment)
		if err != nil {
			return nil, err
		}

	case appsV1beta1:
		beta, err := conn.AppsV1beta1().Deployments(namespace).Patch(name, pkgApi.JSONPatchType, data)
		if err != nil {
			return nil, err
		}

		err = Convert(beta, deployment)
		if err != nil {
			return nil, err
		}

	case extensionsV1beta1:
		beta, err := conn.ExtensionsV1beta1().Deployments(namespace).Patch(name, pkgApi.JSONPatchType, data)
		if err != nil {
			return nil, err
		}

		err = Convert(beta, deployment)
		if err != nil {
			return nil, err
		}

	default:
		err = deploymentNotSupportedError
	}

	return
}

func resourceKubernetesDeploymentDelete(d *schema.ResourceData, meta interface{}) error {
	kp := meta.(*kubernetesProvider)
	conn := kp.conn

	namespace, name, err := idParts(d.Id())
	log.Printf("[INFO] Deleting deployment: %#v", name)

	// Drain all replicas before deleting
	var ops PatchOperations
	ops = append(ops, &ReplaceOperation{
		Path:  "/spec/replicas",
		Value: 0,
	})
	data, err := ops.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = resourceKubernetesPatchDeployment(d, kp, data)
	if err != nil {
		return err
	}

	// Wait until all replicas are gone
	err = resource.Retry(d.Timeout(schema.TimeoutDelete),
		waitForDeploymentReplicasFunc(
			kp,
			namespace,
			name,
		),
	)
	if err != nil {
		return err
	}

	policy := metav1.DeletePropagationForeground
	apiGroup, err := kp.highestSupportedAPIGroup(deploymentsResourceGroupName, deploymentsAPIGroups...)
	if err != nil {
		return err
	}
	switch apiGroup {
	case appsV1:
		err = conn.AppsV1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{
			PropagationPolicy: &policy,
		})
	case appsV1beta2:
		err = conn.AppsV1beta2().Deployments(namespace).Delete(name, &metav1.DeleteOptions{
			PropagationPolicy: &policy,
		})
	case appsV1beta1:
		err = conn.AppsV1beta1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{
			PropagationPolicy: &policy,
		})
	case extensionsV1beta1:
		err = conn.ExtensionsV1beta1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{
			PropagationPolicy: &policy,
		})
	default:
		err = deploymentNotSupportedError
	}

	log.Printf("[INFO] Deployment %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesDeploymentExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	kp := meta.(*kubernetesProvider)

	namespace, name, err := idParts(d.Id())
	log.Printf("[INFO] Checking deployment %s", name)

	_, err = readDeployment(kp, namespace, name)
	if err != nil {
		if statusErr, ok := err.(*kerrors.StatusError); ok && statusErr.ErrStatus.Code == 404 && statusErr.ErrStatus.Message != "the server could not find the requested resource" {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}

	return true, err
}

func readDeployment(kp *kubernetesProvider, namespace, name string) (dep *appsv1.Deployment, err error) {
	conn := kp.conn

	log.Printf("[INFO] Reading deployment %s", name)
	dep = &appsv1.Deployment{}

	apiGroup, err := kp.highestSupportedAPIGroup(deploymentsResourceGroupName, deploymentsAPIGroups...)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Reading deployment using %s API Group", apiGroup)

	switch apiGroup {
	case appsV1:
		dep, err = conn.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		return dep, err

	case appsV1beta2:
		out, err := conn.AppsV1beta2().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		err = Convert(out, dep)

	case appsV1beta1:
		out, err := conn.AppsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		err = Convert(out, dep)

	case extensionsV1beta1:
		out, err := conn.ExtensionsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		err = Convert(out, dep)

	default:
		return nil, deploymentNotSupportedError
	}

	return dep, err
}

// func waitForDeploymentReplicasFunc(conn *kubernetes.Clientset, ns, name string) resource.RetryFunc {
func waitForDeploymentReplicasFunc(kp *kubernetesProvider, ns, name string) resource.RetryFunc {
	return func() *resource.RetryError {

		deployment, err := readDeployment(kp, ns, name)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		desiredReplicas := *deployment.Spec.Replicas
		log.Printf("[DEBUG] Current number of labelled replicas of %q: %d (of %d)\n",
			deployment.GetName(), deployment.Status.Replicas, desiredReplicas)

		if deployment.Status.Replicas == desiredReplicas {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("Waiting for %d replicas of %q to be scheduled (%d)",
			desiredReplicas, deployment.GetName(), deployment.Status.Replicas))
	}
}

func resourceKubernetesDeploymentStateUpgrader(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	var err error

	switch v {
	case 0:
		log.Println("[INFO] Found Kubernetes Deployment State v0; migrating to v1")
		is, err = migrateStateV0toV1(is)
	case 1:
		log.Println("[INFO] Found Kubernetes Deployment State v1; migrating to v2")
		is, err = migrateStateV1toV2(is)

	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}

	return is, err
}

// This deployment resource originally had the podSpec directly below spec.template level
// This migration moves the state to spec.template.spec to match the Kubernetes documented structure
func migrateStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	newTemplate := make(map[string]string)

	for k, v := range is.Attributes {
		log.Println("[DEBUG] - checking attribute for state upgrade: ", k, v)
		if strings.HasPrefix(k, "name") {
			// don't clobber an existing metadata.0.name value
			if _, ok := is.Attributes["metadata.0.name"]; ok {
				continue
			}

			newK := "metadata.0.name"

			newTemplate[newK] = v
			log.Printf("[DEBUG] moved attribute %s -> %s ", k, newK)
			delete(is.Attributes, k)

		} else if !strings.HasPrefix(k, "spec.0.template") {
			continue

		} else if strings.HasPrefix(k, "spec.0.template.0.spec") || strings.HasPrefix(k, "spec.0.template.0.metadata") {
			continue

		} else {
			newK := strings.Replace(k, "spec.0.template.0", "spec.0.template.0.spec.0", 1)

			newTemplate[newK] = v
			log.Printf("[DEBUG] moved attribute %s -> %s ", k, newK)
			delete(is.Attributes, k)
		}
	}

	for k, v := range newTemplate {
		is.Attributes[k] = v
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

// Add schema fields: paused, progress_deadline_seconds
func migrateStateV1toV2(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	is.Attributes["spec.0.paused"] = "false"
	is.Attributes["spec.0.progress_deadline_seconds"] = "600"

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}
