package api

import (
	authnv1 "k8s.io/api/authentication/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func FormatEventControllerActor(name string) string {
	return kargoapi.EventActorControllerPrefix + name
}

func FormatEventKubernetesUserActor(u authnv1.UserInfo) string {
	return kargoapi.EventActorKubernetesUserPrefix + u.Username
}
