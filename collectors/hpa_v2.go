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
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	autoscaling "k8s.io/client-go/pkg/apis/autoscaling/v2alpha1"
	"k8s.io/client-go/tools/cache"
)

var (
	descHorizontalPodAutoscalerV2LabelsName          = "kube_hpav2_labels"
	descHorizontalPodAutoscalerV2LabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descHorizontalPodAutoscalerV2LabelsDefaultLabels = []string{"namespace", "hpav2"}

	descHorizontalPodAutoscalerV2MetadataGeneration = prometheus.NewDesc(
		"kube_hpav2_metadata_generation",
		"The generation observed by the HorizontalPodAutoscaler controller.",
		[]string{"namespace", "hpav2"}, nil,
	)
	descHorizontalPodAutoscalerV2SpecMaxReplicas = prometheus.NewDesc(
		"kube_hpav2_spec_max_replicas",
		"Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.",
		[]string{"namespace", "hpav2"}, nil,
	)
	descHorizontalPodAutoscalerV2SpecMinReplicas = prometheus.NewDesc(
		"kube_hpav2_spec_min_replicas",
		"Lower limit for the number of pods that can be set by the autoscaler, default 1.",
		[]string{"namespace", "hpav2"}, nil,
	)
	descHorizontalPodAutoscalerV2StatusCurrentReplicas = prometheus.NewDesc(
		"kube_hpav2_status_current_replicas",
		"Current number of replicas of pods managed by this autoscaler.",
		[]string{"namespace", "hpav2"}, nil,
	)
	descHorizontalPodAutoscalerV2StatusDesiredReplicas = prometheus.NewDesc(
		"kube_hpav2_status_desired_replicas",
		"Desired number of replicas of pods managed by this autoscaler.",
		[]string{"namespace", "hpav2"}, nil,
	)
	descHorizontalPodAutoscalerV2SpecTargetCPU = prometheus.NewDesc(
		"kube_hpav2_spec_target_cpu",
		"Target CPU percentage of pods managed by this autoscaler.",
		[]string{"namespace", "hpav2"}, nil,
	)
	descHorizontalPodAutoscalerV2StatusCurrentCPU = prometheus.NewDesc(
		"kube_hpav2_status_current_cpu",
		"Current CPU percentage of the pods managed by this autoscaler.",
		[]string{"namespace", "hpav2"}, nil,
	)
	descHorizontalPodAutoscalerV2Labels = prometheus.NewDesc(
		descHorizontalPodAutoscalerV2LabelsName,
		descHorizontalPodAutoscalerV2LabelsHelp,
		descHorizontalPodAutoscalerV2LabelsDefaultLabels, nil,
	)
)

type HPAListerV2 func() (autoscaling.HorizontalPodAutoscalerList, error)

func (l HPAListerV2) List() (autoscaling.HorizontalPodAutoscalerList, error) {
	return l()
}

func RegisterHorizontalPodAutoScalerV2Collector(registry prometheus.Registerer, kubeClient kubernetes.Interface, namespace string) {
	client := kubeClient.AutoscalingV2alpha1().RESTClient()
	hpalw := cache.NewListWatchFromClient(client, "horizontalpodautoscalers", api.NamespaceAll, nil)
	hpainf := cache.NewSharedInformer(hpalw, &autoscaling.HorizontalPodAutoscaler{}, resyncPeriod)

	hpaLister := HPAListerV2(func() (hpas autoscaling.HorizontalPodAutoscalerList, err error) {
		for _, h := range hpainf.GetStore().List() {
			//spew.Dump(h)
			switch h := h.(type) {
			case *autoscaling.HorizontalPodAutoscaler:
				hpas.Items = append(hpas.Items, *h)
			default:
				glog.Infof("JOE Skipping type %T", h)
			}
			//hpas.Items = append(hpas.Items, *(h.(*autoscaling.HorizontalPodAutoscaler)))
		}
		return hpas, nil
	})

	registry.MustRegister(&hpaV2Collector{store: hpaLister})
	go hpainf.Run(context.Background().Done())
}

type hpaV2Store interface {
	List() (hpas autoscaling.HorizontalPodAutoscalerList, err error)
}

// hpaV2Collector collects metrics about all Horizontal Pod Austoscalers in the cluster.
type hpaV2Collector struct {
	store hpaV2Store
}

// Describe implements the prometheus.Collector interface.
func (hc *hpaV2Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descHorizontalPodAutoscalerV2MetadataGeneration
	ch <- descHorizontalPodAutoscalerV2SpecMaxReplicas
	ch <- descHorizontalPodAutoscalerV2SpecMinReplicas
	ch <- descHorizontalPodAutoscalerV2StatusCurrentReplicas
	ch <- descHorizontalPodAutoscalerV2StatusDesiredReplicas
	ch <- descHorizontalPodAutoscalerV2SpecTargetCPU
	ch <- descHorizontalPodAutoscalerV2StatusCurrentCPU
	ch <- descHorizontalPodAutoscalerV2Labels
}

// Collect implements the prometheus.Collector interface.
func (hc *hpaV2Collector) Collect(ch chan<- prometheus.Metric) {
	hpas, err := hc.store.List()
	if err != nil {
		glog.Errorf("listing HorizontalPodAutoscalers failed: %s", err)
		return
	}
	for _, h := range hpas.Items {
		glog.Info("JOE spew")
		spew.Dump(h)
		hc.collectHPA(ch, h)
	}
}

func hpaV2LabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descHorizontalPodAutoscalerV2LabelsName,
		descHorizontalPodAutoscalerV2LabelsHelp,
		append(descHorizontalPodAutoscalerV2LabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (hc *hpaV2Collector) collectHPA(ch chan<- prometheus.Metric, h autoscaling.HorizontalPodAutoscaler) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{h.Namespace, h.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(h.Labels)
	addGauge(hpaLabelsDesc(labelKeys), 1, labelValues...)
	addGauge(descHorizontalPodAutoscalerV2MetadataGeneration, float64(h.ObjectMeta.Generation))
	addGauge(descHorizontalPodAutoscalerV2SpecMaxReplicas, float64(h.Spec.MaxReplicas))
	addGauge(descHorizontalPodAutoscalerV2SpecMinReplicas, float64(*h.Spec.MinReplicas))
	addGauge(descHorizontalPodAutoscalerV2StatusCurrentReplicas, float64(h.Status.CurrentReplicas))
	addGauge(descHorizontalPodAutoscalerV2StatusDesiredReplicas, float64(h.Status.DesiredReplicas))
	// if h.Spec.TargetCPUUtilizationPercentage != nil {
	// 	addGauge(descHorizontalPodAutoscalerV2SpecTargetCPU, float64(*h.Spec.TargetCPUUtilizationPercentage))
	// }
	// if h.Status.CurrentCPUUtilizationPercentage != nil {
	// 	addGauge(descHorizontalPodAutoscalerV2StatusCurrentCPU, float64(*h.Status.CurrentCPUUtilizationPercentage))
	// }
}
