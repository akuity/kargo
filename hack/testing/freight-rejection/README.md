# Freight Rejection Test Resources

Apply this fixture from the freight-rejection worktree after Tilt has deployed
the branch's CRDs and control plane:

```shell
cd /Users/jacob/dev/github.com/akuity/kargo-freight-rejection
./hack/testing/freight-rejection/apply.sh
```

Tilt should also be started from this same directory:

```shell
cd /Users/jacob/dev/github.com/akuity/kargo-freight-rejection
make hack-tilt-up
```

The fixture creates the `freight-rejection` Project with four Stages:

- `manual-lab`: manual approve, reject, clear rejection, and promote flow.
- `reject-lab`: auto-promotion should skip rejected `frontend-v003-rejected`
  and choose `frontend-v002-fallback`.
- `hold-resume-lab`: an active auto-promotion hold from
  `akuity/kargo#6334`; resuming the hold should still skip rejected
  `frontend-v003-rejected`.
- `pending-promo-lab`: a pending Promotion for rejected Freight that should be
  aborted before it runs.

## CLI Setup

```shell
make build-cli
bin/kargo-darwin-arm64 login http://localhost:30081 --admin --password admin
```

## User Stories

### Reject and Clear Manually

```shell
bin/kargo-darwin-arm64 reject freight \
  --project freight-rejection \
  --alias manual-good \
  --reason "manual rejection walkthrough"

bin/kargo-darwin-arm64 approve \
  --project freight-rejection \
  --freight-alias manual-good \
  --stage manual-lab

bin/kargo-darwin-arm64 unreject freight \
  --project freight-rejection \
  --alias manual-good

bin/kargo-darwin-arm64 approve \
  --project freight-rejection \
  --freight-alias manual-good \
  --stage manual-lab
```

The first approval should fail while the Freight is rejected. The second should
succeed after the rejection is cleared.

### Reject Already Approved Freight

`manual-approve-then-reject` starts approved for `manual-lab`. Reject it, then
try to promote it:

```shell
bin/kargo-darwin-arm64 reject freight \
  --project freight-rejection \
  --alias manual-approve-then-reject \
  --reason "approved but known bad"

bin/kargo-darwin-arm64 promote \
  --project freight-rejection \
  --freight-alias manual-approve-then-reject \
  --stage manual-lab
```

The promotion should be blocked even though the approval status still exists.
Clear the rejection and retry to confirm the preserved approval works again.

### Rejected Newest Candidate Is Skipped

`frontend-v003-rejected` is newer than `frontend-v002-fallback`, but starts
rejected. `reject-lab` should auto-promote the fallback:

```shell
kubectl get promotions -n freight-rejection --watch
kubectl get stage reject-lab -n freight-rejection -o jsonpath='{.status.freightSummary}{"\n"}'
```

Look for an auto Promotion targeting
`fb3165f8155d864b6133171bb4e1dcb10611575a`, not
`fb556073349ba968d34d884ce7e3c47b3ee407a1`.

### Resume Hold With Rejected Candidate Present

`hold-resume-lab` starts with an active hold for `Warehouse/frontend`. Resume it:

```shell
bin/kargo-darwin-arm64 resume-auto-promotion \
  --project freight-rejection \
  --stage hold-resume-lab \
  --origin Warehouse/frontend
```

The resumed auto-promotion should pick `frontend-v002-fallback`, not the
rejected newest Freight.

### Pending Promotion Is Aborted

The fixture creates `pending-promo-lab.rejected-freight` for rejected Freight.
The Promotion controller should mark it `Aborted` before any step runs:

```shell
kubectl get promotion pending-promo-lab.rejected-freight \
  -n freight-rejection \
  -o jsonpath='{.status.phase}{" "}{.status.message}{"\n"}'
```
