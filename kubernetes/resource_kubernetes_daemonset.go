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
	"k8s.io/api/apps/v1"
	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const daemonSetResourceGroupName = "daemonsets"

var daemonSetAPIGroups = []APIGroup{appsV1, appsV1beta2, extensionsV1beta1}
var daemonSetNotSupportedError = errors.New("could not find Kubernetes API group that supports DaemonSet resources")

func resourceKubernetesDaemonSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesDaemonSetCreate,
		Read:   resourceKubernetesDaemonSetRead,
		Exists: resourceKubernetesDaemonSetExists,
		Update: resourceKubernetesDaemonSetUpdate,
		Delete: resourceKubernetesDaemonSetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		SchemaVersion: 1,
		MigrateState:  resourceKubernetesDaemonSetStateUpgrader,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("daemonset", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the specification of the desired behavior of the daemonset. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status",
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
						"selector": {
							Type:        schema.TypeMap,
							Description: "A label query over pods that should match the Replicas count. If Selector is empty, it is defaulted to the labels present on the Pod template. Label keys and values that must match in order to be controlled by this deployment, if empty defaulted to labels on Pod template. More info: http://kubernetes.io/docs/user-guide/labels#label-selectors",
							Required:    true,
						},
						"strategy": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: "Update strategy. One of RollingUpdate, Destroy. Defaults to RollingUpdate",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:        schema.TypeString,
										Optional:    true,
										Default:     "RollingUpdate",
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
													Description: "max surge",
													Optional:    true,
													Default:     1,
												},
												"max_unavailable": {
													Type:        schema.TypeString,
													Description: "max unavailable",
													Optional:    true,
													Default:     1,
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
									"metadata": metadataSchema("daemonsetSpec", true),
									"spec": &schema.Schema{
										Type:        schema.TypeList,
										Description: "Spec describes the pods that will be created.",
										Required:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: podSpecFields(false),
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

func buildDaemonSetObject(d *schema.ResourceData) (*v1.DaemonSet, error) {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandDaemonSetSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return nil, err
	}
	if metadata.Namespace == "" {
		metadata.Namespace = "default"
	}

	daemonset := v1.DaemonSet{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	return &daemonset, err
}

func resourceKubernetesDaemonSetCreate(d *schema.ResourceData, meta interface{}) error {
	kp := meta.(*kubernetesProvider)
	conn := kp.conn

	daemonset, err := buildDaemonSetObject(d)
	if err != nil {
		return err
	}

	out := &v1.DaemonSet{}
	log.Printf("[INFO] Creating new daemonset: %#v", daemonset)
	apiGroup, err := kp.highestSupportedAPIGroup(daemonSetResourceGroupName, daemonSetAPIGroups...)
	if err != nil {
		return err
	}
	switch apiGroup {
	case appsV1:
		out, err = conn.AppsV1().DaemonSets(daemonset.ObjectMeta.Namespace).Create(daemonset)

	case appsV1beta2:
		dsBeta := &v1beta2.DaemonSet{}
		Convert(daemonset, dsBeta)
		dsBeta, err = conn.AppsV1beta2().DaemonSets(daemonset.ObjectMeta.Namespace).Create(dsBeta)
		Convert(dsBeta, out)

	case extensionsV1beta1:
		dsBeta := &v1beta1.DaemonSet{}
		Convert(daemonset, dsBeta)
		dsBeta, err = conn.ExtensionsV1beta1().DaemonSets(daemonset.ObjectMeta.Namespace).Create(dsBeta)
		Convert(dsBeta, out)

	default:
		err = daemonSetNotSupportedError
	}
	if err != nil {
		return fmt.Errorf("Failed to create daemonset: %s", err)
	}

	d.SetId(buildId(out.ObjectMeta))

	log.Printf("[INFO] Submitted new daemonset: %#v", out)

	return resourceKubernetesDaemonSetRead(d, meta)
}

func resourceKubernetesDaemonSetRead(d *schema.ResourceData, meta interface{}) error {
	kp := meta.(*kubernetesProvider)
	namespace, name, err := idParts(d.Id())

	daemonset, err := readDaemonSet(kp, namespace, name)
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received daemonset: %#v", daemonset)

	daemonset.ObjectMeta.Labels = reconcileTopLevelLabels(
		daemonset.ObjectMeta.Labels,
		expandMetadata(d.Get("metadata").([]interface{})),
		expandMetadata(d.Get("spec.0.template.0.metadata").([]interface{})),
	)

	err = d.Set("metadata", flattenMetadata(daemonset.ObjectMeta, d))
	if err != nil {
		return err
	}

	spec, err := flattenDaemonSetSpec(daemonset.Spec, d)
	if err != nil {
		return err
	}

	err = d.Set("spec", spec)
	if err != nil {
		return err
	}

	return nil
}

func readDaemonSet(kp *kubernetesProvider, namespace, name string) (dset *v1.DaemonSet, err error) {
	log.Printf("[INFO] Reading DaemonSet %s", name)
	conn := kp.conn
	dset = &v1.DaemonSet{}

	apiGroup, err := kp.highestSupportedAPIGroup(daemonSetResourceGroupName, daemonSetAPIGroups...)
	if err != nil {
		return nil, err
	}
	switch apiGroup {
	case appsV1:
		dset, err = conn.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
		return dset, err

	case appsV1beta2:
		out := &v1beta2.DaemonSet{}
		out, err = conn.AppsV1beta2().DaemonSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			break
		}
		err = Convert(out, dset)

	case extensionsV1beta1:
		out := &v1beta1.DaemonSet{}
		out, err = conn.ExtensionsV1beta1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			break
		}
		err = Convert(out, dset)

	default:
		return nil, daemonSetNotSupportedError
	}

	return dset, err
}

