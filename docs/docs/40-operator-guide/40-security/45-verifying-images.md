---
sidebar_label: Verifying Images
---

# Verifying Images

Akuity continuously scans the official Kargo image (`ghcr.io/akuity/kargo`) for
known vulnerabilities (CVEs) and publishes the results of that triage as
[OpenVEX](https://openvex.dev/) assessments. VEX ("Vulnerability Exploitability
eXchange") records, for each CVE a scanner might flag, whether the image is
actually affected — and when it is not, _why_ (for example, a vulnerable
function is present in a bundled library but is never reachable in this image).
Folding these assessments into your own scans drops the false positives Akuity
has already triaged.

The assessments are attached to **each image digest** as a signed
[cosign](https://github.com/sigstore/cosign) attestation (keyless, via Sigstore
/ GitHub OIDC), so they travel with the image — there is no extra endpoint to
configure, and the signature proves Akuity authored the assessment for exactly
the image you are running.

## Apply the assessments with Grype

Verify and extract the attestation, then pass it to
[Grype](https://github.com/anchore/grype) with `--vex`. Verifying first means
you only apply assessments whose Akuity signature checks out:

```bash
IMAGE=ghcr.io/akuity/kargo:<version>

# Verify Akuity's signature and extract the OpenVEX document in one step.
cosign verify-attestation --type openvex \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity https://github.com/akuityio/cve-triage/.github/workflows/attest-vex.yml@refs/heads/main \
  "$IMAGE" \
  | jq -r '.payload | @base64d | fromjson | .predicate' > kargo-vex.json

# Scan, applying the assessments. CVEs Akuity has marked not_affected are
# suppressed (Grype lists them under its ignored set with vex-status
# not_affected) and annotated with the justification.
grype "$IMAGE" --vex kargo-vex.json
```

A CVE is suppressed only when Akuity has dispositioned it **and** your scanner
still reports it — assessments are added as CVEs are triaged, so a freshly built
image may surface findings that are not yet dispositioned.

## Verify provenance only

To simply confirm that an image carries an authentic Akuity assessment (for
example, as a release gate), run the `verify-attestation` step on its own; it
exits non-zero if the signature or signer identity does not match:

```bash
cosign verify-attestation --type openvex \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity https://github.com/akuityio/cve-triage/.github/workflows/attest-vex.yml@refs/heads/main \
  ghcr.io/akuity/kargo:<version>
```

:::info Scope of the assessments

Akuity attaches an assessment to the **latest patch of each supported release
line** (e.g. the newest `1.10.x`). An assessment pertains to that specific
digest; older patches keep the assessment they received while they were current.
Suppression is keyed to the exact package versions present in the image, so an
assessment authored for one patch will not mis-apply to an image built from
different package versions.

The attestation is signed by Akuity's vulnerability-triage workflow, a distinct
identity from the workflow that builds and signs the Kargo release image itself.

:::
