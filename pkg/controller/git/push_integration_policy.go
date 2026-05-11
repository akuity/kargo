package git

// PushIntegrationPolicy controls how remote changes are integrated into the
// local branch before pushing. The four options form a spectrum from least
// conservative (AlwaysRebase) to most conservative (AlwaysMerge).
type PushIntegrationPolicy string

const (
	// PushIntegrationPolicyNone skips remote change integration entirely. This
	// is used internally when integration is not applicable (e.g. tag pushes
	// or force pushes).
	PushIntegrationPolicyNone PushIntegrationPolicy = "None"

	// PushIntegrationPolicyAlwaysRebase unconditionally uses pull --rebase to
	// integrate remote changes. This is the least secure option because it may
	// re-sign commits that Kargo did not author or strip existing signatures.
	PushIntegrationPolicyAlwaysRebase PushIntegrationPolicy = "AlwaysRebase"

	// PushIntegrationPolicyRebaseOrMerge uses pull --rebase when the
	// signature-trust decision matrix determines it is safe, and falls back to
	// a merge commit otherwise. This preserves linear history when possible
	// without undermining trust.
	PushIntegrationPolicyRebaseOrMerge PushIntegrationPolicy = "RebaseOrMerge"

	// PushIntegrationPolicyRebaseOrFail uses pull --rebase when the
	// signature-trust decision matrix determines it is safe, and fails the
	// push otherwise. This puts constraints on promotion process design and
	// treats the failure scenario as worthy of human investigation.
	PushIntegrationPolicyRebaseOrFail PushIntegrationPolicy = "RebaseOrFail"

	// PushIntegrationPolicyAlwaysMerge unconditionally uses a merge commit to
	// integrate remote changes. This is the most conservative option — it
	// never touches existing commits and always preserves original signatures.
	PushIntegrationPolicyAlwaysMerge PushIntegrationPolicy = "AlwaysMerge"
)
