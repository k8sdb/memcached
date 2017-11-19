package e2e_test

import (
	"fmt"

	api "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/k8sdb/memcached/test/e2e/framework"
	"github.com/k8sdb/memcached/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		_, err := f.TryPatchDormantDatabase(memcached.ObjectMeta, func(in *api.DormantDatabase) *api.DormantDatabase {
			in.Spec.WipeOut = true
			return in
		})
		Expect(err).NotTo(HaveOccurred())

		By("Wait for memcached to be wipedOut")
		f.EventuallyDormantDatabaseStatus(memcached.ObjectMeta).Should(matcher.HaveWipedOut())

		err = f.DeleteDormantDatabase(memcached.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
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
					memcached.Spec.Replicas = 3
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
				Expect(err).NotTo(HaveOccurred())

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

			var shouldResumeSuccessfully = func() {
				// Create and wait for running Memcached
				createAndWaitForRunning()

				By("Delete memcached")
				f.DeleteMemcached(memcached.ObjectMeta)

				By("Wait for memcached to be paused")
				f.EventuallyDormantDatabaseStatus(memcached.ObjectMeta).Should(matcher.HavePaused())

				_, err = f.TryPatchDormantDatabase(memcached.ObjectMeta, func(in *api.DormantDatabase) *api.DormantDatabase {
					in.Spec.Resume = true
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Wait for DormantDatabase to be deleted")
				f.EventuallyDormantDatabase(memcached.ObjectMeta).Should(BeFalse())

				By("Wait for Running memcached")
				f.EventuallyMemcachedRunning(memcached.ObjectMeta).Should(BeTrue())

				memcached, err = f.GetMemcached(memcached.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				// Delete test resource
				deleteTestResource()
			}

			Context("-", func() {
				It("should resume DormantDatabase successfully", shouldResumeSuccessfully)
			})

			Context("With original Memcached", func() {
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
		})

	})
})
