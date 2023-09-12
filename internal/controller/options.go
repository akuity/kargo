package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

func CommonOptions() controller.Options {
	t := true
	return controller.Options{
		RecoverPanic: &t,
	}
}
