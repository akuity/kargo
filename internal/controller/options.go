package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

func CommonOptions() controller.Options {
	return controller.Options{
		RecoverPanic: true,
	}
}
