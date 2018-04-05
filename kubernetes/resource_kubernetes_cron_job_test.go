package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	batchv2 "k8s.io/client-go/pkg/apis/batch/v2alpha1"
)

func TestAccKubernetesCronJob_basic(t *testing.T) {
	var conf batchv2.CronJob
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_cron_job.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesCronJobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCronJobConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobExists("kubernetes_cron_job.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job.test", "metadata.0.uid"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job.test", "spec.0.schedule"),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "spec.0.schedule", "1 0 * * *"),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "spec.0.job_template.0.spec.0.parallelism", "1"),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "spec.0.job_template.0.spec.0.template.0.spec.0.container.0.name", "hello"),
				),
			},
			{
				Config: testAccKubernetesCronJobConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobExists("kubernetes_cron_job.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "spec.0.schedule", "2 0 * * *"),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "spec.0.job_template.0.spec.0.parallelism", "2"),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "spec.0.job_template.0.spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "spec.0.job_template.0.spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cron_job.test", "spec.0.job_template.0.spec.0.template.0.metadata.0.labels.%", "2"),
				),
			},
		},
	})
}

func testAccCheckKubernetesCronJobDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_cron_job" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CronJobs(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("CronJob still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesCronJobExists(n string, obj *batchv2.CronJob) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.CronJobs(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesCronJobConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cron_job" "test" {
	metadata {
		name = "%s"
	}
	spec {
		schedule = "1 0 * * *"
		job_template {
			spec {
				template {
					spec {
						container {
							name = "hello"
							image = "alpine"
							command = ["echo", "'hello'"]
						}
					}
				}
			}
		}
	}
}`, name)
}

func testAccKubernetesCronJobConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cron_job" "test" {
	metadata {
		name = "%s"
	}
	spec {
		schedule = "2 0 * * *"
		job_template {
			spec {
				parallelism = 2
				template {
					metadata {
						labels {
							foo = "bar"
							baz = "foo"
						}
					}
					spec {
						container {
							name = "hello"
							image = "alpine"
							command = ["echo", "'abcdef'"]
						}
					}
				}
			}
		}
	}
}`, name)
}
