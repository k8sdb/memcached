package e2e_test

import (
	"fmt"

	"github.com/appscode/go/crypto/rand"
	exec_util "github.com/appscode/kutil/tools/exec"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"
	"github.com/kubedb/memcached/test/e2e/framework"
	"github.com/kubedb/memcached/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
)

var _ = Describe("Memcached", func() {
	var (
		err              error
		f                *framework.Invocation
		memcached        *api.Memcached
		memcachedVersion *api.MemcachedVersion
		skipMessage      string
		testSvc          *core.Service
	)

	BeforeEach(func() {
		f = root.Invoke()
		memcached = f.Memcached()
		memcachedVersion = f.MemcachedVersion()
		skipMessage = ""
	})

	JustBeforeEach(func() {
		if skipMessage != "" {
			Skip(skipMessage)
		}

		testSvc = f.GetTestService(memcached.ObjectMeta)

		By("Creating Service: " + testSvc.Name)
		f.CreateService(testSvc)
	})

	AfterEach(func() {
		By("Delete memcached")
		err = f.DeleteMemcached(memcached.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		By("Wait for memcached to be paused")
		f.EventuallyDormantDatabaseStatus(memcached.ObjectMeta).Should(matcher.HavePaused())

		By("WipeOut memcached")
		_, err := f.PatchDormantDatabase(memcached.ObjectMeta, func(in *api.DormantDatabase) *api.DormantDatabase {
			in.Spec.WipeOut = true
			return in
		})
		Expect(err).NotTo(HaveOccurred())

		By("Delete Dormant Database")
		err = f.DeleteDormantDatabase(memcached.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		By("Wait for memcached resources to be wipedOut")
		f.EventuallyWipedOut(memcached.ObjectMeta).Should(Succeed())

		By("Deleting Service: " + testSvc.Name)
		f.DeleteService(testSvc.ObjectMeta)

		err = f.DeleteMemcachedVersion(memcachedVersion.ObjectMeta)
		if err != nil && !kerr.IsNotFound(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	})

	var createAndWaitForRunning = func() {
		By("Create MemcachedVersion: " + memcachedVersion.Name)
		err = f.CreateMemcachedVersion(memcachedVersion)
		Expect(err).NotTo(HaveOccurred())

		By("Create Memcached: " + memcached.Name)
		err = f.CreateMemcached(memcached)
		Expect(err).NotTo(HaveOccurred())

		By("Wait for Running memcached")
		f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())
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

		})

		Context("DoNotPause", func() {
			BeforeEach(func() {
				memcached.Spec.DoNotPause = true
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

				By("Update memcached to set DoNotPause=false")
				f.TryPatchMemcached(memcached.ObjectMeta, func(in *api.Memcached) *api.Memcached {
					in.Spec.DoNotPause = false
					return in
				})
			})
		})

		Context("Resume", func() {
			var usedInitSpec bool
			BeforeEach(func() {
				usedInitSpec = false
			})
			Context("Super Fast User - Create-Delete-Create-Delete-Create ", func() {
				It("should resume DormantDatabase successfully", func() {
					// Create and wait for running Memcached
					createAndWaitForRunning()

					By("Delete memcached")
					err = f.DeleteMemcached(memcached.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for memcached to be paused")
					f.EventuallyDormantDatabaseStatus(memcached.ObjectMeta).Should(matcher.HavePaused())

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

					By("Wait for DormantDatabase to be deleted")
					f.EventuallyDormantDatabase(memcached.ObjectMeta).Should(BeFalse())

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
					f.DeleteMemcached(memcached.ObjectMeta)

					By("Wait for memcached to be paused")
					f.EventuallyDormantDatabaseStatus(memcached.ObjectMeta).Should(matcher.HavePaused())

					// Create Memcached object again to resume it
					By("Create Memcached: " + memcached.Name)
					err = f.CreateMemcached(memcached)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for DormantDatabase to be deleted")
					f.EventuallyDormantDatabase(memcached.ObjectMeta).Should(BeFalse())

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
						f.DeleteMemcached(memcached.ObjectMeta)

						By("Wait for memcached to be paused")
						f.EventuallyDormantDatabaseStatus(memcached.ObjectMeta).Should(matcher.HavePaused())

						// Create Memcached object again to resume it
						By("Create Memcached: " + memcached.Name)
						err = f.CreateMemcached(memcached)
						Expect(err).NotTo(HaveOccurred())

						By("Wait for DormantDatabase to be deleted")
						f.EventuallyDormantDatabase(memcached.ObjectMeta).Should(BeFalse())

						By("Wait for Running memcached")
						f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())

						_, err := f.GetMemcached(memcached.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())
					}
				})
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

					out, err := exec_util.ExecIntoPod(f.RestConfig(), pod, "env")
					Expect(err).NotTo(HaveOccurred())
					for _, env := range envList {
						Expect(out).Should(ContainSubstring(env.Name + "=" + env.Value))
					}

				})
			})

			Context("Update Envs", func() {
				It("should reject to update Env", func() {
					memcached.Spec.PodTemplate.Spec.Env = envList
					createAndWaitForRunning()

					By("Updating Envs")
					_, _, err := util.PatchMemcached(f.ExtClient(), memcached, func(in *api.Memcached) *api.Memcached {
						in.Spec.PodTemplate.Spec.Env = []core.EnvVar{
							{
								Name:  "TEST_ENV",
								Value: "patched",
							},
						}
						return in
					})

					Expect(err).To(HaveOccurred())
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
					f.DeleteConfigMap(userConfig.ObjectMeta)
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
