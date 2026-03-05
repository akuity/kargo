package git

type SigningKeyType string

const SigningKeyTypeGPG SigningKeyType = "gpg"

// SigningKeyTrustLevel represents the GPG ownertrust level to assign to an imported
// signing key. The trust level determines how git reports signature
// verification status (%G? format):
//
//   - Unknown (zero value): no ownertrust is set; git reports "U"
//   - Full: the key is fully trusted; git reports "U" for bare keys, "G" only
//     if the key has been certified by another ultimately trusted key
//   - Ultimate: the key is unconditionally trusted; git reports "G"
//
// For standalone keys imported without certifications (Kargo's typical case),
// only ultimate trust produces "G". Full trust is useful when the key has
// been cross-signed by an ultimately trusted key (e.g. Kargo's system key
// certifies a user-provided key).
type SigningKeyTrustLevel string

const (
	// SigningKeyTrustLevelUnknown is the zero value — no ownertrust is set after
	// import. GPG defaults to "unknown" trust (level 2), and git reports
	// signature status "U" (untrusted).
	SigningKeyTrustLevelUnknown SigningKeyTrustLevel = ""

	// SigningKeyTrustLevelFull marks the key as fully trusted (GPG ownertrust
	// level 5). For a bare key this still produces "U" in git, but if the
	// key has been certified by an ultimately trusted key, git reports "G".
	SigningKeyTrustLevelFull SigningKeyTrustLevel = "full"

	// SigningKeyTrustLevelUltimate marks the key as ultimately trusted (GPG
	// ownertrust level 6). Use this for Kargo's system-level signing key
	// so that github-verified-push can identify commits signed by it.
	SigningKeyTrustLevelUltimate SigningKeyTrustLevel = "ultimate"
)

// gpgOwntrustLevel maps a SigningKeyTrustLevel value to the numeric level used
// by `gpg --import-ownertrust`. Returns "" for the zero value (no trust).
func gpgOwntrustLevel(trust SigningKeyTrustLevel) string {
	switch trust {
	case SigningKeyTrustLevelFull:
		return "5"
	case SigningKeyTrustLevelUltimate:
		return "6"
	default:
		return ""
	}
}
