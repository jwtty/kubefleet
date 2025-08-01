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

package clusterresourcebinding

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	fleetv1beta1 "github.com/kubefleet-dev/kubefleet/apis/placement/v1beta1"
)

const (
	eventuallyDuration   = time.Second * 5
	eventuallyInterval   = time.Millisecond * 250
	consistentlyDuration = time.Second
	consistentlyInterval = time.Millisecond * 200

	crbName       = "test-crb"
	rbName        = "test-rb"
	crpName       = "test-crp"
	rpName        = "test-rp"
	clusterName   = "test-cluster"
	testNamespace = "test-ns"
)

var (
	noKeyEnqueuedActual = func() error {
		if queueLen := keyCollector.Len(); queueLen != 0 {
			return fmt.Errorf("work queue is not empty: current len %d, want 0", queueLen)
		}
		return nil
	}

	expectedCRPKeySetEnqueuedActual = func() error {
		return isKeyPresent(crpName)
	}

	expectedRPKeySetEnqueuedActual = func() error {
		expectedKey := fmt.Sprintf("%s/%s", testNamespace, rpName)
		return isKeyPresent(expectedKey)
	}
)

func isKeyPresent(key string) error {
	if isAllPresent, absentKeys := keyCollector.IsPresent(key); !isAllPresent {
		return fmt.Errorf("expected key(s) %v is not found", absentKeys)
	}

	if queueLen := keyCollector.Len(); queueLen != 1 {
		return fmt.Errorf("more than one key is enqueued: current len %d, want 1", queueLen)
	}

	return nil
}

