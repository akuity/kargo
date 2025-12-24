package git

type MockRepo struct {
	AddAllFn          func() error
	AddAllAndCommitFn func(
		message string,
		commitOpts *CommitOptions,
	) error
	CleanFn                   func() error
	ClearFn                   func() error
	CloseFn                   func() error
	CheckoutFn                func(branch string) error
	CommitFn                  func(message string, opts *CommitOptions) error
	CreateChildBranchFn       func(branch string) error
	CreateOrphanedBranchFn    func(branch string) error
	CurrentBranchFn           func() (string, error)
	DeleteBranchFn            func(branch string) error
	DirFn                     func() string
	HasDiffsFn                func() (bool, error)
	HomeDirFn                 func() string
	GetDiffPathsForCommitIDFn func(commitID string) ([]string, error)
	IsAncestorFn              func(parent string, child string) (bool, error)
	IsRebasingFn              func() (bool, error)
	LastCommitIDFn            func() (string, error)
	ListTagsFn                func() ([]TagMetadata, error)
	ListCommitsFn             func(limit, skip uint) ([]CommitMetadata, error)
	CommitMessageFn           func(id string) (string, error)
	PushFn                    func(*PushOptions) error
	RefsHaveDiffsFn           func(commit1 string, commit2 string) (bool, error)
	RemoteBranchExistsFn      func(branch string) (bool, error)
	ResetHardFn               func() error
	URLFn                     func() string
	UpdateSubmodulesFn        func() error
}

func (m *MockRepo) AddAll() error {
	return m.AddAllFn()
}

func (m *MockRepo) AddAllAndCommit(
	message string,
	commitOpts *CommitOptions,
) error {
	return m.AddAllAndCommitFn(message, commitOpts)
}

func (m *MockRepo) Clean() error {
	return m.CleanFn()
}

func (m *MockRepo) Clear() error {
	return m.ClearFn()
}

func (m *MockRepo) Close() error {
	if m.CloseFn == nil {
		return nil
	}
	return m.CloseFn()
}

func (m *MockRepo) Checkout(branch string) error {
	return m.CheckoutFn(branch)
}

func (m *MockRepo) Commit(message string, opts *CommitOptions) error {
	return m.CommitFn(message, opts)
}

func (m *MockRepo) CreateChildBranch(branch string) error {
	return m.CreateChildBranchFn(branch)
}

func (m *MockRepo) CreateOrphanedBranch(branch string) error {
	return m.CreateOrphanedBranchFn(branch)
}

func (m *MockRepo) CurrentBranch() (string, error) {
	return m.CurrentBranchFn()
}

func (m *MockRepo) DeleteBranch(branch string) error {
	return m.DeleteBranchFn(branch)
}

func (m *MockRepo) Dir() string {
	return m.DirFn()
}

func (m *MockRepo) HasDiffs() (bool, error) {
	return m.HasDiffsFn()
}

func (m *MockRepo) HomeDir() string {
	return m.HomeDirFn()
}

func (m *MockRepo) GetDiffPathsForCommitID(
	commitID string,
) ([]string, error) {
	return m.GetDiffPathsForCommitIDFn(commitID)
}

func (m *MockRepo) IsAncestor(parent string, child string) (bool, error) {
	return m.IsAncestorFn(parent, child)
}

func (m *MockRepo) IsRebasing() (bool, error) {
	return m.IsRebasingFn()
}

func (m *MockRepo) LastCommitID() (string, error) {
	return m.LastCommitIDFn()
}

func (m *MockRepo) ListTags() ([]TagMetadata, error) {
	return m.ListTagsFn()
}

func (m *MockRepo) ListCommits(limit, skip uint) ([]CommitMetadata, error) {
	return m.ListCommitsFn(limit, skip)
}

func (m *MockRepo) CommitMessage(id string) (string, error) {
	return m.CommitMessageFn(id)
}

func (m *MockRepo) Push(opts *PushOptions) error {
	return m.PushFn(opts)
}

func (m *MockRepo) RefsHaveDiffs(
	commit1 string,
	commit2 string,
) (bool, error) {
	return m.RefsHaveDiffsFn(commit1, commit2)
}

func (m *MockRepo) RemoteBranchExists(branch string) (bool, error) {
	return m.RemoteBranchExistsFn(branch)
}

func (m *MockRepo) ResetHard() error {
	return m.ResetHardFn()
}

func (m *MockRepo) URL() string {
	return m.URLFn()
}

func (m *MockRepo) UpdateSubmodules() error {
	return m.UpdateSubmodulesFn()
}
