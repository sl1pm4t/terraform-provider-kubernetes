package kubernetes

import (
	"flag"
	"fmt"
	"testing"

	"github.com/golang/glog"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	rbac "k8s.io/api/rbac/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesRole_basic(t *testing.T) {
	flag.Set("v", "2")
	flag.Set("alsologtostderr", "true")
	flag.Parse()
	glog.Info("Logging configured")

	var conf rbac.Role
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_role.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleConfig_emptyRules(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleExists("kubernetes_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesRolesConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleExists("kubernetes_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.annotations.Different", "1234"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rules.0.template.0.rules.0.container.0.image", "nginx:1.7.9"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rules.0.template.0.rules.0.container.0.name", "tf-acc-test"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rules.0.paused", "true"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rules.0.progress_deadline_seconds", "30"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rules.0.revision_history_limit", "4"),
				),
			},
			{
				Config: testAccKubernetesRoleConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleExists("kubernetes_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rules.0.template.0.rules.0.container.0.image", "nginx:1.7.8"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rules.0.template.0.rules.0.container.0.name", "tf-acc-test"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rules.0.paused", "false"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rules.0.progress_deadline_seconds", "600"),
				),
			},
		},
	})
}

func testAccKubernetesRoleConfig_emptyRules(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			TestAnnotationTwo = "two"
		}
		labels {
			TestLabelOne = "one"
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
		name = "%s"
	}

}`, name)
}

func testAccKubernetesRoleConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			TestAnnotationTwo = "two"
		}
		labels {
			TestLabelOne = "one"
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
		name = "%s"
	}

	rule {
		api_groups = [ "core/v1" ]
		verbs =  [ "get" ]
	}

}`, name)
}

func testAccKubernetesRolesConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			Different = "1234"
		}
		labels {
			TestLabelOne = "one"
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
		name = "%s"
	}

	rule {
		api_groups = [ "core/v1" ]
		resources = ["pods"]
		verbs =  [ "get" ]
	}
}`, name)
}
func testAccCheckKubernetesRoleExists(n string, obj *rbac.Role) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetesProvider).conn

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.RbacV1beta1().Roles(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccCheckKubernetesRoleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetesProvider).conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_role" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		err = conn.RbacV1beta1().Roles(namespace).Delete(name, &meta_v1.DeleteOptions{})
		if err != nil {
			return err
		}

		resp, err := conn.RbacV1beta1().Roles(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Role still exists: %s", rs.Primary.ID)
			}
		}
	}
	return nil
}
