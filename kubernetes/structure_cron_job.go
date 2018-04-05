package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	batchv2 "k8s.io/client-go/pkg/apis/batch/v2alpha1"
)

func flattenCronJobSpec(in batchv2.CronJobSpec, d *schema.ResourceData) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["concurrency_policy"] = in.ConcurrencyPolicy
	if in.FailedJobsHistoryLimit != nil {
		att["failed_jobs_history_limit"] = int(*in.FailedJobsHistoryLimit)
	} else {
		att["failed_jobs_history_limit"] = 1
	}

	att["schedule"] = in.Schedule

	jobTemplate, err := flattenJobTemplate(in.JobTemplate, d)
	if err != nil {
		return nil, err
	}
	att["job_template"] = jobTemplate

	if in.StartingDeadlineSeconds != nil {
		att["starting_deadline_seconds"] = int(*in.StartingDeadlineSeconds)
	} else {
		att["starting_deadline_seconds"] = 0
	}

	if in.SuccessfulJobsHistoryLimit != nil {
		att["successful_jobs_history_limit"] = int(*in.SuccessfulJobsHistoryLimit)
	} else {
		att["successful_jobs_history_limit"] = 3
	}

	return []interface{}{att}, nil
}

func flattenJobTemplate(in batchv2.JobTemplateSpec, d *schema.ResourceData) ([]interface{}, error) {
	att := make(map[string]interface{})

	meta := flattenMetadata(in.ObjectMeta, d)
	att["metadata"] = meta

	jobSpec, err := flattenJobSpec(in.Spec, d)
	if err != nil {
		return nil, err
	}
	att["spec"] = jobSpec

	return []interface{}{att}, nil
}

func expandCronJobSpec(j []interface{}) (batchv2.CronJobSpec, error) {
	obj := batchv2.CronJobSpec{}

	if len(j) == 0 || j[0] == nil {
		return obj, nil
	}

	in := j[0].(map[string]interface{})

	obj.ConcurrencyPolicy = batchv2.ConcurrencyPolicy(in["concurrency_policy"].(string))

	if v, ok := in["failed_jobs_history_limit"].(int); ok && v != 1 {
		obj.FailedJobsHistoryLimit = ptrToInt32(int32(v))
	}

	obj.Schedule = in["schedule"].(string)

	jtSpec, err := expandJobTemplate(in["job_template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.JobTemplate = jtSpec

	if v, ok := in["starting_deadline_seconds"].(int); ok && v > 0 {
		obj.StartingDeadlineSeconds = ptrToInt64(int64(v))
	}

	if v, ok := in["successful_jobs_history_limit"].(int); ok && v != 3 {
		obj.StartingDeadlineSeconds = ptrToInt64(int64(v))
	}

	if v, ok := in["suspend"].(bool); ok {
		obj.Suspend = ptrToBool(v)
	}

	return obj, nil
}

func expandJobTemplate(in []interface{}) (batchv2.JobTemplateSpec, error) {
	obj := batchv2.JobTemplateSpec{}

	tpl := in[0].(map[string]interface{})

	spec, err := expandJobSpec(tpl["spec"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Spec = spec

	if metaCfg, ok := tpl["metadata"]; ok {
		metadata := expandMetadata(metaCfg.([]interface{}))
		obj.ObjectMeta = metadata
	}

	return obj, nil
}

func patchCronJobSpec(pathPrefix, prefix string, d *schema.ResourceData) (PatchOperations, error) {
	ops := make([]PatchOperation, 0)

	return ops, nil
}
