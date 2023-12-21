package controller

import (
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

func CommonOptions() controller.Options {
	return controller.Options{
		RecoverPanic: ptr.To(true),
	}
}
