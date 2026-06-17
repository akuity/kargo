---
sidebar_label: Verifying Images
---

# Verifying Images

Akuity continuously scans the official Kargo image
(`ghcr.io/akuity/kargo`) for known vulnerabilities (CVEs) and publishes the
results of that triage as [OpenVEX](https://openvex.dev/) documents. VEX
("Vulnerability Exploitability eXchange") records, for each CVE a scanner might
flag, whether the image is actually affected — and when it is not, _why_ (for
example, the vulnerable code is present in a bundled OS library but never in an
execute path).

This lets you cut false-positive noise from your own image scans: a CVE that
Akuity has assessed as `not_affected` carries a machine-readable justification
you can fold into your scanner's output instead of triaging it by hand.

There are two ways to consume these assessments.

## Apply VEX during scanning

The assessments are published as OpenVEX documents at `vex.akuity.io`, keyed by
image repository. This path works with any scanner that supports VEX and needs
no additional tooling. Fetch the document once, then pass it to your scanner —
CVEs that Akuity has assessed as `not_affected` are suppressed (and annotated
with their justification).

```bash
# Fetch Akuity's VEX document for the Kargo image.
curl -fsSL https://vex.akuity.io/pkg/oci/ghcr.io/akuity/kargo/vex.json -o kargo-vex.json
```

With [Grype](https://github.com/anchore/grype):

```bash
grype ghcr.io/akuity/kargo:<version> --vex kargo-vex.json
```

With [Trivy](https://github.com/aquasecurity/trivy):

```bash
trivy image --vex kargo-vex.json ghcr.io/akuity/kargo:<version>
```

The published document carries assessments for the latest patch of each
supported release line.

## Verify the signed attestation

Akuity also attaches the same OpenVEX assessment to each image **digest** as a
[cosign](https://github.com/sigstore/cosign) attestation, signed keylessly via
Sigstore (GitHub OIDC / Fulcio). This gives you cryptographic provenance: proof
that the assessment was produced by Akuity and pertains to exactly the image you
are running.

```bash
cosign verify-attestation --type openvex \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity https://github.com/akuityio/cve-triage/.github/workflows/attest-vex.yml@refs/heads/main \
  ghcr.io/akuity/kargo:<version>
```

A successful verification prints the signer identity and the attestation
payload. To inspect the assessment without verifying the signature, use
`cosign download attestation <image> | jq -r '.payload | @base64d | fromjson'`.

:::info

The attestation is signed by Akuity's vulnerability-triage workflow, which is a
distinct identity from the workflow that builds and signs the Kargo release
image itself. The assessments are appended on change — a digest keeps its
attestation until Akuity re-authors a disposition for it.

:::
