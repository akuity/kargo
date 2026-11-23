package server

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/projectconfigs"
)

// The generic resources endpoints write most resources to Kubernetes, but
// ProjectConfigs live in the database. A ProjectConfig in a manifest is
// handed to the same code that serves the ProjectConfig's own routes, which
// authorizes and validates it exactly as those routes do.

// errProjectConfigNoProject reports a ProjectConfig manifest that names no
// Project.
var errProjectConfigNoProject = libhttp.ErrorStr(
	"ProjectConfig has neither a metadata.namespace nor a metadata.name",
	http.StatusBadRequest,
)

// projectConfigHandler returns the Handler that serves the ProjectConfig
// routes, for the generic resources endpoints to write ProjectConfigs with.
func (s *server) projectConfigHandler() *projectconfigs.Handler {
	return projectconfigs.New(s.projectConfigs, s.authorize, s.natsConn)
}

// resourceClient returns the client with which the generic resources
// endpoints write obj: the authorizing client, unless obj is namespaced and
// belongs to a Project created earlier in the same request.
func (s *server) resourceClient(
	obj *unstructured.Unstructured,
	createdProjects map[string]*kargoapi.Project,
) client.Client {
	if _, ok := createdProjects[obj.GetNamespace()]; !ok {
		return s.client
	}
	// Kubernetes ignores metadata.namespace for a cluster-scoped kind, so a
	// spoofed value could bypass authorization there. Confirm real scope via
	// the RESTMapper before trusting it.
	if namespaced, err := s.client.IsObjectNamespaced(obj); err == nil && namespaced {
		// Creating the Project made the user its owner/admin, but RBAC for it
		// is wired up asynchronously by the management controller. Use the API
		// server's own permissions here so this isn't blocked by that race.
		return s.client.InternalClient()
	}
	return s.client
}

// recordCreatedProject records obj, a Project the generic resources
// endpoints just created, in createdProjects, keyed by name.
func recordCreatedProject(
	ctx context.Context,
	obj *unstructured.Unstructured,
	createdProjects map[string]*kargoapi.Project,
) {
	project := &kargoapi.Project{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		obj.Object,
		project,
	); err != nil {
		// Leaving the Project out only withholds the exception for Projects
		// created in the same request, so the request's other resources are
		// written with the caller's own permissions.
		logging.LoggerFromContext(ctx).Error(
			err, "error converting created Project",
			"project", obj.GetName(),
		)
		return
	}
	createdProjects[project.Name] = project
}

// projectOfConfig returns the Project a ProjectConfig manifest belongs to,
// which is named by its namespace or, failing that, its name, and whether
// the Project was created earlier in the same request. A Project created
// earlier in the request is the one its create returned. Any other is read
// with the internal client; one that does not exist is reported as
// NotFound, as the routes of a Project that does not exist report it.
func (s *server) projectOfConfig(
	ctx context.Context,
	obj *unstructured.Unstructured,
	createdProjects map[string]*kargoapi.Project,
) (*kargoapi.Project, bool, error) {
	name := obj.GetNamespace()
	if name == "" {
		name = obj.GetName()
	}
	if name == "" {
		return nil, false, errProjectConfigNoProject
	}
	if project, ok := createdProjects[name]; ok {
		return project, true, nil
	}
	// The Project is looked up with the API server's own client rather than
	// the authorizing one on purpose, as middleware.RequireProject does for
	// the ProjectConfig's own routes. Through the authorizing client, this
	// would also require permission to get the Project, which a user whose
	// access comes only from project-scoped RBAC does not have. This internal
	// client read discloses nothing but the Project's existence. The
	// ProjectConfig's own can checks still run afterwards, in the Handler.
	project := &kargoapi.Project{}
	if err := s.client.InternalClient().Get(
		ctx,
		client.ObjectKey{Name: name},
		project,
	); err != nil {
		return nil, false, err
	}
	return project, false, nil
}

// upsertProjectConfig writes the ProjectConfig in obj as opts direct and
// returns it as stored, in the form of a manifest, and whether it was
// created.
func (s *server) upsertProjectConfig(
	ctx context.Context,
	obj *unstructured.Unstructured,
	createdProjects map[string]*kargoapi.Project,
	opts projectconfigs.UpsertOptions,
) (map[string]any, bool, error) {
	project, createdInRequest, err := s.projectOfConfig(ctx, obj, createdProjects)
	if err != nil {
		return nil, false, err
	}
	// SECURITY: The caller's authorization is skipped only for a Project the
	// caller created earlier in this same request, for the reason the
	// internal client is used for other resources in such a Project (see
	// resourceClient): the management controller grants the caller RBAC on
	// the new Project asynchronously. createdProjects holds only Projects
	// this request created successfully.
	opts.SkipUserAuthorization = createdInRequest
	out, created, err := s.projectConfigHandler().UpsertObject(
		ctx,
		project,
		obj.Object,
		opts,
	)
	if err != nil {
		return nil, false, err
	}
	manifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(out)
	if err != nil {
		return nil, false, fmt.Errorf("error converting ProjectConfig: %w", err)
	}
	return manifest, created, nil
}

// createProjectConfig is createResource for a ProjectConfig.
func (s *server) createProjectConfig(
	ctx context.Context,
	obj *unstructured.Unstructured,
	createdProjects map[string]*kargoapi.Project,
) (createResourceResult, error) {
	manifest, _, err := s.upsertProjectConfig(
		ctx,
		obj,
		createdProjects,
		projectconfigs.UpsertOptions{CreateOnly: true},
	)
	if err != nil {
		return createResourceResult{
			Error: fmt.Errorf("create resource: %w", err).Error(),
		}, err
	}
	return createResourceResult{CreatedResourceManifest: manifest}, nil
}

// updateProjectConfig is updateResource for a ProjectConfig.
func (s *server) updateProjectConfig(
	ctx context.Context,
	obj *unstructured.Unstructured,
	createdProjects map[string]*kargoapi.Project,
	upsert bool,
) (createOrUpdateResourceResult, error) {
	manifest, created, err := s.upsertProjectConfig(
		ctx,
		obj,
		createdProjects,
		projectconfigs.UpsertOptions{MustExist: !upsert},
	)
	if err != nil {
		return createOrUpdateResourceResult{Error: err.Error()}, err
	}
	if created {
		return createOrUpdateResourceResult{CreatedResourceManifest: manifest}, nil
	}
	return createOrUpdateResourceResult{UpdatedResourceManifest: manifest}, nil
}

// deleteProjectConfig is deleteResource for a ProjectConfig.
func (s *server) deleteProjectConfig(
	ctx context.Context,
	obj *unstructured.Unstructured,
) (deleteResourceResult, error) {
	project, _, err := s.projectOfConfig(ctx, obj, nil)
	if err == nil {
		err = s.projectConfigHandler().Delete(ctx, project)
	}
	if err != nil {
		return deleteResourceResult{
			Error: fmt.Errorf("delete resource: %w", err).Error(),
		}, err
	}
	return deleteResourceResult{DeletedResourceManifest: obj.Object}, nil
}
