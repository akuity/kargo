package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupValidatingAndDefaultingWebhook registers both a validating webhook and
// a defaulting webhook (see RegisterDefaultingWebhook) for obj's type on mgr,
// using h for both.
func SetupValidatingAndDefaultingWebhook[T runtime.Object](
	mgr ctrl.Manager,
	obj T,
	h interface {
		admission.Validator[T]
		admission.Defaulter[T]
	},
) error {
	if err := ctrl.NewWebhookManagedBy(mgr, obj).
		WithValidator(h).
		Complete(); err != nil {
		return fmt.Errorf("error creating validating webhook for %T: %w", obj, err)
	}
	return RegisterDefaultingWebhook(mgr, obj, h)
}

// RegisterDefaultingWebhook registers a mutating admission webhook for obj's
// type on mgr's webhook server, running defaulter's Default() method the way
// ctrl.NewWebhookManagedBy(mgr, obj).WithDefaulter(defaulter) would, except
// for how the resulting JSON patch is computed. See NewDefaultingWebhook.
//
// Like controller-runtime's own webhook registration, this panics if called
// twice for the same path against the same manager, so a given obj type must
// only be registered once per manager.
func RegisterDefaultingWebhook[T runtime.Object](
	mgr ctrl.Manager,
	obj T,
	defaulter admission.Defaulter[T],
) error {
	gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("error getting GroupVersionKind for %T: %w", obj, err)
	}
	wh, err := NewDefaultingWebhook(mgr.GetScheme(), obj, defaulter)
	if err != nil {
		return err
	}
	mgr.GetWebhookServer().Register(mutatePath(gvk), wh)
	return nil
}

// mutatePath reproduces controller-runtime's own formula for generating a
// defaulting webhook's path from its GVK, so it can't drift from the paths
// hand-configured in the Helm chart.
func mutatePath(gvk schema.GroupVersionKind) string {
	return "/mutate-" + strings.ReplaceAll(gvk.Group, ".", "-") + "-" +
		gvk.Version + "-" + strings.ToLower(gvk.Kind)
}

// NewDefaultingWebhook is like admission.WithDefaulter, except its patch
// diffs obj as marshaled before and after Default() runs, rather than the
// raw request against the marshaled object. This avoids spurious patches for
// fields that don't round-trip byte-for-byte through JSON, such as
// metav1.Duration ("1h" remarshals as "1h0m0s"), when Default() never
// touched them. See https://github.com/akuityio/akuity-platform/issues/12384.
func NewDefaultingWebhook[T runtime.Object](
	scheme *runtime.Scheme,
	obj T,
	defaulter admission.Defaulter[T],
) (*admission.Webhook, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return nil, fmt.Errorf("error getting GroupVersionKind for %T: %w", obj, err)
	}
	return &admission.Webhook{
		Handler: &defaultingHandler[T]{
			defaulter: defaulter,
			decoder:   admission.NewDecoder(scheme),
			new: func() T {
				copied, ok := obj.DeepCopyObject().(T)
				if !ok {
					panic(fmt.Sprintf("defaulting webhook: DeepCopyObject() did not return a %T", obj))
				}
				return copied
			},
		},
		LogConstructor: defaultingLogConstructor(gvk),
	}, nil
}

// defaultingLogConstructor adds the same fields to the logger that
// controller-runtime's own webhook builder would.
func defaultingLogConstructor(gvk schema.GroupVersionKind) func(logr.Logger, *admission.Request) logr.Logger {
	return func(base logr.Logger, req *admission.Request) logr.Logger {
		log := base.WithValues("webhookGroup", gvk.Group, "webhookKind", gvk.Kind)
		if req == nil {
			return log
		}
		return log.WithValues(
			gvk.Kind, klog.KRef(req.Namespace, req.Name),
			"namespace", req.Namespace, "name", req.Name,
			"resource", req.Resource, "user", req.UserInfo.Username,
			"requestID", req.UID,
		)
	}
}

type defaultingHandler[T runtime.Object] struct {
	defaulter admission.Defaulter[T]
	decoder   admission.Decoder
	new       func() T
}

func (h *defaultingHandler[T]) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation == admissionv1.Delete {
		return admission.Response{
			AdmissionResponse: admissionv1.AdmissionResponse{
				Allowed: true,
				Result:  &metav1.Status{Code: http.StatusOK},
			},
		}
	}

	ctx = admission.NewContextWithRequest(ctx, req)

	obj := h.new()
	if err := h.decoder.Decode(req, obj); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	orig := obj.DeepCopyObject()

	if defaultErr := h.defaulter.Default(ctx, obj); defaultErr != nil {
		var apiStatus apierrors.APIStatus
		if errors.As(defaultErr, &apiStatus) {
			status := apiStatus.Status()
			return admission.Response{
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed: false,
					Result:  &status,
				},
			}
		}
		return admission.Denied(defaultErr.Error())
	}

	// Default() left the object unchanged: skip the marshal/diff below, just
	// like controller-runtime's own no-op short-circuit.
	if reflect.DeepEqual(orig, obj) {
		return admission.Response{
			AdmissionResponse: admissionv1.AdmissionResponse{Allowed: true},
		}
	}

	before, err := json.Marshal(orig)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	after, err := json.Marshal(obj)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(before, after)
}
