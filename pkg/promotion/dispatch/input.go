package dispatch

import (
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocdapi "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/pattern"
)

// Promotion classes distinguished by dispatch policy.
const (
	// ClassAutoForward is a forward promotion created by the system (e.g.
	// auto-promotion).
	ClassAutoForward = "auto-forward"
	// ClassManualForward is a forward promotion created by a user.
	ClassManualForward = "manual-forward"
	// ClassRollback is a promotion marked as a rollback.
	ClassRollback = "rollback"
)

// Freeze scopes and the promotion classes each freezes.
var defaultScopes = map[string]any{
	"no-promotions": []any{ClassAutoForward, ClassManualForward, ClassRollback},
	"no-forward":    []any{ClassAutoForward, ClassManualForward},
	"no-auto":       []any{ClassAutoForward},
}

// ClassOf infers a Promotion's class. The actor inference mirrors the
// promotion webhook's intent inference: the Stage controller's
// auto-promotions carry no create-actor annotation and other controllers
// identify themselves with a "controller:" actor, while user-initiated
// Promotions name the requesting user.
func ClassOf(promo *kargoapi.Promotion) string {
	if promo.Annotations[kargoapi.AnnotationKeyRollback] == kargoapi.AnnotationValueTrue {
		return ClassRollback
	}
	actor := promo.Annotations[kargoapi.AnnotationKeyCreateActor]
	if actor == "" || strings.HasPrefix(actor, kargoapi.EventActorControllerPrefix) {
		return ClassAutoForward
	}
	return ClassManualForward
}

// BuildInput assembles the policy input document for one candidate
// Promotion. freight and project may be nil; apps may be empty.
func BuildInput(
	promo *kargoapi.Promotion,
	freight *kargoapi.Freight,
	stage *kargoapi.Stage,
	project *kargoapi.Project,
	apps []argocdapi.Application,
	now time.Time,
) map[string]any {
	input := map[string]any{
		"promotion": map[string]any{
			"name":        promo.Name,
			"class":       ClassOf(promo),
			"createdAt":   promo.CreationTimestamp.UTC().Format(time.RFC3339),
			"actor":       promo.Annotations[kargoapi.AnnotationKeyCreateActor],
			"labels":      stringMap(promo.Labels),
			"annotations": stringMap(promo.Annotations),
		},
		"freight": freightDoc(freight),
		"stage": map[string]any{
			"name":          stage.Name,
			"project":       stage.Namespace,
			"labels":        stringMap(stage.Labels),
			"annotations":   stringMap(stage.Annotations),
			"lastPromotion": lastPromotionDoc(stage.Status.LastPromotion),
		},
		"project":      projectDoc(project),
		"applications": applicationDocs(apps),
		"now":          now.UTC().Format(time.RFC3339),
	}
	return input
}

// CurrentFreight is a Stage's current Freight for one origin, paired with its
// discovery time. A FreightReference (all the FreightHistory carries) has no
// discovery time, so the gate caller resolves the Freight object to obtain it
// and passes the result to BuildData.
type CurrentFreight struct {
	Name         string
	DiscoveredAt time.Time
}