// This container cannot be run in parallel since we are trying to access a common shared queue.
var _ = Describe("scheduler - cluster resource binding and resource binding watcher", Ordered, func() {
	BeforeAll(func() {
		Eventually(noKeyEnqueuedActual, eventuallyDuration, eventuallyInterval).Should(Succeed(), "Workqueue is not empty")
		Consistently(noKeyEnqueuedActual, consistentlyDuration, consistentlyInterval).Should(Succeed(), "Workqueue is not empty")
	})

	Context("create, update & delete cluster resource binding", func() {
		BeforeAll(func() {
			affinityScore := int32(1)
			topologyScore := int32(1)
			crb := fleetv1beta1.ClusterResourceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: crbName,
					Labels: map[string]string{
						fleetv1beta1.PlacementTrackingLabel: crpName,
					},
					Finalizers: []string{fleetv1beta1.SchedulerBindingCleanupFinalizer},
				},
				Spec: fleetv1beta1.ResourceBindingSpec{
					State: fleetv1beta1.BindingStateScheduled,
					// Leave the associated resource snapshot name empty; it is up to another controller
					// to fulfill this field.
					SchedulingPolicySnapshotName: "test-policy",
					TargetCluster:                clusterName,
					ClusterDecision: fleetv1beta1.ClusterDecision{
						ClusterName: clusterName,
						Selected:    true,
						ClusterScore: &fleetv1beta1.ClusterScore{
							AffinityScore:       &affinityScore,
							TopologySpreadScore: &topologyScore,
						},
						Reason: "test-reason",
					},
				},
			}
			// Create cluster resource binding.
			Expect(hubClient.Create(ctx, &crb)).Should(Succeed(), "Failed to create cluster resource binding")
		})

		It("should not enqueue the CRP name when CRB is created", func() {
			Consistently(noKeyEnqueuedActual, consistentlyDuration, consistentlyInterval).Should(Succeed(), "Workqueue is not empty")
		})

		It("update cluster resource binding", func() {
			var crb fleetv1beta1.ClusterResourceBinding
			Expect(hubClient.Get(ctx, client.ObjectKey{Name: crbName}, &crb)).Should(Succeed())
			crb.Spec.State = fleetv1beta1.BindingStateBound
			Expect(hubClient.Update(ctx, &crb)).Should(Succeed())
		})

		It("should not enqueue the CRP name when it CRB is updated", func() {
			Consistently(noKeyEnqueuedActual, consistentlyDuration, consistentlyInterval).Should(Succeed(), "Workqueue is not empty")
		})

		It("delete cluster resource binding", func() {
			var crb fleetv1beta1.ClusterResourceBinding
			Expect(hubClient.Get(ctx, client.ObjectKey{Name: crbName}, &crb)).Should(Succeed())
			Expect(hubClient.Delete(ctx, &crb)).Should(Succeed())
		})

		It("should enqueue CRP name when CRB is deleted", func() {
			Eventually(expectedCRPKeySetEnqueuedActual, eventuallyDuration, eventuallyInterval).Should(Succeed(), "Workqueue is either empty or it contains more than one element")
			Consistently(expectedCRPKeySetEnqueuedActual, consistentlyDuration, consistentlyInterval).Should(Succeed(), "Workqueue is either empty or it contains more than one element")
		})

		AfterAll(func() {
			keyCollector.Reset()
		})
	})

	Context("create, update & delete resource binding", func() {
		BeforeAll(func() {
			affinityScore := int32(1)
			topologyScore := int32(1)
			rb := fleetv1beta1.ResourceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rbName,
					Namespace: testNamespace,
					Labels: map[string]string{
						fleetv1beta1.PlacementTrackingLabel: rpName,
					},
					Finalizers: []string{fleetv1beta1.SchedulerBindingCleanupFinalizer},
				},
				Spec: fleetv1beta1.ResourceBindingSpec{
					State: fleetv1beta1.BindingStateScheduled,
					// Leave the associated resource snapshot name empty; it is up to another controller
					// to fulfill this field.
					SchedulingPolicySnapshotName: "test-policy",
					TargetCluster:                clusterName,
					ClusterDecision: fleetv1beta1.ClusterDecision{
						ClusterName: clusterName,
						Selected:    true,
						ClusterScore: &fleetv1beta1.ClusterScore{
							AffinityScore:       &affinityScore,
							TopologySpreadScore: &topologyScore,
						},
						Reason: "test-reason",
					},
				},
			}
			// Create resource binding.
			Expect(hubClient.Create(ctx, &rb)).Should(Succeed(), "Failed to create resource binding")
		})

		It("should not enqueue the RP name when RB is created", func() {
			Consistently(noKeyEnqueuedActual, consistentlyDuration, consistentlyInterval).Should(Succeed(), "Workqueue is not empty")
		})

		It("update resource binding", func() {
			var rb fleetv1beta1.ResourceBinding
			Expect(hubClient.Get(ctx, client.ObjectKey{Name: rbName, Namespace: testNamespace}, &rb)).Should(Succeed())
			rb.Spec.State = fleetv1beta1.BindingStateBound
			Expect(hubClient.Update(ctx, &rb)).Should(Succeed())
		})

		It("should not enqueue the RP name when RB is updated", func() {
			Consistently(noKeyEnqueuedActual, consistentlyDuration, consistentlyInterval).Should(Succeed(), "Workqueue is not empty")
		})

		It("delete resource binding", func() {
			var rb fleetv1beta1.ResourceBinding
			Expect(hubClient.Get(ctx, client.ObjectKey{Name: rbName, Namespace: testNamespace}, &rb)).Should(Succeed())
			Expect(hubClient.Delete(ctx, &rb)).Should(Succeed())
		})

		It("should enqueue RP name when RB is deleted", func() {
			Eventually(expectedRPKeySetEnqueuedActual, eventuallyDuration, eventuallyInterval).Should(Succeed(), "Workqueue is either empty or it contains more than one element")
			Consistently(expectedRPKeySetEnqueuedActual, consistentlyDuration, consistentlyInterval).Should(Succeed(), "Workqueue is either empty or it contains more than one element")
		})

		AfterAll(func() {
			keyCollector.Reset()
		})
	})
})