func resourceKubernetesDaemonSetUpdate(d *schema.ResourceData, meta interface{}) error {
	kp := meta.(*kubernetesProvider)
	conn := kp.conn
	namespace, name, err := idParts(d.Id())

	daemonset, err := buildDaemonSetObject(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Updating daemonset: %q", name)
	out := &v1.DaemonSet{}
	apiGroup, err := kp.highestSupportedAPIGroup(daemonSetResourceGroupName, daemonSetAPIGroups...)
	if err != nil {
		return err
	}
	switch apiGroup {
	case appsV1:
		out, err = conn.AppsV1().DaemonSets(namespace).Update(daemonset)

	case appsV1beta2:
		beta := &v1beta2.DaemonSet{}
		err = Convert(daemonset, beta)
		if err != nil {
			break
		}
		betaOut, err2 := conn.AppsV1beta2().DaemonSets(namespace).Update(beta)
		if err2 != nil {
			err = err2
			break
		}
		err = Convert(betaOut, out)
		if err != nil {
			break
		}

	case extensionsV1beta1:
		beta := &v1beta1.DaemonSet{}
		err = Convert(daemonset, beta)
		if err != nil {
			break
		}

		betaOut, err2 := conn.ExtensionsV1beta1().DaemonSets(namespace).Update(beta)
		if err != nil {
			err = err2
			break
		}
		err = Convert(betaOut, out)
		if err != nil {
			break
		}

	default:
		err = daemonSetNotSupportedError
	}

	if err != nil {
		return fmt.Errorf("Failed to update daemonset: %s", err)
	}
	log.Printf("[INFO] Submitted updated daemonset: %#v", out)

	err = resource.Retry(d.Timeout(schema.TimeoutUpdate),
		waitForDaemonSetReplicasFunc(kp, namespace, name))
	if err != nil {
		return err
	}

	return resourceKubernetesDaemonSetRead(d, meta)
}

func resourceKubernetesDaemonSetDelete(d *schema.ResourceData, meta interface{}) error {
	kp := meta.(*kubernetesProvider)
	conn := kp.conn

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[INFO] Deleting daemonset: %#v", name)

	policy := metav1.DeletePropagationForeground
	apiGroup, err := kp.highestSupportedAPIGroup(daemonSetResourceGroupName, daemonSetAPIGroups...)
	if err != nil {
		return err
	}
	switch apiGroup {
	case appsV1:
		conn.AppsV1().DaemonSets(namespace).Delete(name, &metav1.DeleteOptions{PropagationPolicy: &policy})
	case appsV1beta2:
		conn.AppsV1beta2().DaemonSets(namespace).Delete(name, &metav1.DeleteOptions{PropagationPolicy: &policy})
	case extensionsV1beta1:
		conn.ExtensionsV1beta1().DaemonSets(namespace).Delete(name, &metav1.DeleteOptions{PropagationPolicy: &policy})
	default:
		err = daemonSetNotSupportedError
	}

	log.Printf("[INFO] DaemonSet %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesDaemonSetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	kp := meta.(*kubernetesProvider)
	namespace, name, err := idParts(d.Id())
	log.Printf("[INFO] Checking daemonset %s", name)

	_, err = readDaemonSet(kp, namespace, name)
	if err != nil {
		if statusErr, ok := err.(*kerrors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func resourceKubernetesDaemonSetStateUpgrader(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	var err error

	switch v {
	case 0:
		log.Println("[INFO] Found Kubernetes DaemonSet State v0; migrating to v1")
		is, err = migrateDaemonSetStateV0toV1(is)
		if err != nil {
			return is, err
		}

	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}

	return is, err
}

// This deployment resource originally had the podSpec directly below spec.template level
// This migration moves the state to spec.template.spec match the Kubernetes documented structure
func migrateDaemonSetStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
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

func waitForDaemonSetReplicasFunc(kp *kubernetesProvider, ns, name string) resource.RetryFunc {
	return func() *resource.RetryError {
		daemonSet, err := readDaemonSet(kp, ns, name)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		desiredReplicas := daemonSet.Status.DesiredNumberScheduled
		log.Printf("[DEBUG] Current number of labelled replicas of %q: %d (of %d)\n",
			daemonSet.GetName(), daemonSet.Status.CurrentNumberScheduled, desiredReplicas)

		if daemonSet.Status.CurrentNumberScheduled == desiredReplicas {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("Waiting for %d replicas of %q to be scheduled (%d)",
			desiredReplicas, daemonSet.GetName(), daemonSet.Status.CurrentNumberScheduled))
	}
}
