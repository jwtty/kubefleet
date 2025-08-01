/*
Copyright 2025 The KubeFleet Authors.

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
package clusterresourcebindingwatcher

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	fleetv1beta1 "github.com/kubefleet-dev/kubefleet/apis/placement/v1beta1"
	"github.com/kubefleet-dev/kubefleet/pkg/utils/controller"
)

const (
	testPlacementName                = "test-placement"
	testCRBName                      = "test-crb"
	testRBName                       = "test-rb"
	testNamespace                    = "test-ns"
	testResourceSnapshotName         = "test-rs"
	testSchedulingPolicySnapshotName = "test-sps"
	testTargetCluster                = "test-cluster"
	testReason1                      = "testReason1"
	testReason2                      = "testReason2"

	eventuallyTimeout    = time.Second * 10
	consistentlyDuration = time.Second * 10
	interval             = time.Millisecond * 250
)

// This container cannot be run in parallel with other ITs because it uses a shared fakePlacementController.
var _ = Describe("Test ClusterResourceBinding Watcher - create, delete events", Serial, func() {
	var crb *fleetv1beta1.ClusterResourceBinding
	It("When creating, deleting clusterResourceBinding", func() {
		fakePlacementController.ResetQueue()
		By("Creating a new clusterResourceBinding")
		crb = clusterResourceBindingForTest()
		Expect(k8sClient.Create(ctx, crb)).Should(Succeed(), "failed to create cluster resource binding")

		By("Checking placement controller queue")
		consistentlyCheckPlacementControllerQueueIsEmpty()

		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")

		By("Deleting clusterResourceBinding")
		Expect(k8sClient.Delete(ctx, crb)).Should(Succeed(), "failed to delete cluster resource binding")

		By("Checking placement controller queue")
		consistentlyCheckPlacementControllerQueueIsEmpty()
	})
})

// This container cannot be run in parallel with other ITs because it uses a shared fakePlacementController.
var _ = Describe("Test ResourceBinding Watcher - create, delete events", Serial, Ordered, func() {
	var rb *fleetv1beta1.ResourceBinding

	BeforeEach(func() {
		fakePlacementController.ResetQueue()
	})

	It("When creating, deleting resourceBinding", func() {
		By("Creating a new resourceBinding")
		rb = resourceBindingForTest()
		Expect(k8sClient.Create(ctx, rb)).Should(Succeed(), "failed to create resource binding")

		By("Checking placement controller queue")
		consistentlyCheckPlacementControllerQueueIsEmpty()

		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testRBName, Namespace: testNamespace}, rb)).Should(Succeed(), "failed to get resource binding")

		By("Deleting resourceBinding")
		Expect(k8sClient.Delete(ctx, rb)).Should(Succeed(), "failed to delete resource binding")

		By("Checking placement controller queue")
		consistentlyCheckPlacementControllerQueueIsEmpty()
	})
})

// This container cannot be run in parallel with other ITs because it uses a shared fakePlacementController. These tests are also ordered.
var _ = Describe("Test ResourceBinding Watcher - update status", Serial, Ordered, func() {
	var rb *fleetv1beta1.ResourceBinding

	BeforeEach(func() {
		fakePlacementController.ResetQueue()
		By("Creating a new resourceBinding")
		rb = resourceBindingForTest()
		Expect(k8sClient.Create(ctx, rb)).Should(Succeed(), "failed to create resource binding")
		fakePlacementController.ResetQueue()
	})

	AfterEach(func() {
		rb.Name = testRBName
		rb.Namespace = testNamespace
		By("Deleting the resourceBinding")
		Expect(k8sClient.Delete(ctx, rb)).Should(Succeed(), "failed to delete resource binding")
	})

	It("Should enqueue the resourcePlacement name for reconciling, when resourceBinding status changes - RolloutStarted", func() {
		validateWhenUpdateResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingRolloutStarted, rb.Generation, metav1.ConditionTrue, testReason1)
		validateWhenUpdateResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingRolloutStarted, rb.Generation, metav1.ConditionFalse, testReason1)
	})
})

// This container cannot be run in parallel with other ITs because it uses a shared fakePlacementController.
var _ = Describe("Test ClusterResourceBinding Watcher - update metadata", Serial, func() {
	var crb *fleetv1beta1.ClusterResourceBinding
	BeforeEach(func() {
		fakePlacementController.ResetQueue()
		By("Creating a new clusterResourceBinding")
		crb = clusterResourceBindingForTest()
		Expect(k8sClient.Create(ctx, crb)).Should(Succeed(), "failed to create cluster resource binding")
		fakePlacementController.ResetQueue()
	})

	AfterEach(func() {
		By("Deleting the clusterResourceBinding")
		Expect(k8sClient.Delete(ctx, crb)).Should(Succeed(), "failed to delete cluster resource binding")
	})

	It("Should not enqueue the clusterResourcePlacement name for reconciling, when only meta data changed", func() {
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
		labels := crb.GetLabels()
		labels["test-key"] = "test-value"
		crb.SetLabels(labels)
		Expect(k8sClient.Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding")

		By("Checking placement controller queue")
		consistentlyCheckPlacementControllerQueueIsEmpty()
	})

	It("Should not enqueue the clusterResourcePlacement name for reconciling, when only spec changed", func() {
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
		crb.Spec.State = fleetv1beta1.BindingStateBound
		Expect(k8sClient.Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding")

		By("Checking placement controller queue")
		consistentlyCheckPlacementControllerQueueIsEmpty()
	})
})

// This container cannot be run in parallel with other ITs because it uses a shared fakePlacementController. These tests are also ordered.
var _ = Describe("Test ClusterResourceBinding Watcher - update status", Serial, Ordered, func() {
	var crb *fleetv1beta1.ClusterResourceBinding
	var currentTime metav1.Time
	BeforeAll(func() {
		currentTime = metav1.Now()
		fakePlacementController.ResetQueue()
		By("Creating a new clusterResourceBinding")
		crb = clusterResourceBindingForTest()
		Expect(k8sClient.Create(ctx, crb)).Should(Succeed(), "failed to create cluster resource binding")
		fakePlacementController.ResetQueue()
	})

	AfterAll(func() {
		crb.Name = testCRBName
		By("Deleting the clusterResourceBinding")
		Expect(k8sClient.Delete(ctx, crb)).Should(Succeed(), "failed to delete cluster resource binding")
	})

	It("Should enqueue the clusterResourcePlacement name for reconciling, when clusterResourceBinding status changes - RolloutStarted", func() {
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingRolloutStarted, crb.Generation, metav1.ConditionTrue, testReason1)
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingRolloutStarted, crb.Generation, metav1.ConditionFalse, testReason1)
	})

	It("Should enqueue the clusterResourcePlacement name for reconciling, when clusterResourceBinding status changes - Overridden", func() {
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingOverridden, crb.Generation, metav1.ConditionTrue, testReason1)
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingOverridden, crb.Generation, metav1.ConditionFalse, testReason1)
	})

	It("Should enqueue the clusterResourcePlacement name for reconciling, when clusterResourceBinding status changes - WorkCreated", func() {
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingWorkSynchronized, crb.Generation, metav1.ConditionTrue, testReason1)
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingWorkSynchronized, crb.Generation, metav1.ConditionFalse, testReason1)
	})

	It("Should enqueue the clusterResourcePlacement name for reconciling, when clusterResourceBinding status changes - Applied", func() {
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingApplied, crb.Generation, metav1.ConditionTrue, testReason1)
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingApplied, crb.Generation, metav1.ConditionFalse, testReason1)
	})

	It("Should enqueue the clusterResourcePlacement name for reconciling, when clusterResourceBinding status changes - Available", func() {
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingAvailable, crb.Generation, metav1.ConditionTrue, testReason1)
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingAvailable, crb.Generation, metav1.ConditionFalse, testReason1)
	})

	It("Should enqueue the clusterResourcePlacement name for reconciling, when condition's reason changes", func() {
		validateWhenUpdateClusterResourceBindingStatusWithCondition(fleetv1beta1.ResourceBindingOverridden, crb.Generation, metav1.ConditionFalse, testReason2)
	})

	It("Should enqueue the clusterResourcePlacement name for reconciling, when condition's observed generation changes", func() {
		crb := &fleetv1beta1.ClusterResourceBinding{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
		condition := metav1.Condition{
			Type:               string(fleetv1beta1.ResourceBindingOverridden),
			ObservedGeneration: crb.Generation + 1,
			Status:             metav1.ConditionFalse,
			Reason:             testReason2,
			LastTransitionTime: currentTime,
		}
		By(fmt.Sprintf("Updating the clusterResourceBinding status - %s, %d, %s, %s", fleetv1beta1.ResourceBindingOverridden, crb.Generation, metav1.ConditionFalse, testReason2))
		crb.SetConditions(condition)
		Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

		By("Checking placement controller queue")
		eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
		fakePlacementController.ResetQueue()
	})

	It("Should not enqueue the clusterResourcePlacement name for reconciling, when only condition's last transition time changes", func() {
		crb := &fleetv1beta1.ClusterResourceBinding{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
		newTime := metav1.NewTime(currentTime.Add(10 * time.Second))
		condition := metav1.Condition{
			Type:               string(fleetv1beta1.ResourceBindingOverridden),
			ObservedGeneration: crb.Generation,
			Status:             metav1.ConditionFalse,
			Reason:             testReason2,
			LastTransitionTime: newTime,
		}
		By(fmt.Sprintf("Updating the clusterResourceBinding status - %s, %d, %s, %s", fleetv1beta1.ResourceBindingOverridden, crb.Generation, metav1.ConditionFalse, testReason2))
		crb.SetConditions(condition)
		Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

		consistentlyCheckPlacementControllerQueueIsEmpty()
	})

	Context("Should enqueue the clusterResourcePlacement name for reconciling, when the failed placement list has changed", Serial, Ordered, func() {
		It("Should enqueue the clusterResourcePlacement name for reconciling, when there are new failed placements", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.FailedPlacements = []fleetv1beta1.FailedResourcePlacement{
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "Service",
						Name:      "svc-name",
						Namespace: "svc-namespace",
					},
					Condition: metav1.Condition{
						Type:               fleetv1beta1.WorkConditionTypeAvailable,
						Status:             metav1.ConditionFalse,
						Reason:             "fakeFailedAvailableReason",
						Message:            "fakeFailedAvailableMessage",
						LastTransitionTime: metav1.Now(),
					},
				},
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "ConfigMap",
						Name:      "config-name",
						Namespace: "config-namespace",
					},
					Condition: metav1.Condition{
						Type:               fleetv1beta1.WorkConditionTypeAvailable,
						Status:             metav1.ConditionFalse,
						Reason:             "fakeFailedAvailableReason",
						Message:            "fakeFailedAvailableMessage",
						LastTransitionTime: metav1.Now(),
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})

		It("Should enqueue the clusterResourcePlacement name for reconciling, when there are one less failed placements", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.FailedPlacements = []fleetv1beta1.FailedResourcePlacement{
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "Service",
						Name:      "svc-name",
						Namespace: "svc-namespace",
					},
					Condition: metav1.Condition{
						Type:               fleetv1beta1.WorkConditionTypeAvailable,
						Status:             metav1.ConditionFalse,
						Reason:             "fakeFailedAvailableReason",
						Message:            "fakeFailedAvailableMessage",
						LastTransitionTime: metav1.Now(),
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})

		It("Should enqueue the clusterResourcePlacement name for reconciling, when there are no more failed placements", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.FailedPlacements = []fleetv1beta1.FailedResourcePlacement{}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})
	})

	Context("Should enqueue the clusterResourcePlacement name for reconciling, when the drifted placement list has changed", Serial, Ordered, func() {
		startTime := metav1.Now()

		It("Should enqueue the clusterResourcePlacement name for reconciling, when there are new drifted placements", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.DriftedPlacements = []fleetv1beta1.DriftedResourcePlacement{
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "Service",
						Name:      "svc-name",
						Namespace: "svc-namespace",
					},
					ObservationTime:                 startTime,
					TargetClusterObservedGeneration: 1,
					FirstDriftedObservedTime:        startTime,
					ObservedDrifts: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "bar",
							ValueInMember: "baz",
						},
					},
				},
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "ConfigMap",
						Name:      "config-name",
						Namespace: "config-namespace",
					},
					ObservationTime:                 startTime,
					TargetClusterObservedGeneration: 1,
					FirstDriftedObservedTime:        startTime,
					ObservedDrifts: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "qux",
							ValueInMember: "quux",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})

		It("Should not enqueue the clusterResourcePlacement name for reconciling, when there are only order changes in the drifted placement list", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.DriftedPlacements = []fleetv1beta1.DriftedResourcePlacement{
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "ConfigMap",
						Name:      "config-name",
						Namespace: "config-namespace",
					},
					ObservationTime:                 startTime,
					TargetClusterObservedGeneration: 1,
					FirstDriftedObservedTime:        startTime,
					ObservedDrifts: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "qux",
							ValueInMember: "quux",
						},
					},
				},
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "Service",
						Name:      "svc-name",
						Namespace: "svc-namespace",
					},
					ObservationTime:                 startTime,
					TargetClusterObservedGeneration: 1,
					FirstDriftedObservedTime:        startTime,
					ObservedDrifts: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "bar",
							ValueInMember: "baz",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			consistentlyCheckPlacementControllerQueueIsEmpty()
		})

		It("Should enqueue the clusterResourcePlacement name for reconciling, when there are one less drifted placement", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.DriftedPlacements = []fleetv1beta1.DriftedResourcePlacement{
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "Service",
						Name:      "svc-name",
						Namespace: "svc-namespace",
					},
					ObservationTime:                 metav1.Now(),
					TargetClusterObservedGeneration: 1,
					FirstDriftedObservedTime:        metav1.Now(),
					ObservedDrifts: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "bar",
							ValueInMember: "baz",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})

		It("Should enqueue the clusterResourcePlacement name for reconciling, when the drift details change", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.DriftedPlacements = []fleetv1beta1.DriftedResourcePlacement{
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "Service",
						Name:      "svc-name",
						Namespace: "svc-namespace",
					},
					ObservationTime:                 metav1.Now(),
					TargetClusterObservedGeneration: 1,
					FirstDriftedObservedTime:        metav1.Now(),
					ObservedDrifts: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "qux",
							ValueInMember: "quux",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})

		It("Should enqueue the clusterResourcePlacement name for reconciling, when there are no more drifted placements", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.DriftedPlacements = []fleetv1beta1.DriftedResourcePlacement{}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})
	})

	Context("Should enqueue the clusterResourcePlacement name for reconciling, when the diffed placement list has changed", Serial, Ordered, func() {
		startTime := metav1.Now()

		It("Should enqueue the clusterResourcePlacement name for reconciling, when there are new diffed placements", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.DiffedPlacements = []fleetv1beta1.DiffedResourcePlacement{
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "Service",
						Name:      "svc-name",
						Namespace: "svc-namespace",
					},
					ObservationTime:                 startTime,
					TargetClusterObservedGeneration: ptr.To(int64(1)),
					FirstDiffedObservedTime:         startTime,
					ObservedDiffs: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "bar",
							ValueInMember: "baz",
						},
					},
				},
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "ConfigMap",
						Name:      "config-name",
						Namespace: "config-namespace",
					},
					ObservationTime:                 startTime,
					TargetClusterObservedGeneration: ptr.To(int64(1)),
					FirstDiffedObservedTime:         startTime,
					ObservedDiffs: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "qux",
							ValueInMember: "quux",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})

		It("Should not enqueue the clusterResourcePlacement name for reconciling, when there are only order changes in the drifted placement list", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.DiffedPlacements = []fleetv1beta1.DiffedResourcePlacement{
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "ConfigMap",
						Name:      "config-name",
						Namespace: "config-namespace",
					},
					ObservationTime:                 startTime,
					TargetClusterObservedGeneration: ptr.To(int64(1)),
					FirstDiffedObservedTime:         startTime,
					ObservedDiffs: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "qux",
							ValueInMember: "quux",
						},
					},
				},
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "Service",
						Name:      "svc-name",
						Namespace: "svc-namespace",
					},
					ObservationTime:                 startTime,
					TargetClusterObservedGeneration: ptr.To(int64(1)),
					FirstDiffedObservedTime:         startTime,
					ObservedDiffs: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "bar",
							ValueInMember: "baz",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			consistentlyCheckPlacementControllerQueueIsEmpty()
		})

		It("Should enqueue the clusterResourcePlacement name for reconciling, when there are one less diffed placement", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.DiffedPlacements = []fleetv1beta1.DiffedResourcePlacement{
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "ConfigMap",
						Name:      "config-name",
						Namespace: "config-namespace",
					},
					ObservationTime:                 metav1.Now(),
					TargetClusterObservedGeneration: ptr.To(int64(1)),
					FirstDiffedObservedTime:         metav1.Now(),
					ObservedDiffs: []fleetv1beta1.PatchDetail{
						{
							Path:          "/metadata/labels/foo",
							ValueInHub:    "qux",
							ValueInMember: "quux",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})

		It("Should enqueue the clusterResourcePlacement name for reconciling, when the diff details change", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.DiffedPlacements = []fleetv1beta1.DiffedResourcePlacement{
				{
					ResourceIdentifier: fleetv1beta1.ResourceIdentifier{
						Group:     "",
						Version:   "v1",
						Kind:      "ConfigMap",
						Name:      "config-name",
						Namespace: "config-namespace",
					},
					ObservationTime:         metav1.Now(),
					FirstDiffedObservedTime: metav1.Now(),
					ObservedDiffs: []fleetv1beta1.PatchDetail{
						{
							Path:       "/",
							ValueInHub: "(the whole object)",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})

		It("Should enqueue the clusterResourcePlacement name for reconciling, when there are no more drifted placements", func() {
			crb := &fleetv1beta1.ClusterResourceBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
			crb.Status.DiffedPlacements = []fleetv1beta1.DiffedResourcePlacement{}
			Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

			By("Checking placement controller queue")
			eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
			fakePlacementController.ResetQueue()
		})
	})
})

func resourceBindingForTest() *fleetv1beta1.ResourceBinding {
	return &fleetv1beta1.ResourceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRBName,
			Namespace: testNamespace,
			Labels:    map[string]string{fleetv1beta1.PlacementTrackingLabel: testPlacementName},
		},
		Spec: fleetv1beta1.ResourceBindingSpec{
			State:                        fleetv1beta1.BindingStateScheduled,
			ResourceSnapshotName:         testResourceSnapshotName,
			SchedulingPolicySnapshotName: testSchedulingPolicySnapshotName,
			TargetCluster:                testTargetCluster,
		},
	}
}

func clusterResourceBindingForTest() *fleetv1beta1.ClusterResourceBinding {
	return &fleetv1beta1.ClusterResourceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testCRBName,
			Labels: map[string]string{fleetv1beta1.PlacementTrackingLabel: testPlacementName},
		},
		Spec: fleetv1beta1.ResourceBindingSpec{
			State:                        fleetv1beta1.BindingStateScheduled,
			ResourceSnapshotName:         testResourceSnapshotName,
			SchedulingPolicySnapshotName: testSchedulingPolicySnapshotName,
			TargetCluster:                testTargetCluster,
		},
	}
}

func validateWhenUpdateClusterResourceBindingStatusWithCondition(conditionType fleetv1beta1.ResourceBindingConditionType, observedGeneration int64, status metav1.ConditionStatus, reason string) {
	crb := &fleetv1beta1.ClusterResourceBinding{}
	By(fmt.Sprintf("Updating the clusterResourceBinding status - %s, %d, %s, %s", conditionType, observedGeneration, status, reason))
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testCRBName}, crb)).Should(Succeed(), "failed to get cluster resource binding")
	condition := metav1.Condition{
		Type:               string(conditionType),
		ObservedGeneration: observedGeneration,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
	}
	crb.SetConditions(condition)
	Expect(k8sClient.Status().Update(ctx, crb)).Should(Succeed(), "failed to update cluster resource binding status")

	By("Checking placement controller queue")
	eventuallyCheckPlacementControllerQueue(crb.GetLabels()[fleetv1beta1.PlacementTrackingLabel])
	fakePlacementController.ResetQueue()
}

func validateWhenUpdateResourceBindingStatusWithCondition(conditionType fleetv1beta1.ResourceBindingConditionType, observedGeneration int64, status metav1.ConditionStatus, reason string) {
	rb := &fleetv1beta1.ResourceBinding{}
	By(fmt.Sprintf("Updating the resourceBinding status - %s, %d, %s, %s", conditionType, observedGeneration, status, reason))
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testRBName, Namespace: testNamespace}, rb)).Should(Succeed(), "failed to get resource binding")
	condition := metav1.Condition{
		Type:               string(conditionType),
		ObservedGeneration: observedGeneration,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
	}
	rb.SetConditions(condition)
	Expect(k8sClient.Status().Update(ctx, rb)).Should(Succeed(), "failed to update resource binding status")

	By("Checking placement controller queue")
	eventuallyCheckPlacementControllerQueue(controller.GetObjectKeyFromNamespaceName(testNamespace, rb.GetLabels()[fleetv1beta1.PlacementTrackingLabel]))
	fakePlacementController.ResetQueue()
}

func eventuallyCheckPlacementControllerQueue(key string) {
	Eventually(func() bool {
		return fakePlacementController.Key() == key
	}, eventuallyTimeout, interval).Should(BeTrue(), "placementController should receive the cluster resource placement name")
}

func consistentlyCheckPlacementControllerQueueIsEmpty() {
	Consistently(func() bool {
		return fakePlacementController.Key() == ""
	}, consistentlyDuration, interval).Should(BeTrue(), "watcher should ignore the create event and not enqueue the request into the placementController queue")
}
