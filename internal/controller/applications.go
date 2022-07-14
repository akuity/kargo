package controller

import (
	"context"

	log "github.com/sirupsen/logrus"

	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func (c *controller) syncApplications(ctx context.Context) {
	applicationsInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return c.argocdClient.ArgoprojV1alpha1().Applications("").List(
					ctx,
					options,
				)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return c.argocdClient.ArgoprojV1alpha1().Applications("").Watch(
					ctx,
					options,
				)
			},
		},
		&v1alpha1.Application{},
		0,
		cache.Indexers{},
	)
	applicationsInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.syncApplicationFn,
			UpdateFunc: func(_, newObj any) {
				c.syncApplicationFn(newObj)
			},
		},
	)
	applicationsInformer.Run(ctx.Done())
}

func (c *controller) syncApplication(obj any) {
	application := obj.(*v1alpha1.Application) // nolint: forcetypeassert

	c.logger.WithFields(log.Fields{
		"namespace":       application.Namespace,
		"name":            application.Name,
		"currentRevision": application.Status.Sync.Revision,
	}).Debug("Syncing Argo CD Application")

	// TODO: Finish implementing this. The idea here is to make determinations
	// about whether or not a given change has progressed into the environment
	// represented by the Application. When it has, we can trigger the next
	// migration.
}
