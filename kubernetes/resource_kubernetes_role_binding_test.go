package kubernetes

//kubernetes/resource_kubernetes_role_binding_binding_test.go
import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	rbac "k8s.io/api/rbac/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesRoleBinding_basic(t *testing.T) {

	var conf rbac.RoleBinding
	name := fmt.Sprintf("tf-rb-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_role_binding.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleBindingConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingExists("kubernetes_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.name", "test"),
				),
			},
			{
				Config: testAccKubernetesRoleBindingConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingExists("kubernetes_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.annotations.Different", "1234"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.name", "test"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.1.name", "test-two"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.name", "test"),
				),
			},
		},
	})
}

func testAccKubernetesRoleBindingConfig_basic(name string) string {
	return fmt.Sprintf(`

resource "kubernetes_role_binding" "test" {
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

	subject {
		name = "test"
	}

  role_ref {
    name = "test"
  }

}`, name)
}

func testAccKubernetesRoleBindingConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			Different = "1234"
		}
		labels {
			TestLabelOne = "one"
			TestLabelThree = "three"
		}
		name = "%s"
	}

  subject {
		name = "test"
	}

	subject {
		name = "test-two"
	}

  role_ref {
    name = "test"
  }
}`, name)
}

func testAccCheckKubernetesRoleBindingExists(n string, obj *rbac.RoleBinding) resource.TestCheckFunc {
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

		out, err := conn.RbacV1beta1().RoleBindings(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}
