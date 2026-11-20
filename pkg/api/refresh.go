package api

import (
	"context"
	"reflect"
	"time"

	"go.opentelemetry.io/otel"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/telemetry"
)

// tracer records this package's spans.
var tracer = otel.Tracer("github.com/akuity/kargo/pkg/api")

// RefreshObject forces reconciliation of a Kubernetes object by setting an
// annotation on the object, causing the controller to reconcile it. Currently,
// the annotation value is the timestamp of the request, but in the future
// may include additional metadata/context necessary for the request.
//
// Each call records a "Refresh <Kind>" span.
func RefreshObject(ctx context.Context, c client.Client, obj client.Object) error {
	name := "Refresh"
	if telemetry.CategoryEnabled(telemetry.CategoryRefresh) {
		// Only a span that may record needs the kind in its name.
		name += " " + reflect.TypeOf(obj).Elem().Name()
	}
	ctx, span := telemetry.StartSpanIn(ctx, telemetry.CategoryRefresh, tracer, name)
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(telemetry.ObjectAttributes(obj)...)
	}
	err := patchAnnotation(
		ctx,
		c,
		obj,
		kargoapi.AnnotationKeyRefresh,
		time.Now().Format(time.RFC3339Nano),
	)
	// Callers report NotFound as a 404.
	if !apierrors.IsNotFound(err) {
		telemetry.RecordErrorCtx(ctx, span, err)
	}
	return err
}
