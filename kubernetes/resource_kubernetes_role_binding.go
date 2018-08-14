package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	rbac "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	//api "k8s.io/api/core/v1"
)

func resourceKubernetesRoleBinding() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesRoleBindingCreate,
		Read:   resourceKubernetesRoleBindingRead,
		Exists: resourceKubernetesRoleBindingExists,
		Update: resourceKubernetesRoleBindingUpdate,
		Delete: resourceKubernetesRoleBindingDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("role_binding", true),
			"subject": {
				Type:        schema.TypeList,
				Description: "Rules defines the set of rules associated with the role",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kind": {
							Type:        schema.TypeString,
							Description: "Kind of the subject (default: User)",
							Default:     "User",
							Optional:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Description: "Case sensitive name of the 'kind' object",
							Required:    true,
						},
						"api_group": {
							Type:        schema.TypeString,
							Description: "API Group of the subject",
							Optional:    true,
							Default:     "rbac.authorization.k8s.io",
						},
					},
				},
			},
			"role_ref": {
				Description: "Reference to the role (Role or ClusterRole)",
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kind": {
							Type:        schema.TypeString,
							Description: "Kind of the subject (default: Role)",
							Default:     "Role",
							Optional:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Description: "Role or ClusterRole name",
							Required:    true,
						},
						"api_group": {
							Type:        schema.TypeString,
							Description: "API Group of the subject",
							Optional:    true,
							Default:     "rbac.authorization.k8s.io",
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesRoleBindingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetesProvider).conn

	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	r := d.Get("subject")

	subjects, err := expandRoleBindingSubjects(r.([]interface{}))
	if err != nil {
		return err
	}

	roleRef, err := expandRoleBindingRoleRef(d.Get("role_ref").([]interface{}))
	roleBinding := rbac.RoleBinding{
		ObjectMeta: metadata,
		Subjects:   subjects,
		RoleRef:    roleRef,
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Creating new rbac role binding: %#v", roleBinding)
	out, err := conn.RbacV1beta1().RoleBindings(metadata.Namespace).Create(&roleBinding)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new rbac role binding: %#v", out)
	d.SetId(buildId(metadata))
	return resourceKubernetesRoleBindingRead(d, meta)
}

func resourceKubernetesRoleBindingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetesProvider).conn

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading rbac role binding %s", name)
	binding, err := conn.RbacV1beta1().RoleBindings(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}

	log.Printf("[INFO] Received role binding: %#v", binding)
	err = d.Set("metadata", flattenMetadata(binding.ObjectMeta, d))
	if err != nil {
		return err
	}
	err = d.Set("subject", flattenRoleBindingSubjects(binding.Subjects))
	if err != nil {
		return err
	}

	err = d.Set("role_ref", flattenRoleBindingRoleRef(binding.RoleRef))
	if err != nil {
		return err
	}
	return nil
}

func resourceKubernetesRoleBindingExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetesProvider).conn

	namespace, name, err := idParts(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return false, err
	}

	log.Printf("[INFO] Checking role binding %s", name)
	_, err = conn.RbacV1beta1().RoleBindings(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func resourceKubernetesRoleBindingUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetesProvider).conn

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	subjects, err := expandRoleBindingSubjects(d.Get("subject").([]interface{}))
	if err != nil {
		return err
	}

	roleref, err := expandRoleBindingRoleRef(d.Get("role_ref").([]interface{}))
	if err != nil {
		return err
	}

	ops = append(ops, &ReplaceOperation{
		Path:  "/subjects",
		Value: subjects,
	})

	ops = append(ops, &ReplaceOperation{
		Path:  "/roleRef",
		Value: roleref,
	})

	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Updating role: %s", ops)
	out, err := conn.RbacV1beta1().RoleBindings(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Submitted updated role binding: %#v", out)
	// d.SetId(out.Name)

	return resourceKubernetesRoleBindingRead(d, meta)
}

func resourceKubernetesRoleBindingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetesProvider).conn

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	err = conn.RbacV1beta1().RoleBindings(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func expandRoleBindingSubjects(l []interface{}) ([]rbac.Subject, error) {
	subjects := []rbac.Subject{}

	if l == nil || len(l) == 0 || l[0] == nil {
		return subjects, nil
	}

	for _, intf := range l {
		subject := rbac.Subject{}

		in := intf.(map[string]interface{})

		if in["api_group"] != nil {
			subject.APIGroup = in["api_group"].(string)
		}
		if in["kind"] != nil {
			subject.Kind = in["kind"].(string)
		}
		if in["name"] != nil {
			subject.Name = in["name"].(string)
		}
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

func expandRoleBindingRoleRef(l []interface{}) (rbac.RoleRef, error) {
	ref := rbac.RoleRef{}

	if len(l) == 0 || l[0] == nil {
		return ref, nil
	}
	in := l[0].(map[string]interface{})

	if in["api_group"] != nil {
		ref.APIGroup = in["api_group"].(string)
	}
	if in["kind"] != nil {
		ref.Kind = in["kind"].(string)
	}
	if in["name"] != nil {
		ref.Name = in["name"].(string)
	}

	return ref, nil
}

func flattenRoleBindingSubjects(in []rbac.Subject) []interface{} {
	var att []interface{}

	for _, v := range in {
		r := make(map[string]interface{})

		r["api_group"] = v.APIGroup
		r["name"] = v.Name
		r["kind"] = v.Kind
		att = append(att, r)
	}
	return att
}

func flattenRoleBindingRoleRef(in rbac.RoleRef) []interface{} {
	var ret []interface{}
	r := make(map[string]string)
	r["api_group"] = in.APIGroup
	r["name"] = in.Name
	r["kind"] = in.Kind

	return append(ret, r)
}
