package e2e_test

import (
	"fmt"

	exec_util "github.com/appscode/kutil/tools/exec"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"
	"github.com/kubedb/memcached/test/e2e/framework"
	"github.com/kubedb/memcached/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
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

	var createAndWaitForRunning = func() {
		By("Create Memcached: " + memcached.Name)
		err = f.CreateMemcached(memcached)
		Expect(err).NotTo(HaveOccurred())

		By("Wait for Running memcached")
		f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())
	}

	var deleteTestResource = func() {
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
	}

	var shouldSuccessfullyRunning = func() {
		if skipMessage != "" {
			Skip(skipMessage)
		}
		// Create Memcached
		createAndWaitForRunning()

		// Delete test resource
		deleteTestResource()
	}

	Describe("Test", func() {

		Context("General", func() {

			Context("-", func() {
				It("should run successfully", shouldSuccessfullyRunning)
			})
			Context("Multiple Replica", func() {
				BeforeEach(func() {
					memcached.Spec.Replicas = new(int32)
					*memcached.Spec.Replicas = 3
				})
				It("should run successfully", shouldSuccessfullyRunning)
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

				// Delete test resource
				deleteTestResource()
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

					// Delete test resource
					deleteTestResource()
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

					// Delete test resource
					deleteTestResource()
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

					// Delete test resource
					deleteTestResource()
				})
			})
		})

		Context("Environment Variables", func() {
			AfterEach(func() {
				deleteTestResource()
			})
			envList := []core.EnvVar{
				{
					Name:  "TEST_ENV",
					Value: "kubedb-memcached-e2e",
				},
			}

			Context("Allowed Envs", func() {
				It("should run successfully with given Env", func() {
					memcached.Spec.Env = envList
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
					memcached.Spec.Env = envList
					createAndWaitForRunning()

					By("Updating Envs")
					_, _, err := util.PatchMemcached(f.ExtClient(), memcached, func(in *api.Memcached) *api.Memcached {
						in.Spec.Env = []core.EnvVar{
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

	})
})
