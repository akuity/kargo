package controller

import (
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

func CommonOptions(maxConcurrentReconciles int) controller.Options {
	return controller.Options{
		MaxConcurrentReconciles: maxConcurrentReconciles,
		RecoverPanic:            ptr.To(true),
	}
}
