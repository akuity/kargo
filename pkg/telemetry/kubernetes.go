package telemetry

import (
	"net/http"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"k8s.io/client-go/rest"
)

// WrapRESTConfig makes every Kubernetes API request through cfg a client
// span, "k8s <verb> <resource>[/<subresource>]", except watches, leases and
// events. A no-op unless CategoryK8s is enabled.
func WrapRESTConfig(cfg *rest.Config) {
	if cfg == nil || !CategoryEnabled(CategoryK8s) {
		return
	}
	// The path cfg.Host puts before /api and /apis, as through a proxy.
	var prefix string
	if u, err := url.Parse(cfg.Host); err == nil && u.Scheme != "" {
		if p := strings.Trim(u.Path, "/"); p != "" {
			prefix = p + "/"
		}
	}
	cfg.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return clientTransport{
			next:   rt,
			tracer: otel.GetTracerProvider().Tracer(clientScope),
			name: func(r *http.Request) (string, bool) {
				return kubeSpanName(r, prefix)
			},
			attributes: func(r *http.Request) []attribute.KeyValue {
				return kubeAttributes(r, prefix)
			},
			propagate: true,
		}
	})
}

// kubeWatch is a watch request's verb, and the deprecated path prefix for one.
const kubeWatch = "watch"

// kubeSpanName returns a Kubernetes API request's span name, and whether it is
// traced: watches, leases and events are not.
func kubeSpanName(r *http.Request, prefix string) (string, bool) {
	req := parseKubeRequest(r, prefix)
	if req.verb == kubeWatch || req.resource == "leases" || req.resource == "events" {
		return "", false
	}
	name := "k8s " + req.verb + " " + req.resource
	if req.subresource != "" {
		name += "/" + req.subresource
	}
	return name, true
}

// kubeAttributes returns a Kubernetes API request's attributes and url.full.
func kubeAttributes(r *http.Request, prefix string) []attribute.KeyValue {
	return append(parseKubeRequest(r, prefix).attributes(), semconv.URLFull(SanitizeURL(r.URL.String())))
}

// kubeRequest is what a Kubernetes API request's method and path name.
type kubeRequest struct {
	verb        string
	group       string
	namespace   string
	resource    string
	name        string
	subresource string
}

// parseKubeRequest reads a Kubernetes API request's verb and object from its
// method and path, after prefix: /api/v1[/namespaces/<ns>]/<resource>[/<name>[/<sub>]]
// or /apis/<group>/<version>[/namespaces/<ns>]/<resource>[/<name>[/<sub>]].
// Only the first segment of <sub> is kept: a proxy subresource's path is
// arbitrary. It allocates nothing.
func parseKubeRequest(r *http.Request, prefix string) kubeRequest {
	var req kubeRequest
	path, _ := strings.CutPrefix(strings.Trim(r.URL.Path, "/"), prefix)
	head, remain, _ := strings.Cut(path, "/")
	switch head {
	case "api":
		_, remain, _ = strings.Cut(remain, "/") // version
	case "apis":
		req.group, remain, _ = strings.Cut(remain, "/")
		_, remain, _ = strings.Cut(remain, "/") // version
	default:
		remain = ""
	}
	if remain == "" {
		req.verb = lowerMethod(r.Method)
		req.resource = "other"
		return req
	}
	// The deprecated /watch/ prefix, as in /api/v1/watch/namespaces/<ns>/pods.
	watch := false
	if seg, after, ok := strings.Cut(remain, "/"); ok && seg == kubeWatch {
		watch, remain = true, after
	}
	if seg, after, _ := strings.Cut(remain, "/"); seg == "namespaces" {
		// namespaces/<name>/status and /finalize are a Namespace's subresources.
		if ns, objPath, ok := strings.Cut(after, "/"); ok && objPath != "status" && objPath != "finalize" {
			req.namespace, remain = ns, objPath
		}
	}
	req.resource, remain, _ = strings.Cut(remain, "/")
	req.name, remain, _ = strings.Cut(remain, "/")
	req.subresource, _, _ = strings.Cut(remain, "/")
	for query := r.URL.RawQuery; !watch && query != ""; {
		var param string
		param, query, _ = strings.Cut(query, "&")
		watch = param == "watch=true" || param == "watch=1"
	}
	switch r.Method {
	case http.MethodGet:
		switch {
		case watch:
			req.verb = kubeWatch
		case req.name == "":
			req.verb = "list"
		default:
			req.verb = "get"
		}
	case http.MethodPost:
		req.verb = "create"
	case http.MethodPut:
		req.verb = "update"
	case http.MethodPatch:
		req.verb = "patch"
	case http.MethodDelete:
		req.verb = "delete"
		if req.name == "" {
			req.verb = "deletecollection"
		}
	default:
		req.verb = lowerMethod(r.Method)
	}
	return req
}

// lowerMethod returns method in lower case: a non-resource request's verb.
func lowerMethod(method string) string {
	switch method {
	case http.MethodGet:
		return "get"
	case http.MethodPost:
		return "post"
	case http.MethodPut:
		return "put"
	case http.MethodPatch:
		return "patch"
	case http.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

// attributes returns the request's verb and object, and for a Kargo object,
// its Project and kargo.<kind>.
func (req kubeRequest) attributes() []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		VerbKey.String(req.verb),
		ResourceKey.String(req.resource),
	}
	if req.subresource != "" {
		attrs = append(attrs, SubresourceKey.String(req.subresource))
	}
	// A Secret's name is left out: it can say whose credentials it holds.
	if req.name != "" && req.resource != "secrets" {
		attrs = append(attrs, NameKey.String(req.name))
	}
	if req.group != "kargo.akuity.io" {
		if req.namespace != "" {
			attrs = append(attrs, NamespaceKey.String(req.namespace))
		}
		return attrs
	}
	if req.namespace != "" {
		attrs = append(attrs, ProjectKey.String(req.namespace))
	}
	if req.name == "" {
		return attrs
	}
	switch req.resource {
	case "freights":
		attrs = append(attrs, FreightKey.String(req.name))
	case "warehouses":
		attrs = append(attrs, WarehouseKey.String(req.name))
	case "stages":
		attrs = append(attrs, StageKey.String(req.name))
	case "promotions":
		attrs = append(attrs, PromotionKey.String(req.name))
	case "promotionrequests":
		attrs = append(attrs, PromotionRequestKey.String(req.name))
	case "projects":
		attrs = append(attrs, ProjectKey.String(req.name))
	}
	return attrs
}