// BuildData assembles the policy data document for one Stage. projectSpec
// and project may be nil. dispatches are the times at which the Stage's
// recent promotions were dispatched (began Running), for rate limiting.
// queue is the Stage's Promotions awaiting dispatch, in the order the gate
// considers them, so a policy can reason about the rest of the backlog (e.g.
// yield to a queued rollback, or grow conservative under a deep backlog).
// currentFreight is the Stage's current Freight per origin, so a policy can
// tell whether a candidate advances or regresses the Stage (data.currentFreight,
// keyed by origin). autoPromotionHolds is the Stage's committed auto-promotion
// holds per origin, so the gate can deny an auto-forward for a held origin
// (data.autoPromotionHolds, keyed by origin). Freezes whose ProjectSelector
// does not match the Project are omitted.
func BuildData(
	projectSpec *kargoapi.ProjectConfigSpec,
	freezes []kargoapi.PromotionFreeze,
	stage *kargoapi.Stage,
	project *kargoapi.Project,
	dispatches []time.Time,
	queue []kargoapi.Promotion,
	currentFreight map[string]CurrentFreight,
	autoPromotionHolds map[string]kargoapi.AutoPromotionHold,
) (map[string]any, error) {
	windows := []any{}
	rateLimit := map[string]any{}
	if projectSpec != nil {
		for _, w := range projectSpec.PromotionWindows {
			matches, err := selectorMatches(w.StageSelector, stage)
			if err != nil {
				return nil, fmt.Errorf("promotion window %q: %w", w.Name, err)
			}
			if !matches {
				continue
			}
			windows = append(windows, map[string]any{
				"name":       w.Name,
				"recurrence": w.Recurrence,
				"start":      w.Start,
				"end":        w.End,
				"location":   w.Location,
			})
		}
		for _, rl := range projectSpec.RateLimits {
			matches, err := selectorMatches(rl.StageSelector, stage)
			if err != nil {
				return nil, fmt.Errorf("rate limit %q: %w", rl.Name, err)
			}
			if !matches {
				continue
			}
			dispatchNS := make([]any, len(dispatches))
			for i, d := range dispatches {
				dispatchNS[i] = d.UnixNano()
			}
			rateLimit[stage.Name] = map[string]any{
				"max":        int64(rl.MaxPromotions),
				"window":     rl.Window.Nanoseconds(),
				"dispatches": dispatchNS,
			}
			break // first matching rate limit wins
		}
	}
	freezeDocs := []any{}
	for _, f := range freezes {
		matches, err := projectSelectorMatches(f.ProjectSelector, project)
		if err != nil {
			return nil, fmt.Errorf("freeze %q: %w", f.Name, err)
		}
		if !matches {
			continue
		}
		servers := make([]any, len(f.ArgoCDServers))
		for j, s := range f.ArgoCDServers {
			servers[j] = s
		}
		freezeDocs = append(freezeDocs, map[string]any{
			"name":          f.Name,
			"start":         f.Start.UTC().Format(time.RFC3339),
			"end":           f.End.UTC().Format(time.RFC3339),
			"scope":         f.Scope,
			"argocdServers": servers,
		})
	}
	return map[string]any{
		"windows":            windows,
		"freezes":            freezeDocs,
		"scopes":             defaultScopes,
		"rateLimit":          rateLimit,
		"queue":              queueDocs(queue),
		"currentFreight":     currentFreightDocs(currentFreight),
		"autoPromotionHolds": autoPromotionHoldsDocs(autoPromotionHolds),
	}, nil
}

// currentFreightDocs projects the Stage's current Freight per origin into
// policy documents, keyed by origin. Each entry carries the current Freight's
// name and discovery time (RFC3339), so a policy can compare the candidate's
// Freight clock against the origin it would replace (kargo.advances /
// kargo.regresses). An origin with no resolvable current Freight is absent, so
// the comparison for that origin is simply undefined (the policy fails open).
func currentFreightDocs(currentFreight map[string]CurrentFreight) map[string]any {
	docs := make(map[string]any, len(currentFreight))
	for origin, cf := range currentFreight {
		docs[origin] = map[string]any{
			"name":         cf.Name,
			"discoveredAt": cf.DiscoveredAt.UTC().Format(time.RFC3339),
		}
	}
	return docs
}

// autoPromotionHoldsDocs projects the Stage's committed auto-promotion holds
// into policy documents, keyed by origin. The gate's auto-hold rule needs only
// an origin's presence; the projected fields (freightName, promotionName,
// actor, createdAt) let a policy or an operator see what established the hold.
// An origin with no hold is absent, so the auto-hold check for it is simply
// undefined.
func autoPromotionHoldsDocs(holds map[string]kargoapi.AutoPromotionHold) map[string]any {
	docs := make(map[string]any, len(holds))
	for origin, hold := range holds {
		doc := map[string]any{
			"freightName":   hold.FreightName,
			"promotionName": hold.PromotionName,
			"actor":         hold.Actor,
		}
		if hold.CreatedAt != nil {
			doc["createdAt"] = hold.CreatedAt.UTC().Format(time.RFC3339)
		}
		docs[origin] = doc
	}
	return docs
}

// queueDocs projects the Promotions awaiting dispatch into policy documents,
// preserving the gate's evaluation order. Each entry carries only identity,
// class, and creation time -- enough for a policy to weigh the backlog
// against the candidate it is evaluating (found by input.promotion.name)
// without fetching Freight for every queued Promotion.
func queueDocs(queue []kargoapi.Promotion) []any {
	docs := make([]any, len(queue))
	for i := range queue {
		promo := &queue[i]
		docs[i] = map[string]any{
			"name":      promo.Name,
			"class":     ClassOf(promo),
			"createdAt": promo.CreationTimestamp.UTC().Format(time.RFC3339),
		}
	}
	return docs
}

