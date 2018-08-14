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

func resourceKubernetesRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesRoleCreate,
		Read:   resourceKubernetesRoleRead,
		Exists: resourceKubernetesRoleExists,
		Update: resourceKubernetesRoleUpdate,
		Delete: resourceKubernetesRoleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("role", true),
			"rule": {
				Type:        schema.TypeList,
				Description: "Rules defines the set of rules associated with the role",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_groups": {
							Type:        schema.TypeSet,
							Description: "A collection of api groups this rule applies to",
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Set:         schema.HashString,
						},
						"resources": {
							Type:        schema.TypeSet,
							Description: "A collection of resources this rule applies to",
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Set:         schema.HashString,
						},
						"verbs": {
							Type:        schema.TypeSet,
							Description: "A collection of API actions this rule applies to",
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Set:         schema.HashString,
						},
						"non_resource_urls": {
							Type:        schema.TypeSet,
							Description: "A collection of non resource urls this rule applies to",
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Set:         schema.HashString,
						},
						"resource_names": {
							Type:        schema.TypeSet,
							Description: "A collection of resource names this rule applies to",
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Set:         schema.HashString,
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesRoleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetesProvider).conn

	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	r := d.Get("rule")

	rules, err := expandRoleRules(r.([]interface{}))

	if err != nil {
		return err
	}

	role := rbac.Role{
		ObjectMeta: metadata,
		Rules:      rules,
	}

	log.Printf("[INFO] Creating new rbac role: %#v", role)
	out, err := conn.RbacV1beta1().Roles(metadata.Namespace).Create(&role)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new rbac role: %#v", out)
	d.SetId(buildId(metadata))
	return resourceKubernetesRoleRead(d, meta)
}

func resourceKubernetesRoleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetesProvider).conn

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading rbac role %s", name)
	role, err := conn.RbacV1beta1().Roles(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}

	log.Printf("[INFO] Received role: %#v", role)
	err = d.Set("metadata", flattenMetadata(role.ObjectMeta, d))
	if err != nil {
		return err
	}
	err = d.Set("rule", flattenRoleRules(role.Rules))
	if err != nil {
		return err
	}
	return nil
}

func resourceKubernetesRoleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetesProvider).conn

	namespace, name, err := idParts(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return false, err
	}

	log.Printf("[INFO] Checking role %s", name)
	_, err = conn.RbacV1beta1().Roles(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func resourceKubernetesRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetesProvider).conn

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	rules, err := expandRoleRules(d.Get("rule").([]interface{}))
	if err != nil {
		return err
	}

	ops = append(ops, &ReplaceOperation{
		Path:  "/rules",
		Value: rules,
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
	out, err := conn.RbacV1beta1().Roles(namespace).Patch(name, pkgApi.JSONPatchType, data)
	// conn.CoreV1().Namespaces().Patch(d.Id(), pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated role: %#v", out)

	return resourceKubernetesRoleRead(d, meta)
}

func resourceKubernetesRoleDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func expandRoleRules(l []interface{}) ([]rbac.PolicyRule, error) {
	rules := []rbac.PolicyRule{}

	if l == nil || len(l) == 0 || l[0] == nil {
		return rules, nil
	}

	for _, intf := range l {
		rule := rbac.PolicyRule{}

		in := intf.(map[string]interface{})

		if in["api_groups"] != nil {
			v := schemaSetToStringArray(in["api_groups"].(*schema.Set))
			rule.APIGroups = v
		}

		if in["resources"] != nil {
			v := schemaSetToStringArray(in["resources"].(*schema.Set))
			rule.Resources = v
		}

		if in["verbs"] != nil {
			v := schemaSetToStringArray(in["verbs"].(*schema.Set))
			rule.Verbs = v
		}

		if in["resource_names"] != nil {
			v := schemaSetToStringArray(in["resource_names"].(*schema.Set))
			rule.ResourceNames = v
		}

		if in["non_resource_urls"] != nil {
			v := schemaSetToStringArray(in["non_resource_urls"].(*schema.Set))
			rule.NonResourceURLs = v
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func flattenRoleRules(in []rbac.PolicyRule) []interface{} {
	var att []interface{}

	for _, v := range in {
		r := make(map[string]interface{})

		r["verbs"] = v.Verbs
		r["api_groups"] = v.APIGroups
		r["resources"] = v.Resources
		r["resource_names"] = v.ResourceNames
		r["non_resource_urls"] = v.NonResourceURLs

		att = append(att, r)
	}
	return att
}
