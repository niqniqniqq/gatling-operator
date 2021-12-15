package controllers

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gatlingv1alpha1 "github.com/st-tech/gatling-operator/api/v1alpha1"
	"github.com/st-tech/gatling-operator/utils"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var _ = Context("Inside of a new namespace", func() {
	ctx := context.TODO()
	ns := SetupTest(ctx)
	gatlingName := "test-gatling"

	Describe("when no existing resources exist", func() {

		It("should create a new Gatling resource with the specified name and a runner Job", func() {
			gatling := &gatlingv1alpha1.Gatling{
				ObjectMeta: metav1.ObjectMeta{
					Name:      gatlingName,
					Namespace: ns.Name,
				},
				Spec: gatlingv1alpha1.GatlingSpec{
					GenerateReport:      false,
					NotifyReport:        false,
					CleanupAfterJobDone: false,
					TestScenarioSpec: gatlingv1alpha1.TestScenarioSpec{
						SimulationClass: "MyBasicSimulation",
					},
				},
			}
			err := k8sClient.Create(ctx, gatling)
			Expect(err).NotTo(HaveOccurred(), "failed to create test Gatling resource")

			job := &batchv1.Job{}
			Eventually(func() error {
				return k8sClient.Get(
					ctx, client.ObjectKey{Namespace: ns.Name, Name: gatlingName + "-runner"}, job)
			}).Should(Succeed())
			//fmt.Printf("parallelism = %d", *job.Spec.Parallelism)

			Expect(job.Spec.Parallelism).Should(Equal(pointer.Int32Ptr(1)))
			Expect(job.Spec.Completions).Should(Equal(pointer.Int32Ptr(1)))
		})

		It("should create a new Gatling resource with the specified name and a runner Job with 2 parallelism", func() {
			gatling := &gatlingv1alpha1.Gatling{
				ObjectMeta: metav1.ObjectMeta{
					Name:      gatlingName,
					Namespace: ns.Name,
				},
				Spec: gatlingv1alpha1.GatlingSpec{
					GenerateReport:      false,
					NotifyReport:        false,
					CleanupAfterJobDone: false,
					TestScenarioSpec: gatlingv1alpha1.TestScenarioSpec{
						SimulationClass: "MyBasicSimulation",
						Parallelism:     2,
					},
				},
			}
			err := k8sClient.Create(ctx, gatling)
			Expect(err).NotTo(HaveOccurred(), "failed to create test Gatling resource")

			job := &batchv1.Job{}
			Eventually(func() error {
				return k8sClient.Get(
					ctx, client.ObjectKey{Namespace: ns.Name, Name: gatlingName + "-runner"}, job)
			}).Should(Succeed())
			fmt.Printf("parallelism = %d", *job.Spec.Parallelism)

			Expect(job.Spec.Parallelism).Should(Equal(pointer.Int32Ptr(2)))
			Expect(job.Spec.Completions).Should(Equal(pointer.Int32Ptr(2)))
		})

	})
})

var _ = Describe("Test gatlingNotificationReconcile", func() {
	namespace := "test-namespace"
	gatlingName := "test-gatling"
	gatlingReconcilerImplMock := utils.NewMockGatlingReconcilerImpl()
	client := utils.NewClient()
	scheme := newTestScheme()
	reconciler := &GatlingReconciler{Client: client, Scheme: scheme, GatlingReconcilerInterface: gatlingReconcilerImplMock}
	ctx := context.TODO()
	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      gatlingName,
		},
	}
	gatling := &gatlingv1alpha1.Gatling{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatlingName,
			Namespace: namespace,
		},
		Spec: gatlingv1alpha1.GatlingSpec{
			GenerateReport:      true,
			NotifyReport:        false,
			CleanupAfterJobDone: false,
			TestScenarioSpec: gatlingv1alpha1.TestScenarioSpec{
				SimulationClass: "MyBasicSimulation",
				Parallelism:     1,
				SimulationData:  map[string]string{"testData": "test"},
			},
		},
	}
	Context("gatling.spec.generateReport is true && getCloudStorageInfo return error", func() {
		gatlingReconcilerImplMock.On("GetCloudStorageInfo",
			mock.IsType(ctx),
			mock.Anything,
			mock.Anything,
		).Return("", "", fmt.Errorf("error mock getCloudStorageInfo")).Once()
		reconciliationResult, err := reconciler.gatlingNotificationReconcile(ctx, request, gatling, log.FromContext(ctx))
		Expect(err).To(HaveOccurred())
		Expect(reconciliationResult).To(Equal(true))
	})
	Context("gatling.spec.generateReport is true && getCloudStorageInfo return url", func() {
		It("sendNotification return error", func() {
			gatlingReconcilerImplMock.On("GetCloudStorageInfo",
				mock.IsType(ctx),
				mock.Anything,
				mock.Anything,
			).Return("", "test_url", nil)
			gatlingReconcilerImplMock.On("SendNotification",
				mock.IsType(ctx),
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return(fmt.Errorf("error mock sendNotification")).Once()
			reconciliationResult, err := reconciler.gatlingNotificationReconcile(ctx, request, gatling, log.FromContext(ctx))
			Expect(err).To(HaveOccurred())
			Expect(reconciliationResult).To(Equal(true))
		})
		It("sendNotification return nil && updateGatlingStatus return error", func() {
			gatlingReconcilerImplMock.On("SendNotification",
				mock.IsType(ctx),
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return(nil)
			gatlingReconcilerImplMock.On("UpdateGatlingStatus",
				mock.IsType(ctx),
				mock.Anything,
				mock.Anything,
			).Return(fmt.Errorf("error mock updateGatlingStatus")).Once()
			reconciliationResult, err := reconciler.gatlingNotificationReconcile(ctx, request, gatling, log.FromContext(ctx))
			Expect(err).To(HaveOccurred())
			Expect(reconciliationResult).To(Equal(true))
		})
		It("sendNotification return nil && updateGatlingStatus return nil", func() {
			gatlingReconcilerImplMock.On("UpdateGatlingStatus",
				mock.IsType(ctx),
				mock.Anything,
				mock.Anything,
			).Return(nil)
			reconciliationResult, err := reconciler.gatlingNotificationReconcile(ctx, request, gatling, log.FromContext(ctx))
			Expect(err).NotTo(HaveOccurred())
			Expect(reconciliationResult).To(Equal(true))
		})
	})
})

var _ = Describe("Test GetCloudStorageInfo", func() {

})

var _ = Describe("Test GetCloudStorageProvider", func() {
	client := utils.NewClient()
	scheme := newTestScheme()
	reconciler := &GatlingReconciler{Client: client, Scheme: scheme, GatlingReconcilerInterface: &GatlingReconcilerInterfaceImpl{}}
	gatling := &gatlingv1alpha1.Gatling{
		Spec: gatlingv1alpha1.GatlingSpec{
			CloudStorageSpec: gatlingv1alpha1.CloudStorageSpec{
				Provider: "",
			},
		},
	}
	It("provicer is empty", func() {
		resultProvider := reconciler.GetCloudStorageProvider(gatling)
		Expect(resultProvider).To(Equal(""))
	})
	It("provicer is aws", func() {
		gatling.Spec.CloudStorageSpec.Provider = "aws"
		resultProvider := reconciler.GetCloudStorageProvider(gatling)
		Expect(resultProvider).To(Equal("aws"))
	})
})

func newTestScheme() *runtime.Scheme {
	testScheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(testScheme)
	return testScheme
}
