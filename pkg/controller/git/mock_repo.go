package git

import "context"

type MockRepo struct {
	AddAllFn          func(ctx context.Context) error
	AddAllAndCommitFn func(
		ctx context.Context,
		message string,
		commitOpts *CommitOptions,
	) error
	CleanFn    func(ctx context.Context) error
	ClearFn    func(ctx context.Context) error
	CloseFn    func(ctx context.Context) error
	CheckoutFn func(ctx context.Context, branch string) error
	CommitFn   func(
		ctx context.Context,
		message string,
		opts *CommitOptions,
	) error
	CreateChildBranchFn    func(ctx context.Context, branch string) error
	CreateOrphanedBranchFn func(ctx context.Context, branch string) error
	CreateTagFn            func(
		ctx context.Context,
		tag string,
		msg string,
		opts *CreateTagOptions,
	) error
	CurrentBranchFn           func(ctx context.Context) (string, error)
	DeleteBranchFn            func(ctx context.Context, branch string) error
	DirFn                     func() string
	FetchFn                   func(ctx context.Context, opts *FetchOptions) error
	HasDiffsFn                func(ctx context.Context) (bool, error)
	HomeDirFn                 func() string
	GetDiffPathsForCommitIDFn func(
		ctx context.Context,
		commitID string,
	) ([]string, error)
	IsAncestorFn func(
		ctx context.Context,
		parent string,
		child string,
	) (bool, error)
	IsRebasingFn   func(ctx context.Context) (bool, error)
	LastCommitIDFn func(ctx context.Context) (string, error)
	ListTagsFn     func(ctx context.Context) ([]TagMetadata, error)
	ListCommitsFn  func(
		ctx context.Context,
		opts *ListCommitsOptions,
	) ([]CommitMetadata, error)
	CommitMessageFn          func(ctx context.Context, id string) (string, error)
	GetCommitSignatureInfoFn func(
		ctx context.Context,
		commitID string,
	) (*CommitSignatureInfo, error)
	IntegrateRemoteChangesFn func(context.Context, *IntegrationOptions) error
	PullFn                   func(context.Context, *PullOptions) error
	PushFn                   func(context.Context, *PushOptions) error
	RefsHaveDiffsFn          func(
		ctx context.Context,
		commit1 string,
		commit2 string,
	) (bool, error)
	RemoteBranchExistsFn func(ctx context.Context, branch string) (bool, error)
	ResetHardFn          func(ctx context.Context) error
	URLFn                func() string
	UpdateSubmodulesFn   func(ctx context.Context) error
}

func (m *MockRepo) AddAll(ctx context.Context) error {
	return m.AddAllFn(ctx)
}

func (m *MockRepo) AddAllAndCommit(
	ctx context.Context,
	message string,
	commitOpts *CommitOptions,
) error {
	return m.AddAllAndCommitFn(ctx, message, commitOpts)
}

func (m *MockRepo) Clean(ctx context.Context) error {
	return m.CleanFn(ctx)
}

func (m *MockRepo) Clear(ctx context.Context) error {
	return m.ClearFn(ctx)
}

func (m *MockRepo) Close(ctx context.Context) error {
	if m.CloseFn == nil {
		return nil
	}
	return m.CloseFn(ctx)
}

func (m *MockRepo) Checkout(ctx context.Context, branch string) error {
	return m.CheckoutFn(ctx, branch)
}

func (m *MockRepo) Commit(
	ctx context.Context,
	message string,
	opts *CommitOptions,
) error {
	return m.CommitFn(ctx, message, opts)
}

func (m *MockRepo) CreateChildBranch(ctx context.Context, branch string) error {
	return m.CreateChildBranchFn(ctx, branch)
}

func (m *MockRepo) CreateOrphanedBranch(ctx context.Context, branch string) error {
	return m.CreateOrphanedBranchFn(ctx, branch)
}

func (m *MockRepo) CreateTag(
	ctx context.Context,
	tag, msg string,
	opts *CreateTagOptions,
) error {
	return m.CreateTagFn(ctx, tag, msg, opts)
}

func (m *MockRepo) CurrentBranch(ctx context.Context) (string, error) {
	return m.CurrentBranchFn(ctx)
}

func (m *MockRepo) DeleteBranch(ctx context.Context, branch string) error {
	return m.DeleteBranchFn(ctx, branch)
}

func (m *MockRepo) Dir() string {
	return m.DirFn()
}

func (m *MockRepo) Fetch(ctx context.Context, opts *FetchOptions) error {
	return m.FetchFn(ctx, opts)
}

func (m *MockRepo) HasDiffs(ctx context.Context) (bool, error) {
	return m.HasDiffsFn(ctx)
}

func (m *MockRepo) HomeDir() string {
	return m.HomeDirFn()
}

func (m *MockRepo) GetDiffPathsForCommitID(
	ctx context.Context,
	commitID string,
) ([]string, error) {
	return m.GetDiffPathsForCommitIDFn(ctx, commitID)
}

func (m *MockRepo) IsAncestor(
	ctx context.Context,
	parent string,
	child string,
) (bool, error) {
	return m.IsAncestorFn(ctx, parent, child)
}

func (m *MockRepo) IsRebasing(ctx context.Context) (bool, error) {
	return m.IsRebasingFn(ctx)
}

func (m *MockRepo) LastCommitID(ctx context.Context) (string, error) {
	return m.LastCommitIDFn(ctx)
}

func (m *MockRepo) ListTags(ctx context.Context) ([]TagMetadata, error) {
	return m.ListTagsFn(ctx)
}

func (m *MockRepo) ListCommits(
	ctx context.Context,
	opts *ListCommitsOptions,
) ([]CommitMetadata, error) {
	return m.ListCommitsFn(ctx, opts)
}

func (m *MockRepo) CommitMessage(
	ctx context.Context,
	id string,
) (string, error) {
	return m.CommitMessageFn(ctx, id)
}

func (m *MockRepo) GetCommitSignatureInfo(
	ctx context.Context,
	commitID string,
) (*CommitSignatureInfo, error) {
	return m.GetCommitSignatureInfoFn(ctx, commitID)
}

func (m *MockRepo) IntegrateRemoteChanges(
	ctx context.Context,
	opts *IntegrationOptions,
) error {
	return m.IntegrateRemoteChangesFn(ctx, opts)
}

func (m *MockRepo) Pull(ctx context.Context, opts *PullOptions) error {
	return m.PullFn(ctx, opts)
}

func (m *MockRepo) Push(ctx context.Context, opts *PushOptions) error {
	return m.PushFn(ctx, opts)
}

func (m *MockRepo) RefsHaveDiffs(
	ctx context.Context,
	commit1 string,
	commit2 string,
) (bool, error) {
	return m.RefsHaveDiffsFn(ctx, commit1, commit2)
}

func (m *MockRepo) RemoteBranchExists(
	ctx context.Context,
	branch string,
) (bool, error) {
	return m.RemoteBranchExistsFn(ctx, branch)
}

func (m *MockRepo) ResetHard(ctx context.Context) error {
	return m.ResetHardFn(ctx)
}

func (m *MockRepo) URL() string {
	return m.URLFn()
}

func (m *MockRepo) UpdateSubmodules(ctx context.Context) error {
	return m.UpdateSubmodulesFn(ctx)
}
