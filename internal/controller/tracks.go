package controller

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/common/config"
	"github.com/akuityio/k8sta/internal/git"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// trackReconciler reconciles Track resources.
type trackReconciler struct {
	config config.Config
	client client.Client
	argoDB db.ArgoDB
	logger *log.Logger
}

// SetupTrackReconcilerWithManager initializes a reconciler for Track resources
// and registers it with the provided Manager.
func SetupTrackReconcilerWithManager(
	ctx context.Context,
	config config.Config,
	mgr manager.Manager,
	argoDB db.ArgoDB,
) error {
	logger := log.New()
	logger.SetLevel(config.LogLevel)

	t := &trackReconciler{
		config: config,
		client: mgr.GetClient(),
		argoDB: argoDB,
		logger: logger,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Track{}).WithEventFilter(predicate.Funcs{
		DeleteFunc: func(event.DeleteEvent) bool {
			// We're not interested in any deletes
			return false
		},
	}).Complete(t)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (t *trackReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	t.logger.WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"name":      req.NamespacedName.Name,
	}).Debug("reconciling Track")

	// Find the Track
	track, err := t.getTrack(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if track == nil {
		// Ignore if not found. This can happen if the Track was deleted after the
		// current reconciliation request was issued.
		return ctrl.Result{}, nil
	}

	if track.Spec.GitRepositorySubscription != nil {
		err = t.syncGitRepo(ctx, track)
	}

	t.updateTrackStatus(ctx, track)
	// TODO: Make RequeueAfter configurable (via API, probably)
	return ctrl.Result{RequeueAfter: 30 * time.Second}, err
}

// updateTrackStatus updates the status subresource of the provided Track.
func (t *trackReconciler) updateTrackStatus(
	ctx context.Context,
	track *api.Track,
) {
	if err := t.client.Status().Update(ctx, track); err != nil {
		t.logger.WithFields(log.Fields{
			"namespace": track.Namespace,
			"name":      track.Name,
		}).Errorf("error updating Track status: %s", err)
	}
}

// getTrack returns a pointer to the Track resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func (t *trackReconciler) getTrack(
	ctx context.Context,
	namespacedName types.NamespacedName,
) (*api.Track, error) {
	track := api.Track{}
	if err := t.client.Get(ctx, namespacedName, &track); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			t.logger.WithFields(log.Fields{
				"namespace": namespacedName.Namespace,
				"name":      namespacedName.Name,
			}).Warn("Track not found")
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting Track %q in namespace %q",
			namespacedName.Name,
			namespacedName.Namespace,
		)
	}
	return &track, nil
}

