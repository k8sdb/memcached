/*
Copyright The KubeDB Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package e2e_test

import (
	"fmt"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"
	"kubedb.dev/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"
	"kubedb.dev/memcached/test/e2e/framework"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	exec_util "kmodules.xyz/client-go/tools/exec"
)

var _ = Describe("Memcached", func() {
	var (
		err         error
		f           *framework.Invocation
		memcached   *api.Memcached
		skipMessage string
	)

	BeforeEach(func() {
		f = root.Invoke()
		memcached = f.Memcached()
		skipMessage = ""
	})

	AfterEach(func() {
		By("Check if memcached " + memcached.Name + " exists.")
		mg, err := f.GetMemcached(memcached.ObjectMeta)
		if err != nil && kerr.IsNotFound(err) {
			// Memcached was not created. Hence, rest of cleanup is not necessary.
			return
		}
		Expect(err).NotTo(HaveOccurred())

		By("Update memcached to set spec.terminationPolicy = WipeOut")
		_, err = f.PatchMemcached(mg.ObjectMeta, func(in *api.Memcached) *api.Memcached {
			in.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
			return in
		})
		Expect(err).NotTo(HaveOccurred())

		By("Delete memcached")
		err = f.DeleteMemcached(memcached.ObjectMeta)
		if err != nil && kerr.IsNotFound(err) {
			// Memcached was not created. Hence, rest of cleanup is not necessary.
			return
		}
		Expect(err).NotTo(HaveOccurred())

		By("Wait for memcached to be deleted")
		f.EventuallyMemcached(memcached.ObjectMeta).Should(BeFalse())

		By("Wait for memcached resources to be wipedOut")
		f.EventuallyWipedOut(memcached.ObjectMeta).Should(Succeed())
	})

	var createAndWaitForRunning = func() {
		By("Create Memcached: " + memcached.Name)
		err = f.CreateMemcached(memcached)
		Expect(err).NotTo(HaveOccurred())

		By("Wait for Running memcached")
		f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())

		By("Wait for AppBinding to create")
		f.EventuallyAppBinding(memcached.ObjectMeta).Should(BeTrue())

		By("Check valid AppBinding Specs")
		err := f.CheckAppBindingSpec(memcached.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	}

	Describe("Test", func() {

		Context("General", func() {
			var (
				key   string
				value string
			)
			BeforeEach(func() {
				key = rand.WithUniqSuffix("kubed-e2e")
				value = rand.GenerateTokenWithLength(10)
			})

			Context("-", func() {
				It("should run successfully", func() {
					createAndWaitForRunning()

					By("Inserting item into database")
					f.EventuallySetItem(memcached.ObjectMeta, key, value).Should(BeTrue())

					By("Retrieving item from database")
					f.EventuallyGetItem(memcached.ObjectMeta, key).Should(BeEquivalentTo(value))
				})
			})

			Context("Multiple Replica", func() {
				BeforeEach(func() {
					memcached.Spec.Replicas = new(int32)
					*memcached.Spec.Replicas = 3
				})

				It("should run successfully", func() {
					createAndWaitForRunning()

					By("Inserting item into database")
					f.EventuallySetItem(memcached.ObjectMeta, key, value).Should(BeTrue())

					By("Retrieving item from database")
					f.EventuallyGetItem(memcached.ObjectMeta, key).Should(BeEquivalentTo(value))
				})
			})

			Context("with custom SA Name", func() {
				BeforeEach(func() {
					memcached.Spec.PodTemplate.Spec.ServiceAccountName = "my-custom-sa"
					memcached.Spec.TerminationPolicy = api.TerminationPolicyHalt
				})

				It("should start and resume successfully", func() {
					//shouldTakeSnapshot()
					createAndWaitForRunning()
					By("Check if Memcached " + memcached.Name + " exists.")
					_, err := f.GetMemcached(memcached.ObjectMeta)
					if err != nil {
						if kerr.IsNotFound(err) {
							// Memcached was not created. Hence, rest of cleanup is not necessary.
							return
						}
						Expect(err).NotTo(HaveOccurred())
					}

					By("Delete memcached: " + memcached.Name)
					err = f.DeleteMemcached(memcached.ObjectMeta)
					if err != nil {
						if kerr.IsNotFound(err) {
							// Memcached was not created. Hence, rest of cleanup is not necessary.
							log.Infof("Skipping rest of cleanup. Reason: Memcached %s is not found.", memcached.Name)
							return
						}
						Expect(err).NotTo(HaveOccurred())
					}

					By("Wait for memcached to be deleted")
					f.EventuallyMemcached(memcached.ObjectMeta).Should(BeFalse())

					By("Resume DB")
					createAndWaitForRunning()
				})
			})
		})

		Context("For Custom Resources", func() {

			Context("with custom SA", func() {
				var customSAForDB *core.ServiceAccount
				var customRoleForDB *rbac.Role
				var customRoleBindingForDB *rbac.RoleBinding
				BeforeEach(func() {
					customSAForDB = f.ServiceAccount()
					memcached.Spec.PodTemplate.Spec.ServiceAccountName = customSAForDB.Name
					customRoleForDB = f.RoleForElasticsearch(memcached.ObjectMeta)
					customRoleBindingForDB = f.RoleBinding(customSAForDB.Name, customRoleForDB.Name)
				})
				It("should and Run DB successfully", func() {
					By("Create Database SA")
					err = f.CreateServiceAccount(customSAForDB)
					Expect(err).NotTo(HaveOccurred())
					By("Create Database Role")
					err = f.CreateRole(customRoleForDB)
					Expect(err).NotTo(HaveOccurred())
					By("Create Database RoleBinding")
					err = f.CreateRoleBinding(customRoleBindingForDB)
					Expect(err).NotTo(HaveOccurred())
					createAndWaitForRunning()
				})
			})
		})

		Context("PDB", func() {
			It("should evict successfully", func() {
				// Create Memcached
				memcached.Spec.Replicas = types.Int32P(3)
				createAndWaitForRunning()
				//Evict Memcached pod
				By("Try to evict a pod")
				err := f.EvictPodsFromDeployment(memcached.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Resume", func() {

			Context("Super Fast User - Create-Delete-Create-Delete-Create ", func() {
				It("should resume DormantDatabase successfully", func() {
					// Create and wait for running Memcached
					createAndWaitForRunning()

					By("Delete memcached")
					err = f.DeleteMemcached(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for memcached to be deleted")
					f.EventuallyMemcached(memcached.ObjectMeta).Should(BeFalse())

					// Create Memcached object again to resume it
					By("Create Memcached: " + memcached.Name)
					err = f.CreateMemcached(memcached)
					Expect(err).NotTo(HaveOccurred())

					// Delete without caring if DB is resumed
					By("Delete memcached")
					err = f.DeleteMemcached(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("wait fot Memcached to be deleted")
					f.EventuallyMemcached(memcached.ObjectMeta).Should(BeFalse())

					// Create Memcached object again to resume it
					By("Create Memcached: " + memcached.Name)
					err = f.CreateMemcached(memcached)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for Running memcached")
					f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())

					_, err = f.GetMemcached(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("-", func() {
				It("should resume DormantDatabase successfully", func() {
					// Create and wait for running Memcached
					createAndWaitForRunning()
					By("Delete memcached")
					err := f.DeleteMemcached(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for memcached to be deleted")
					f.EventuallyMemcached(memcached.ObjectMeta).Should(BeFalse())

					// Create Memcached object again to resume it
					By("Create Memcached: " + memcached.Name)
					err = f.CreateMemcached(memcached)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for Running memcached")
					f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())

					_, err = f.GetMemcached(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

				})
			})

			Context("Multiple times", func() {
				It("should resume DormantDatabase successfully", func() {
					// Create and wait for running Memcached
					createAndWaitForRunning()

					for i := 0; i < 3; i++ {
						By(fmt.Sprintf("%v-th", i+1) + " time running.")
						By("Delete memcached")
						err := f.DeleteMemcached(memcached.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Wait for memcached to be deleted")
						f.EventuallyMemcached(memcached.ObjectMeta).Should(BeFalse())

						// Create Memcached object again to resume it
						By("Create Memcached: " + memcached.Name)
						err = f.CreateMemcached(memcached)
						Expect(err).NotTo(HaveOccurred())

						By("Wait for Running memcached")
						f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())

						_, err = f.GetMemcached(memcached.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())
					}
				})
			})
		})

		Context("Termination Policy", func() {
			var (
				key   string
				value string
			)
			BeforeEach(func() {
				key = rand.WithUniqSuffix("kubed-e2e")
				value = rand.GenerateTokenWithLength(10)
			})

			var shouldRunWithTermination = func() {
				// Create and wait for running Memcached
				createAndWaitForRunning()

				By("Inserting item into database")
				f.EventuallySetItem(memcached.ObjectMeta, key, value).Should(BeTrue())

				By("Retrieving item from database")
				f.EventuallyGetItem(memcached.ObjectMeta, key).Should(BeEquivalentTo(value))

			}

			Context("with TerminationPolicyDoNotTerminate", func() {
				BeforeEach(func() {
					memcached.Spec.TerminationPolicy = api.TerminationPolicyDoNotTerminate
				})

				It("should work successfully", func() {
					// Create and wait for running Memcached
					createAndWaitForRunning()

					By("Delete memcached")
					err = f.DeleteMemcached(memcached.ObjectMeta)
					Expect(err).Should(HaveOccurred())

					By("Memcached is not paused. Check for memcached")
					f.EventuallyMemcached(memcached.ObjectMeta).Should(BeTrue())

					By("Check for Running memcached")
					f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())

					By("Update memcached to set spec.terminationPolicy = Halt")
					_, err := f.PatchMemcached(memcached.ObjectMeta, func(in *api.Memcached) *api.Memcached {
						in.Spec.TerminationPolicy = api.TerminationPolicyHalt
						return in
					})
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with TerminationPolicyHalt)", func() {
				var shouldRunWithTerminationHalt = func() {
					shouldRunWithTermination()

					By("Halt Memcached: Update memcached to set spec.halted = true")
					_, err := f.PatchMemcached(memcached.ObjectMeta, func(in *api.Memcached) *api.Memcached {
						in.Spec.Halted = true
						return in
					})
					Expect(err).NotTo(HaveOccurred())

					By("Wait for halted/paused memcached")
					f.EventuallyMemcachedPhase(memcached.ObjectMeta).Should(Equal(api.DatabasePhaseHalted))

					By("Resume Memcached: Update memcached to set spec.halted = false")
					_, err = f.PatchMemcached(memcached.ObjectMeta, func(in *api.Memcached) *api.Memcached {
						in.Spec.Halted = false
						return in
					})
					Expect(err).NotTo(HaveOccurred())

					By("Wait for Running memcached")
					f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())

					By("Delete memcached")
					err = f.DeleteMemcached(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for memcached to be deleted")
					f.EventuallyMemcached(memcached.ObjectMeta).Should(BeFalse())

					// Create Memcached object again to resume it
					By("Create (pause) Memcached: " + memcached.Name)
					err = f.CreateMemcached(memcached)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for Running memcached")
					f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())

					memcached, err = f.GetMemcached(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Inserting item into database")
					f.EventuallySetItem(memcached.ObjectMeta, key, value).Should(BeTrue())

					By("Retrieving item from database")
					f.EventuallyGetItem(memcached.ObjectMeta, key).Should(BeEquivalentTo(value))
				}

				It("should create dormantdatabase successfully", shouldRunWithTerminationHalt)
			})

			Context("with TerminationPolicyDelete", func() {
				BeforeEach(func() {
					memcached.Spec.TerminationPolicy = api.TerminationPolicyDelete
				})

				var shouldRunWithTerminationDelete = func() {
					shouldRunWithTermination()

					By("Delete memcached")
					err = f.DeleteMemcached(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("wait until memcached is deleted")
					f.EventuallyMemcached(memcached.ObjectMeta).Should(BeFalse())
				}

				It("should run with TerminationPolicyDelete", shouldRunWithTerminationDelete)
			})

			Context("with TerminationPolicyWipeOut", func() {
				BeforeEach(func() {
					memcached.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
				})

				var shouldRunWithTerminationWipeOut = func() {
					shouldRunWithTermination()

					By("Delete memcached")
					err = f.DeleteMemcached(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("wait until memcached is deleted")
					f.EventuallyMemcached(memcached.ObjectMeta).Should(BeFalse())
				}

				It("should run with TerminationPolicyDelete", shouldRunWithTerminationWipeOut)
			})
		})

		Context("Environment Variables", func() {
			envList := []core.EnvVar{
				{
					Name:  "TEST_ENV",
					Value: "kubedb-memcached-e2e",
				},
			}

			Context("Allowed Envs", func() {
				It("should run successfully with given Env", func() {
					memcached.Spec.PodTemplate.Spec.Env = envList
					createAndWaitForRunning()

					By("Checking pod started with given envs")
					pod, err := f.GetPod(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					out, err := exec_util.ExecIntoPod(f.RestConfig(), pod, exec_util.Command("env"))
					Expect(err).NotTo(HaveOccurred())
					for _, env := range envList {
						Expect(out).Should(ContainSubstring(env.Name + "=" + env.Value))
					}

				})
			})

			Context("Update Envs", func() {
				It("should not reject to update Env", func() {
					memcached.Spec.PodTemplate.Spec.Env = envList
					createAndWaitForRunning()

					By("Updating Envs")
					_, _, err := util.PatchMemcached(f.DBClient().KubedbV1alpha1(), memcached, func(in *api.Memcached) *api.Memcached {
						in.Spec.PodTemplate.Spec.Env = []core.EnvVar{
							{
								Name:  "TEST_ENV",
								Value: "patched",
							},
						}
						return in
					})
					Expect(err).NotTo(HaveOccurred())
				})
			})

		})

		Context("Custom config", func() {

			customConfigs := []framework.MemcdConfig{
				{
					Name:  "conn-limit",
					Value: "510",
					Alias: "max_connections",
				},
				{
					Name:  "memory-limit",
					Value: "128", // MB
					Alias: "limit_maxbytes",
				},
			}

			Context("from configMap", func() {
				var (
					userConfig *core.ConfigMap
				)

				BeforeEach(func() {
					userConfig = f.GetCustomConfig(customConfigs)
				})

				AfterEach(func() {
					By("Deleting configMap: " + userConfig.Name)
					err := f.DeleteConfigMap(userConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should set configuration provided in configMap", func() {
					if skipMessage != "" {
						Skip(skipMessage)
					}

					By("Creating configMap: " + userConfig.Name)
					err := f.CreateConfigMap(userConfig)
					Expect(err).NotTo(HaveOccurred())

					memcached.Spec.ConfigSource = &core.VolumeSource{
						ConfigMap: &core.ConfigMapVolumeSource{
							LocalObjectReference: core.LocalObjectReference{
								Name: userConfig.Name,
							},
						},
					}

					// Create Memcached
					createAndWaitForRunning()

					By("Checking database pod has mounted configSource volume")
					f.EventuallyConfigSourceVolumeMounted(memcached.ObjectMeta).Should(BeTrue())

					// TODO
					// currently the memcached go client we have used, does not have Stats() method to get runtime configuration
					// however, there is pending PR that add this method. when the PR will merge, we can complete the code bellow.
					//By("Checking Memcached configured from provided custom configuration")
					//for _, cfg := range customConfigs {
					//	f.EventuallyMemcachedConfigs(memcached.ObjectMeta).Should(matcher.UseCustomConfig(cfg))
					//}
				})
			})

		})

	})
})
