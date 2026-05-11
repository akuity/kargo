package git

import (
	"errors"
	"fmt"
	"strings"

	libExec "github.com/akuity/kargo/pkg/exec"
)

// signatureStatus represents the trust level of a commit's GPG signature.
type signatureStatus int

const (
	// signatureUnsigned indicates the commit has no GPG signature.
	signatureUnsigned signatureStatus = iota
	// signatureTrusted indicates the commit is signed by a trusted key.
	signatureTrusted
	// signatureUntrusted indicates the commit is signed but the key is
	// not trusted (or the signature is invalid).
	signatureUntrusted
)

// CommitSignatureInfo holds signature details for a single commit.
type CommitSignatureInfo struct {
	// Trusted indicates whether the commit is signed by a key with
	// ultimate trust.
	Trusted bool
	// SignerName is the name from the signing key's UID. Empty if the
	// commit is unsigned.
	SignerName string
	// SignerEmail is the email from the signing key's UID. Empty if the
	// commit is unsigned.
	SignerEmail string
}

// GetCommitSignatureInfo returns signature information for the specified
// commit, including whether it is signed by a trusted key and the identity
// of the signer.
func (w *workTree) GetCommitSignatureInfo(
	commitID string,
) (*CommitSignatureInfo, error) {
	// A single git log call retrieves trust status (%G?) and the signer
	// identity (%GS, formatted as "Name <email>") separated by a tab.
	cmd := w.buildGitCommand("log", "-1", "--format=%G?\t%GS", commitID, "--")
	res, err := libExec.Exec(cmd)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting signature info for commit %s: %w",
			commitID, err,
		)
	}

	parts := strings.SplitN(strings.TrimSpace(string(res)), "\t", 2)
	info := &CommitSignatureInfo{}
	if len(parts) >= 1 {
		info.Trusted = parts[0] == "G"
	}
	if len(parts) >= 2 {
		info.SignerName, info.SignerEmail = parseSignerIdentity(parts[1])
	}
	return info, nil
}

// parseSignerIdentity parses a GPG signer string in the format
// "Name <email>" into separate name and email components.
func parseSignerIdentity(signer string) (string, string) {
	// Format: "Name <email>"
	lt := strings.LastIndex(signer, "<")
	gt := strings.LastIndex(signer, ">")
	if lt < 0 || gt < 0 || gt <= lt {
		return strings.TrimSpace(signer), ""
	}
	return strings.TrimSpace(signer[:lt]), signer[lt+1 : gt]
}

// verifyCommitSignature checks the GPG signature status of the specified
// commit. It uses git's %G? format which returns:
//
//   - G: good (valid) signature from a trusted key
//   - U: good signature from an untrusted key
//   - B: bad signature
//   - X: good signature that has expired
//   - Y: good signature made by an expired key
//   - R: good signature made by a revoked key
//   - E: signature cannot be checked (missing key)
//   - N: no signature
func (w *workTree) verifyCommitSignature(
	commitID string,
) (signatureStatus, error) {
	cmd := w.buildGitCommand("log", "-1", "--format=%G?", commitID, "--")
	res, err := libExec.Exec(cmd)
	if err != nil {
		return signatureUnsigned, fmt.Errorf(
			"error checking signature of commit %s: %w",
			commitID, err,
		)
	}
	switch strings.TrimSpace(string(res)) {
	case "G":
		return signatureTrusted, nil
	case "N", "":
		return signatureUnsigned, nil
	default:
		// U, B, X, Y, R, E — all treated as untrusted.
		return signatureUntrusted, nil
	}
}

// isSigningConfigured returns true if GPG commit signing is enabled in the
// git configuration for this repository.
func (w *workTree) isSigningConfigured() (bool, error) {
	cmd := w.buildGitCommand("config", "--get", "commit.gpgSign")
	res, err := libExec.Exec(cmd)
	if err != nil {
		var exitErr *libExec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode == 1 {
			// Exit code 1 means the key was not found — signing is
			// not configured.
			return false, nil
		}
		return false, fmt.Errorf(
			"error reading commit.gpgSign config: %w", err,
		)
	}
	return strings.TrimSpace(string(res)) == "true", nil
}
