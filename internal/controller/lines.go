package controller

import (
	"context"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func (c *controller) syncLines(ctx context.Context) {
	linesSelector := labels.Set(
		map[string]string{
			LabelKeyComponent: "line",
		},
	).AsSelector().String()
	linesInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = linesSelector
				return c.kubeClient.CoreV1().ConfigMaps("").List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = linesSelector
				return c.kubeClient.CoreV1().ConfigMaps("").Watch(ctx, options)
			},
		},
		&corev1.ConfigMap{},
		0,
		cache.Indexers{},
	)
	linesInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.syncLineFn,
			UpdateFunc: func(_, newObj any) {
				c.syncLineFn(newObj)
			},
		},
	)
	linesInformer.Run(ctx.Done())
}

func (c *controller) syncLine(obj any) {
	configMap := obj.(*corev1.ConfigMap) // nolint: forcetypeassert

	c.logger.WithFields(log.Fields{
		"namespace": configMap.Namespace,
		"name":      configMap.Name,
	}).Debug("Syncing K8sTA Line")

	// TODO: Finish implementing this. The idea here is to continuously update
	// internal configuration based on what's in the ConfigMap.
}
