package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccKubernetesPodDisruptionBudget_minimal(t *testing.T) {
	//var conf v1beta1.PodDisruptionBudget
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_pod_disruption_budget.test",
		Providers:     testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodDisruptionBudget_minimal(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.0.key", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.0.operator", "Exists"),
				),
			},
		},
	})
}

func testAccKubernetesPodDisruptionBudget_minimal(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_pod_disruption_budget" "test" {
  metadata {
    name = "%s"
  }
  spec {
    min_available = "50%%"
    selector {
      match_expressions {
      	key = "foo"
      	operator = "Exists"
      }
	}
  }
}
`, name)
}


