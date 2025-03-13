package event

import (
	"context"
	"encoding/json"
	"time"

	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/x/promotion/runners/builtin"
)

// NewPromotionAnnotations returns annotations for a Promotion related event.
// It may skip some fields when error occurred during serialization, to record event with best-effort.
func NewPromotionAnnotations(
	ctx context.Context,
	actor string,
	p *kargoapi.Promotion,
	f *kargoapi.Freight,
) map[string]string {
	logger := logging.LoggerFromContext(ctx)

	annotations := map[string]string{
		kargoapi.AnnotationKeyEventProject:             p.GetNamespace(),
		kargoapi.AnnotationKeyEventPromotionName:       p.GetName(),
		kargoapi.AnnotationKeyEventFreightName:         p.Spec.Freight,
		kargoapi.AnnotationKeyEventStageName:           p.Spec.Stage,
		kargoapi.AnnotationKeyEventPromotionCreateTime: p.GetCreationTimestamp().Format(time.RFC3339),
	}

	if actor != "" {
		annotations[kargoapi.AnnotationKeyEventActor] = actor
	}
	// All Promotion-related events are emitted after the promotion was created.
	// Therefore, if the promotion knows who triggered it, set them as an actor.
	if promoteActor, ok := p.Annotations[kargoapi.AnnotationKeyCreateActor]; ok {
		annotations[kargoapi.AnnotationKeyEventActor] = promoteActor
	}

	if f != nil {
		annotations[kargoapi.AnnotationKeyEventFreightCreateTime] = f.CreationTimestamp.Format(time.RFC3339)
		annotations[kargoapi.AnnotationKeyEventFreightAlias] = f.Alias
		if len(f.Commits) > 0 {
			data, err := json.Marshal(f.Commits)
			if err != nil {
				logger.Error(err, "marshal freight commits in JSON")
			} else {
				annotations[kargoapi.AnnotationKeyEventFreightCommits] = string(data)
			}
		}
		if len(f.Images) > 0 {
			data, err := json.Marshal(f.Images)
			if err != nil {
				logger.Error(err, "marshal freight images in JSON")
			} else {
				annotations[kargoapi.AnnotationKeyEventFreightImages] = string(data)
			}
		}
		if len(f.Charts) > 0 {
			data, err := json.Marshal(f.Charts)
			if err != nil {
				logger.Error(err, "marshal freight charts in JSON")
			} else {
				annotations[kargoapi.AnnotationKeyEventFreightCharts] = string(data)
			}
		}
	}

	var apps []types.NamespacedName
	for _, step := range p.Spec.Steps {
		if step.Uses != "argocd-update" || step.Config == nil {
			continue
		}
		var cfg builtin.ArgoCDUpdateConfig
		if err := json.Unmarshal(step.Config.Raw, &cfg); err != nil {
			logger.Error(err, "unmarshal ArgoCD update config")
			continue
		}
		for _, app := range cfg.Apps {
			namespacedName := types.NamespacedName{
				Namespace: app.Namespace,
				Name:      app.Name,
			}
			if namespacedName.Namespace == "" {
				namespacedName.Namespace = libargocd.Namespace()
			}
			apps = append(apps, namespacedName)
		}
	}
	if len(apps) > 0 {
		data, err := json.Marshal(apps)
		if err != nil {
			logger.Error(err, "marshal ArgoCD apps in JSON")
		} else {
			annotations[kargoapi.AnnotationKeyEventApplications] = string(data)
		}
	}

	return annotations
}
