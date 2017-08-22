/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package collectors

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscaling "k8s.io/client-go/pkg/apis/autoscaling/v1"
	"k8s.io/client-go/pkg/util"
)

var (
	hpa1MinReplicas int32 = 2
)

type mockHPAStore struct {
	list func() (autoscaling.HorizontalPodAutoscalerList, error)
}

func (hs mockHPAStore) List() (autoscaling.HorizontalPodAutoscalerList, error) {
	return hs.list()
}

func TestHPACollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_hpa_metadata_generation The generation observed by the HorizontalPodAutoscaler controller.
		# TYPE kube_hpa_metadata_generation gauge
		# HELP kube_hpa_spec_max_replicas Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.
		# TYPE kube_hpa_spec_max_replicas gauge
		# HELP kube_hpa_spec_min_replicas Lower limit for the number of pods that can be set by the autoscaler, default 1.
		# TYPE kube_hpa_spec_min_replicas gauge
		# HELP kube_hpa_status_current_replicas Current number of replicas of pods managed by this autoscaler.
		# TYPE kube_hpa_status_current_replicas gauge
		# HELP kube_hpa_status_desired_replicas Desired number of replicas of pods managed by this autoscaler.
		# TYPE kube_hpa_status_desired_replicas gauge
		# HELP kube_hpa_spec_target_cpu Target CPU percentage of pods managed by this autoscaler.
		# TYPE kube_hpa_spec_target_cpu gauge
		# HELP kube_hpa_status_current_cpu Current CPU percentage of the pods managed by this autoscaler.
		# TYPE kube_hpa_status_current_cpu gauge
	`
	cases := []struct {
		hpas    []autoscaling.HorizontalPodAutoscaler
		metrics []string // which metrics should be checked
		want    string
	}{
		// Verify populating base metrics.
		{
			hpas: []autoscaling.HorizontalPodAutoscaler{
				{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 2,
						Name:       "hpa1",
						Namespace:  "ns1",
					},
					Spec: autoscaling.HorizontalPodAutoscalerSpec{
						MaxReplicas: 4,
						MinReplicas: &hpa1MinReplicas,
						ScaleTargetRef: autoscaling.CrossVersionObjectReference{
							APIVersion: "extensions/v1beta1",
							Kind:       "Deployment",
							Name:       "deployment1",
						},
						TargetCPUUtilizationPercentage: util.Int32Ptr(50),
					},
					Status: autoscaling.HorizontalPodAutoscalerStatus{
						CurrentReplicas:                 2,
						DesiredReplicas:                 2,
						CurrentCPUUtilizationPercentage: util.Int32Ptr(25),
					},
				},
			},
			want: metadata + `
				kube_hpa_metadata_generation{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_spec_max_replicas{hpa="hpa1",namespace="ns1"} 4
				kube_hpa_spec_min_replicas{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_status_current_replicas{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_status_desired_replicas{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_spec_target_cpu{hpa="hpa1",namespace="ns1"} 50
				kube_hpa_status_current_cpu{hpa="hpa1",namespace="ns1"} 25
			`,
			metrics: []string{
				"kube_hpa_metadata_generation",
				"kube_hpa_spec_max_replicas",
				"kube_hpa_spec_min_replicas",
				"kube_hpa_status_current_replicas",
				"kube_hpa_status_desired_replicas",
				"kube_hpa_spec_target_cpu",
				"kube_hpa_status_current_cpu",
			},
		},
	}
	for _, c := range cases {
		hc := &hpaCollector{
			store: &mockHPAStore{
				list: func() (autoscaling.HorizontalPodAutoscalerList, error) {
					return autoscaling.HorizontalPodAutoscalerList{Items: c.hpas}, nil
				},
			},
		}
		if err := gatherAndCompare(hc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
