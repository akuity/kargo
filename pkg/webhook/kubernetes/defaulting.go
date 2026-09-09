package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

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
) *admission.Webhook {
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
			AdmissionResponse: admissionv1.AdmissionResponse{Allowed: true},
		}
	}

	ctx = admission.NewContextWithRequest(ctx, req)

	obj := h.new()
	if err := h.decoder.Decode(req, obj); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	before, err := json.Marshal(obj)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

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

	after, err := json.Marshal(obj)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(before, after)
}
