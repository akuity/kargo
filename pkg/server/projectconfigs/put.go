package projectconfigs

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/request"
)

// @id UpdateProjectConfig
// @Summary Create or update ProjectConfig
// @Description Create a project's ProjectConfig, or replace its labels,
// @Description annotations and spec if it exists. The body's metadata.name
// @Description and metadata.namespace, if given, must be the project. A
// @Description metadata.uid or metadata.resourceVersion in the body is a
// @Description precondition of an update: it is refused if the ProjectConfig
// @Description no longer matches it. Status, the refresh annotation and other
// @Description server-managed metadata are ignored.
// @Tags Core, Project-Level, Config, Singleton
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body Resource true "ProjectConfig resource"
// @Success 200 {object} Resource "Updated ProjectConfig resource"
// @Success 201 {object} Resource "Created ProjectConfig resource"
// @Router /v1beta1/projects/{project}/config [put]
func (h *Handler) put(c *gin.Context) {
	project, ok := projectOf(c)
	if !ok {
		return
	}
	var obj map[string]any
	if !request.BindJSON(c, &obj) {
		return
	}
	cfg, decodeErrs := Decode(obj)
	if cfg != nil {
		// The URL names the ProjectConfig, so the body need not.
		if cfg.Name == "" {
			cfg.Name = project.Name
		}
		if cfg.Namespace == "" {
			cfg.Namespace = project.Name
		}
	}
	out, created, err := h.upsert(
		c.Request.Context(),
		project,
		cfg,
		decodeErrs,
		UpsertOptions{},
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	code := http.StatusOK
	if created {
		code = http.StatusCreated
	}
	c.JSON(code, ResponseBody(out))
}

// Upsert creates the Project's ProjectConfig, or updates it if it exists,
// and returns it as stored. created reports which happened. The caller must
// hold the get verb on the ProjectConfig, to learn whether it exists, and
// then the create or update verb. The ProjectConfig is validated as the
// Kubernetes API server and the admission webhook would have validated it.
func (h *Handler) Upsert(
	ctx context.Context,
	project *kargoapi.Project,
	cfg *kargoapi.ProjectConfig,
) (out *kargoapi.ProjectConfig, created bool, err error) {
	if cfg == nil {
		return nil, false, errNoProjectConfig
	}
	return h.upsert(ctx, project, cfg, nil, UpsertOptions{})
}

// errNoProjectConfig reports a call to Upsert without a ProjectConfig.
var errNoProjectConfig = errors.New("no ProjectConfig to write")

// errNoLongerExists reports a write carrying a uid or resourceVersion of a
// ProjectConfig that does not exist.
var errNoLongerExists = errors.New(
	"the ProjectConfig no longer exists; reload it and try again",
)

// UpsertOptions adjusts UpsertObject to the semantics of the endpoint a
// ProjectConfig arrived through. The zero value is the semantics of PUT on
// the ProjectConfig's own route. At most one of CreateOnly and MustExist may
// be set.
type UpsertOptions struct {
	// CreateOnly refuses to write a ProjectConfig that exists, as creating a
	// Kubernetes resource that exists is refused, with an AlreadyExists
	// error.
	CreateOnly bool
	// MustExist refuses to create a ProjectConfig that does not exist, with a
	// NotFound error.
	MustExist bool
	// SkipUserAuthorization writes the ProjectConfig without checking that
	// the caller may, using the API server's own authority instead.
	//
	// This is security-sensitive. It exists only for a Project the caller
	// created earlier in the same request to the generic resources
	// endpoints, as the project wizard does. Creating the Project made the
	// caller its owner, but the management controller grants the
	// corresponding RBAC asynchronously, so the caller's own permissions
	// cannot be relied on yet. This mirrors the existing exception that
	// writes other resources in such a Project with the internal Kubernetes
	// client. It must never be set for any other Project. Validation still
	// applies.
	SkipUserAuthorization bool
}

// UpsertObject is Upsert for a ProjectConfig not yet decoded from the
// generic map encoding/json produces. It decodes the ProjectConfig with
// Decode, so that it is checked exactly as PUT on the ProjectConfig's own
// route checks it, and writes it as opts direct.
func (h *Handler) UpsertObject(
	ctx context.Context,
	project *kargoapi.Project,
	obj map[string]any,
	opts UpsertOptions,
) (out *kargoapi.ProjectConfig, created bool, err error) {
	cfg, decodeErrs := Decode(obj)
	return h.upsert(ctx, project, cfg, decodeErrs, opts)
}

// upsert is UpsertObject for a ProjectConfig in which decoding found
// decodeErrs. A ProjectConfig that could not be decoded at all is nil. The
// errors are reported, with those validation finds, only once the caller is
// known to be authorized, so that no one learns anything of a ProjectConfig
// they may not write.
func (h *Handler) upsert(
	ctx context.Context,
	project *kargoapi.Project,
	cfg *kargoapi.ProjectConfig,
	decodeErrs field.ErrorList,
	opts UpsertOptions,
) (*kargoapi.ProjectConfig, bool, error) {
	authorize := h.authorizerFor(opts)
	if err := authorize(ctx, can.Get().ProjectConfig(project.Name)); err != nil {
		return nil, false, err
	}
	_, err := h.store.Get(ctx, project.Name, string(project.UID))
	switch {
	case errors.Is(err, database.ErrNotFound):
		if opts.MustExist {
			return nil, false, storeError(err, project.Name)
		}
		return h.create(ctx, authorize, project, cfg, decodeErrs, opts)
	case err != nil:
		return nil, false, storeError(err, project.Name)
	case opts.CreateOnly:
		return nil, false, storeError(database.ErrAlreadyExists, project.Name)
	}
	out, err := h.update(ctx, authorize, project, cfg, decodeErrs)
	return out, false, err
}

// authorizerFor returns the function that checks the caller may write a
// ProjectConfig as opts direct: the Handler's own, unless opts skip the
// caller's authorization.
func (h *Handler) authorizerFor(opts UpsertOptions) func(context.Context, can.Access) error {
	if opts.SkipUserAuthorization {
		return func(context.Context, can.Access) error { return nil }
	}
	return h.authorize
}

// create creates the Project's ProjectConfig. If another request created it
// first, it is updated instead, provided the caller may update it, unless
// opts allow only a create.
func (h *Handler) create(
	ctx context.Context,
	authorize func(context.Context, can.Access) error,
	project *kargoapi.Project,
	cfg *kargoapi.ProjectConfig,
	decodeErrs field.ErrorList,
	opts UpsertOptions,
) (*kargoapi.ProjectConfig, bool, error) {
	// A uid or resourceVersion is a precondition on a ProjectConfig the
	// client read earlier. If it no longer exists, the client's copy is
	// stale, and writing it would silently re-create what someone deleted.
	// The caller already holds the get verb, so this discloses nothing.
	if cfg != nil && (cfg.UID != "" || cfg.ResourceVersion != "") {
		return nil, false, apierrors.NewConflict(
			projectConfigsResource,
			project.Name,
			errNoLongerExists,
		)
	}
	if err := authorize(ctx, can.Create().ProjectConfig(project.Name)); err != nil {
		return nil, false, err
	}
	if err := check(ctx, project.Name, cfg, decodeErrs, true); err != nil {
		return nil, false, err
	}
	stored, err := h.store.Create(ctx, withoutRefresh(cfg), string(project.UID))
	if errors.Is(err, database.ErrAlreadyExists) && !opts.CreateOnly {
		out, updateErr := h.update(ctx, authorize, project, cfg, nil)
		return out, false, updateErr
	}
	if err != nil {
		return nil, false, storeError(err, project.Name)
	}
	logChange(ctx, "ProjectConfig created", project.Name)
	return projectconfigstore.ToAPI(stored), true, nil
}

// update replaces the labels, annotations and spec of the Project's
// ProjectConfig.
func (h *Handler) update(
	ctx context.Context,
	authorize func(context.Context, can.Access) error,
	project *kargoapi.Project,
	cfg *kargoapi.ProjectConfig,
	decodeErrs field.ErrorList,
) (*kargoapi.ProjectConfig, error) {
	if err := authorize(ctx, can.Update().ProjectConfig(project.Name)); err != nil {
		return nil, err
	}
	if err := check(ctx, project.Name, cfg, decodeErrs, false); err != nil {
		return nil, err
	}
	stored, err := h.store.Update(ctx, withoutRefresh(cfg), string(project.UID))
	if err != nil {
		return nil, storeError(err, project.Name)
	}
	logChange(ctx, "ProjectConfig updated", project.Name)
	return projectconfigstore.ToAPI(stored), nil
}

// check reports the errors decoding found in a ProjectConfig together with
// those validation finds, as one error. A ProjectConfig that could not be
// decoded is nil and is not validated.
//
// Validate checks the name and namespace only on create, because the
// Kubernetes API server refused a mismatch on update before the admission
// webhook saw it. Nothing does that here, so the same check applies to an
// update.
func check(
	ctx context.Context,
	project string,
	cfg *kargoapi.ProjectConfig,
	decodeErrs field.ErrorList,
	create bool,
) error {
	errs := slices.Clone(decodeErrs)
	if cfg != nil {
		if !create {
			errs = append(errs, validateObjectMeta(field.NewPath("metadata"), project, cfg)...)
		}
		errs = append(errs, Validate(ctx, project, cfg, create)...)
	}
	if len(errs) > 0 {
		return invalid(project, errs)
	}
	return nil
}

// withoutRefresh returns a copy of a ProjectConfig without the refresh
// annotation. A refresh is requested through its own endpoint, and a client
// that read the ProjectConfig and writes it back must not store the token
// it was served.
func withoutRefresh(cfg *kargoapi.ProjectConfig) *kargoapi.ProjectConfig {
	cfg = cfg.DeepCopy()
	delete(cfg.Annotations, kargoapi.AnnotationKeyRefresh)
	return cfg
}