// selectorMatches reports whether the selector matches the Stage, using the
// same name-pattern and label-selector semantics as PromotionPolicies. A
// nil selector matches every Stage.
func selectorMatches(
	selector *kargoapi.PromotionPolicySelector,
	stage *kargoapi.Stage,
) (bool, error) {
	if selector == nil {
		return true, nil
	}
	if selector.Name != "" {
		m, err := pattern.ParseNamePattern(selector.Name)
		if err != nil {
			return false, fmt.Errorf("error parsing stage selector name pattern %q: %w", selector.Name, err)
		}
		if !m.Matches(stage.Name) {
			return false, nil
		}
	}
	if selector.LabelSelector != nil {
		s, err := metav1.LabelSelectorAsSelector(selector.LabelSelector)
		if err != nil {
			return false, fmt.Errorf("error parsing stage label selector: %w", err)
		}
		if !s.Matches(labels.Set(stage.Labels)) {
			return false, nil
		}
	}
	return true, nil
}

// projectSelectorMatches reports whether the label selector matches the
// Project's labels. A nil selector matches every Project; a nil Project is
// treated as an empty label set (a non-nil matchLabels selector will not
// match it).
func projectSelectorMatches(
	selector *metav1.LabelSelector,
	project *kargoapi.Project,
) (bool, error) {
	if selector == nil {
		return true, nil
	}
	s, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false, fmt.Errorf("error parsing project label selector: %w", err)
	}
	var projectLabels map[string]string
	if project != nil {
		projectLabels = project.Labels
	}
	return s.Matches(labels.Set(projectLabels)), nil
}

func freightDoc(freight *kargoapi.Freight) map[string]any {
	if freight == nil {
		return map[string]any{}
	}
	return map[string]any{
		"name":         freight.Name,
		"alias":        freight.Alias,
		"origin":       freight.Origin.String(),
		"discoveredAt": freight.EffectiveDiscoveredAt().UTC().Format(time.RFC3339),
		"images":       imageDocs(freight.Images),
		"commits":      commitDocs(freight.Commits),
		"charts":       chartDocs(freight.Charts),
	}
}

func lastPromotionDoc(ref *kargoapi.PromotionReference) map[string]any {
	if ref == nil {
		return map[string]any{}
	}
	doc := map[string]any{"name": ref.Name}
	if ref.Status != nil {
		doc["phase"] = string(ref.Status.Phase)
	}
	if ref.FinishedAt != nil {
		doc["finishedAt"] = ref.FinishedAt.UTC().Format(time.RFC3339)
	}
	if ref.Freight != nil {
		doc["freight"] = map[string]any{
			"name":    ref.Freight.Name,
			"origin":  ref.Freight.Origin.String(),
			"images":  imageDocs(ref.Freight.Images),
			"commits": commitDocs(ref.Freight.Commits),
			"charts":  chartDocs(ref.Freight.Charts),
		}
	}
	return doc
}

func projectDoc(project *kargoapi.Project) map[string]any {
	if project == nil {
		return map[string]any{
			"labels":      map[string]any{},
			"annotations": map[string]any{},
		}
	}
	return map[string]any{
		"labels":      stringMap(project.Labels),
		"annotations": stringMap(project.Annotations),
	}
}

func applicationDocs(apps []argocdapi.Application) []any {
	docs := make([]any, len(apps))
	for i, app := range apps {
		docs[i] = map[string]any{
			"name":      app.Name,
			"namespace": app.Namespace,
			"destination": map[string]any{
				"server":    app.Spec.Destination.Server,
				"name":      app.Spec.Destination.Name,
				"namespace": app.Spec.Destination.Namespace,
			},
			"health":      string(app.Status.Health.Status),
			"sync":        string(app.Status.Sync.Status),
			"labels":      stringMap(app.Labels),
			"annotations": stringMap(app.Annotations),
		}
	}
	return docs
}

func imageDocs(images []kargoapi.Image) []any {
	docs := make([]any, len(images))
	for i, img := range images {
		docs[i] = map[string]any{
			"repoURL": img.RepoURL,
			"tag":     img.Tag,
			"digest":  img.Digest,
		}
	}
	return docs
}

func commitDocs(commits []kargoapi.GitCommit) []any {
	docs := make([]any, len(commits))
	for i, c := range commits {
		docs[i] = map[string]any{
			"repoURL": c.RepoURL,
			"id":      c.ID,
			"branch":  c.Branch,
			"tag":     c.Tag,
		}
	}
	return docs
}

func chartDocs(charts []kargoapi.Chart) []any {
	docs := make([]any, len(charts))
	for i, c := range charts {
		docs[i] = map[string]any{
			"repoURL": c.RepoURL,
			"name":    c.Name,
			"version": c.Version,
		}
	}
	return docs
}

// stringMap converts a string map into a JSON-friendly document, never nil.
func stringMap(m map[string]string) map[string]any {
	doc := make(map[string]any, len(m))
	for k, v := range m {
		doc[k] = v
	}
	return doc
}
