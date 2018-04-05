package kubernetes

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes"
	batchv2 "k8s.io/client-go/pkg/apis/batch/v2alpha1"
)

func resourceKubernetesCronJob() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesCronJobCreate,
		Read:   resourceKubernetesCronJobRead,
		Update: resourceKubernetesCronJobUpdate,
		Delete: resourceKubernetesCronJobDelete,
		Exists: resourceKubernetesCronJobExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("cronjob", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec of the cron job owned by the cluster",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: cronJobSpecFields(),
				},
			},
		},
	}
}

func resourceKubernetesCronJobCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandCronJobSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}
	spec.JobTemplate.ObjectMeta.Annotations = metadata.Annotations

	job := batchv2.CronJob{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new cron job: %#v", job)

	out, err := conn.BatchV2alpha1().CronJobs(metadata.Namespace).Create(&job)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new cron job: %#v", out)

	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesCronJobRead(d, meta)
}

func resourceKubernetesCronJobUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		spec, err := expandCronJobSpec(d.Get("spec").([]interface{}))
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

	log.Printf("[INFO] Updating cron job %s: %s", d.Id(), ops)

	out, err := conn.BatchV2alpha1().CronJobs(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated cron job: %#v", out)

	d.SetId(buildId(out.ObjectMeta))
	return resourceKubernetesCronJobRead(d, meta)
}

func resourceKubernetesCronJobRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading cron job %s", name)
	job, err := conn.BatchV2alpha1().CronJobs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received cron job: %#v", job)

	// Remove server-generated labels unless using manual selector
	if _, ok := d.GetOk("spec.0.manual_selector"); !ok {
		labels := job.ObjectMeta.Labels

		if _, ok := labels["controller-uid"]; ok {
			delete(labels, "controller-uid")
		}

		if _, ok := labels["cron-job-name"]; ok {
			delete(labels, "cron-job-name")
		}

		if job.Spec.JobTemplate.Spec.Selector != nil &&
			job.Spec.JobTemplate.Spec.Selector.MatchLabels != nil {
			labels = job.Spec.JobTemplate.Spec.Selector.MatchLabels
		}

		if _, ok := labels["controller-uid"]; ok {
			delete(labels, "controller-uid")
		}
	}

	err = d.Set("metadata", flattenMetadata(job.ObjectMeta, d))
	if err != nil {
		return err
	}

	jobSpec, err := flattenCronJobSpec(job.Spec, d)
	if err != nil {
		return err
	}

	err = d.Set("spec", jobSpec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesCronJobDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting cron job: %#v", name)
	err = conn.BatchV2alpha1().CronJobs(namespace).Delete(name, nil)
	if err != nil {
		return err
	}

	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.BatchV2alpha1().CronJobs(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		e := fmt.Errorf("Cron Job %s still exists", name)
		return resource.RetryableError(e)
	})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Cron Job %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesCronJobExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking cron job %s", name)
	_, err = conn.BatchV2alpha1().CronJobs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