func (t *trackReconciler) syncGitRepo(
	ctx context.Context,
	track *api.Track,
) error {
	logger := t.logger.WithFields(log.Fields{})

	repoCreds, err := getRepoCredentials(
		ctx,
		track.Spec.GitRepositorySubscription.RepoURL,
		t.argoDB,
	)
	if err != nil {
		return err
	}

	repo, err := git.Clone(
		ctx,
		track.Spec.GitRepositorySubscription.RepoURL,
		repoCreds,
	)
	if err != err {
		// TODO: Wrap this error?
		return err
	}
	defer repo.Close()
	logger.WithFields(log.Fields{
		"url": track.Spec.GitRepositorySubscription.RepoURL,
	}).Debug("cloned git repository")

	// Get the ID of the last commit
	mostRecentSHA, err := repo.LastCommitID()
	if err != nil {
		return err
	}

	// If there was no previous sync status, we cannot make a meaningful
	// comparison, so we'll simply establish a baseline here and return.
	if track.Status.GitSyncStatus == nil {
		t.logger.WithFields(log.Fields{
			"repo":   track.Spec.GitRepositorySubscription.RepoURL,
			"commit": mostRecentSHA,
		}).Debug("this is the first repository sync; nothing to compare")
		updateSyncStatus(track, mostRecentSHA)
		return nil
	}

	// If the most recent commit ID is the same as the commit ID recorded last
	// time we synced, then we're already fully synced.
	if mostRecentSHA == track.Status.GitSyncStatus.Commit {
		t.logger.WithFields(log.Fields{
			"repo":   track.Spec.GitRepositorySubscription.RepoURL,
			"commit": mostRecentSHA,
		}).Debug("found no changes since the previous sync")
		updateSyncStatus(track, mostRecentSHA)
		return nil
	}

	// Were any of the commits since the last sync not authored by K8sTA itself?
	var diffContainsNonK8staAuthors bool
	// The following command returns results like these:
	//
	//   3a41b887f95870a8a7b2d419de64728fd70b896c Committer Name
	//   80ea8286d8b6f94c80f104abc45c726d4b2bda42 Committer Name
	//   4e17b9a53911a0dea2fb45664ac005eca1083655 Committer Name
	cmd := exec.Command( // nolint: gosec
		"git",
		"log",
		fmt.Sprintf("HEAD...%s", track.Status.GitSyncStatus.Commit),
		`--format="%H %an"`,
	)
	cmd.Dir = repo.WorkingDir() // We need to be in the root of the repo for this
	commitListBytes, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(
			err,
			"error getting listing commits between HEAD and commit %q",
			track.Status.GitSyncStatus.Commit,
		)
	}
	commits := strings.Split(
		strings.TrimSpace(string(commitListBytes)),
		"\n",
	)
	for _, commit := range commits {
		author := strings.SplitN(commit, " ", 2)[1]
		if author != "k8sta" {
			diffContainsNonK8staAuthors = true
			break
		}
	}

	// If the diffs don't include any non-k8sta authors, then k8sta already knows
	// about and has already applied all the changes we just discovered. This can
	// happen, for instance, if a Track is subscribed to an image repository and a
	// push to that repository has already triggered progression of a new image
	// along the Track and, in the wake of that, this sync procedure discovers
	// commits that k8sta made in the course of rolling out the new image AND no
	// one else has also made commits since the last sync.
	//
	// In a case such as the above, there is nothing to do except update sync
	// status.
	if !diffContainsNonK8staAuthors {
		t.logger.WithFields(log.Fields{
			"repo":           track.Spec.GitRepositorySubscription.RepoURL,
			"previousCommit": track.Status.GitSyncStatus.Commit,
			"currentCommit":  mostRecentSHA,
		}).Debug("found no changes that were not authored by k8sta")
		updateSyncStatus(track, mostRecentSHA)
		return nil
	}

	// Do any of the commits since the last sync contain changes to the base
	// configuration? Only changes to the base configuration have the potential
	// to affect every environment, and therefore only those changes are eligible
	// for a Ticket to be created that will progress the changes along the Track.
	// nolint: gosec
	cmd = exec.Command(
		"git",
		"diff",
		track.Status.GitSyncStatus.Commit,
		"--name-only",
	)
	cmd.Dir = repo.WorkingDir() // We need to be in the root of the repo for this
	changedFilesBytes, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(
			err,
			"error getting diffs between HEAD and commit %q",
			track.Status.GitSyncStatus.Commit,
		)
	}
	changedFiles := strings.Split(
		strings.TrimSpace(string(changedFilesBytes)),
		"\n",
	)
	var diffContainsBaseChanges bool
	for _, changedFile := range changedFiles {
		if strings.HasPrefix(changedFile, "base/") {
			diffContainsBaseChanges = true
			break
		}
	}

	// If we didn't find any changes eligible to be progressed along the Track,
	// just update sync status and return.
	if !diffContainsBaseChanges {
		t.logger.WithFields(log.Fields{
			"repo":           track.Spec.GitRepositorySubscription.RepoURL,
			"previousCommit": track.Status.GitSyncStatus.Commit,
			"currentCommit":  mostRecentSHA,
		}).Debug("found no changes affecting base configuration")
		updateSyncStatus(track, mostRecentSHA)
		return nil
	}

	// Create a Ticket
	ticket := api.Ticket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uuid.NewV4().String(),
			Namespace: track.Namespace,
		},
		Track: track.Name,
		Change: api.Change{
			BaseConfiguration: &api.BaseConfigurationChange{
				Commit: mostRecentSHA,
			},
		},
	}
	if err := t.client.Create(
		ctx,
		&ticket,
		&client.CreateOptions{},
	); err != nil {
		return errors.Wrapf(
			err,
			"error creating new Ticket for base config changes observed in repo "+
				"%q between commits %q and %q",
			track.Spec.GitRepositorySubscription.RepoURL,
			track.Status.GitSyncStatus.Commit,
			mostRecentSHA,
		)
	}
	t.logger.WithFields(log.Fields{
		"name":           ticket.Name,
		"track":          ticket.Track,
		"namespace":      ticket.Namespace,
		"previousCommit": track.Status.GitSyncStatus.Commit,
		"currentCommit":  mostRecentSHA,
	}).Debug("Created Ticket resource")

	// Update status
	updateSyncStatus(track, mostRecentSHA)

	return nil
}

func updateSyncStatus(track *api.Track, commit string) {
	track.Status.GitSyncStatus = &api.GitSyncStatus{
		Commit: commit,
		Time:   &metav1.Time{Time: time.Now().UTC()},
	}
}
