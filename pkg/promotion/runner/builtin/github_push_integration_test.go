//go:build integration && github

package builtin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-github/v76/github"
	"github.com/jferrl/go-githubauth"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	ghutil "github.com/akuity/kargo/pkg/github"
	gptest "github.com/akuity/kargo/pkg/gitprovider/testing"
	"github.com/akuity/kargo/pkg/promotion"
)

const (
	someoneElseName  = "Someone Else"
	someoneElseEmail = "someone@else.com"

	kargoName  = "Kargo Test Key"
	kargoEmail = "no-reply@kargo.io"

	// testSigningKey is a key used exclusively for testing. No one outside of
	// this test suite trusts it.
	testSigningKey = `-----BEGIN PGP PRIVATE KEY BLOCK-----

lQcYBGmyJr0BEADd/Zhbf+xKK6FZx8L69j8XJjo5dZprPNIabhams70U81QQx2wM
KQ+yX0RYtww2Tt70pTquT92furvRkTMtRp9ELFHzj/9g96AqdLFpS/R4y+3UXsP5
hhzZTN9W0cmVL4dL3V36oi6qfKybCk8RNoKsG2Q0COh2WM7nYKpI9ZiBCxDm7cwQ
ymo1HVNVy5oT5j5ERQOittod/TxllY1u+VLczevZQF8SO9tyHn8AI8qpHBoiB2PQ
f7xEin0LA10DBTy+E6d4H5att4/HzWO5fRIFKra+XN4naO2StsQ5hv1uKFkxi3oG
yQ1hhDPHojd8zJRIrzaWmtn00DoH0gWdB+QYCF3SCEiN29ADFC9syk3zGMpJ8d5P
yoGNmEoM3uAl5FZM9CKW8lTmCilM1E2otaWcs+YyiPIwxDrWgpxHivCK+U1pNWPI
+a0ebCW9O9DN/iKU5RlLDif+Cd4P4p5FIu2YRUaWIm9oY5u0snl1pjvnMdlV89Tf
ZNa/ByVbrdrVMgTk81YvT89QR73Haa7VehXQvgNTScUXNV81giL6LkBhHGSwLab4
u77QRnE4ADMxf1AA/qyrMce1KWK0EhXpCkpmJAs0QrnFEE/FgoUJsFInVaKVNWBO
6MLpxCgOfQxTYWBs8QpgnQZlQ3B9wGLCUJwy8nttpaX2XmchdXwQR9GVSQARAQAB
AA/7ByU6Cbvw4aRM4cRt0MErX7XhyuavrUL8alBf10bSz8FTU+TvY+bQdAff/dEK
ihb40zwcSu8ULaeHkyPO12a+CqY4jxPz/u2JkMRHz6FbwaWixqK0QSxhclcomzVO
fUhV3QnHlCEzSlaQAb+PsvijtSF+lLQys1iBdG4mnQmWupSeGyjNWD8Dsyj5/Tl8
AKb5Gx5zkwG6vJthnc12C3voAGZ6cHhDdyVJF3/Gy7zbMZ1PdAfz3Qq3hydEBh46
GLZK8b1VNychJP9TF/XS/234rgTlc/QuTGtytjW+1DE1qB2RXWhlaaGAFkL4nbTp
f1FgdoR5R9i9KkWnkIfgmWDfJR2JPBhfiPsinQ0SeFw1Q4tJZCG/5xLUD5SWBtYX
F/ZFX1rDlg4CRtqw8ZQp/OEUz7f0ssdDG3heCWysuXg9LyztdA59EK/tocGK+GVQ
WJJDPglCHNDB5ZFkSQ7YxZipGQuJC3k+6oe1KVm4ieW44Ybjw7PQK8nEDNFbtyek
61+bE4j8XPHG5IYBIKvbvLeByAq6r206Eh9c283PK1edWP9gP2WHaw0ScIBaU7mV
msNuJ3DBhbs+I3yOiQxu3L5VTqvAPF9VSbkZCho5ZdDjLWnreGJu2kwyqcYg5kss
RMN4XCDjcrcTdktEVzjArhZVKLNeb47+5gmfnw9d3BQ934EIAOc+bRA7j3aRdR8+
SE/l8cjsIVRP3LobHo3nzpbPB7P2ikuK531wPqur0F7VHlujqJcS0p6ZAnaV1aza
Cbt2hPB3Ec5z+Yjk1at+oEQdlW4voung/FTde35Y2CL8wCpDyqI50eduLG4aVPR1
4Lz4vAlkM/wNZIGgn+UAuME2el7K9S5yHtRrY6IQdSlZdLVv0gV8LqLYQpCGfzqz
VTkYqTOq85C143JPM2AMGqb04ARpIFvKYvpvsHkFByFXy/jqnrfquh9AxgFQPLL7
EDZaY34nxnWMGmRRh9Q4roxh/53bJWrijBf9Wj3HRnYweOFk9LyNtio1TVzt+h3X
XefYNykIAPXBkgod8Mhw9IpjltWXD4ElZ4V2kFfsTNxwBs7HwITMDiATEMW8j/OK
IpT0YPX8wnIWk8zAXaKXvXT1njtnZ+T2BGbcflClIRvS/Z+RAzaX1Nsace7If5jy
r9BmD98IniuvgOdIYDq3MC9frSaS7gXGbgxs0KIQIIpDJ+L8NtcOHC2DDIVcfDk6
MatCUObPbpwK2x8ByGUjFTg1jqeGbEb9PPlQ4kII0M4F9lft4LyBMnDjzPJ9fUr4
c7mY5YYcurvO6Ts9KT5bErMUvW5wmUtzAQCap451oijj+g2Z5CXs2DXuHOfMI2MH
JY+ApDT+7g1CAdInav2CiVaup4/a0SEIAMpGgFj1F+JbnCYv8IO4fLF6bPVeXFcR
5LAQA7zfmES0ghJUM6N0Ksr4cm7IMwQ9jmlcz3D/0rrBWFo38oFF5snYOsjmFXcT
jJr8cIh19RSAkjKdj5MKyQuEbtqWblHziW1q5SDIf+cUSaV+gOkO6M4gCXiKYIps
YLHoNGDDm7i/nxsFPQ6J5UXKU73LCBjRAKHiR+OLgBj9jZnGYor2Qw8AOgMKgWfT
GJBiuvYuhmpEIl9CTT8RJP7fqDw3taubJE5VCQFJ6aCebwThN+ms1I0+gfzDu5V1
SyKcbCtnQtSqqKHJbYwp6Gbh3jp1LiD5g3g8+chNgU6SBUhc6PNF7vN8RLQiS2Fy
Z28gVGVzdCBLZXkgPG5vLXJlcGx5QGthcmdvLmlvPokCUgQTAQgAPBYhBCRrxTni
jUkF34GjRl818mRiDs0nBQJpsia9AxsvBAULCQgHAgIiAgYVCgkICwIEFgIDAQIe
BwIXgAAKCRBfNfJkYg7NJ7YkD/9LYlG+kXwCi7CD1bkf1AAQH84ldvy3/u1ur5HV
LFTeabrVjr4ePT+oYwR8z9HHc3V9dlirMLUUyGxhAz4hmJLNtOLTXHF7aA2fUdh4
LLEAjVj+eT/uj9VPq7BmaySZIFndOPK5eNDQZEYShjpIJsLEoG1OS1UcsR04Yvrm
msxKu8w+vYmII0bOogpyiR+hlNWxvhWq4PmXR5dXCUOyVXUhb1lyDV/wNB1Yr7+e
1S9hvCvXkAZ2YfER6bs0dcwGyEjQSjMOLN4vjMrpf8dGPjQZGFqJGZsEvOJ7d/Z9
ixUxjEU1dOw622KsuvCW6bZforQvTksIGm9sQR1aMhaiqjdiADfMGEG8qvaI0AdF
WtVggG8BYFzyMOLfB74X74gM47fOVuuSLGEUN20vYyFIB99TMdDHTe//R4lBEyoh
/YjMZnSpV8iy9pnKNVvJYy1E6//P4IkZn8pGrYisrZekY5a6bq6ElJriBxg4WsBj
LPZG/sWEtQyXkUP1FS9xna8uLIFq6dd/2IDUheB13BnRDVrjX7Iqv8QscKEUS7m6
hVr01D+FxD/PBNxVqDQs+12cEIRV8JjzBZjcFAmP/nZRc887TrNQ2hOK2kypyjiF
Yie5CR9wlEoKVZN53ro8tnLI0uSLaA+bKlfPoL1tYrquvlNO4AdEWYl9AvBuChv2
iLHwN50HGARpsia9ARAAnHQFiuKoouRbDL5eTRwsrGyUVi8tUz/8bTaBh6mGPO0H
jTyhkCriyXWjfYciEza+80cx4SXnuhe1KLH/AuUZMO2yvxH4cEUrcWrUGbAXiw83
xvCJwLLX30eTlJBz1T+FcJsH09gGSeLKmv/tt/C7xjbt56V5Kt078jhgo218NKhC
zUtMBqF+wg+61pVy6ROif4fh5IACYFW//RIC9qbmqBh/rwe8SuONXjHQRPcImyVI
F/mrYCuuJ2B1S+i5m2P5zLIXDi9HiUogLltYh0zxv0Ll7DSPg+vVGP/dQ7DytZW4
h+QZNzpPGcgCDNEjlfAoMR4OP5KVwTSpRpPevus4pVhB/suTN2bMMPbmTuvVgYtl
fOhcUsWqwU8m0r2kNp6ChxQSTRulflR6TWUM4RnEr2wKLg/bVPTykbC/O71sLDMB
je6SKu8QnulW24TtG3ROPrLzX4bycisjw3tE1ewuPmVqzm5XPzL4mYKulKjv+L5s
oZP9lJ2XQ4FpEGmuZRuNq6IyrMsGOujRN7YYGL/t0rrBqC6NVd2n20j4X5JJS8Wj
SGse1j5zKR0L7eEnrJ6J+HnWmzyvJByG5uVytNZJmi7HJ8BmXxlC4fkQFDheH4G9
M3wRtE9zBVTeouEqZUkO4ofOeqLoIZSo43X63GA++6PGhq+6sdBLd1WRk0MW4wMA
EQEAAQAP+wSf2hpaxBbCXqh2xaO7UDIx269C3Zc9fSM0d/ibtU26QGQGZ6H/mwyb
pZ8jo3hFhR64ahKNrFbJnDTfiCTafreoY0EKKQIW7qVU112Svvfv4llPerDAlEZg
ZhqqARBcxOX16c+mdmRSsoZFJaf8I/YRLj7lEn8PQQDSMWIhIO/cLvVNSZkg1/Xb
0l4rXzuMCrHFHyzsmdiUlIzIe08kjg00+6PbZpxDAGjJOIyr+FjRXLB/qbtS/ABb
sXedQ7t6B6H8SM8bhoTq8NpN7Qk6xgCkhk3RHHBTVTN+s4uGEcNLV9UOC4zCsCBC
5HONu8k1Hj419UjssptJCJkyUuz6JKiDMcb8sgHxlyjTJ8xdDTuW27+sSKnOKgYA
TZDt6ro1sxzHlyJcI1D4sGuO4r1gJEfHHXZzCya8oYSvuiK8WSHai1Hl1e2vYJuC
YQRbDbl0Gh55JZHbtJ7gzWAyqkD4KdvVqwtk8B0GZXj4dfobtokbtwZhtAxU5oW+
Ql9mbZHw/M0lz4AO2HSgrS/7nb2EhdZmyBLUZpSA9vaJiW+TEqnEqMRiZei3xFmL
uyKsmAxk5h/k4JNsCFx3MfgMzd5FxcqkyfksvThC4f1Di2NkQufhM60tfCJo8jPL
Ver7hHDy51paYnR4XoKUSFHj86zNtYILqrd+dxt9+IyCfdP+Ay7ZCADFIVCaM+CA
YtvVIBnqMaVuq1VM4mry4tUYzF5nURMEh1Bg8bNz5n3d+hvNvHyrgcWOfeRpEYjG
u3+QJTHhF2DhDOjW0cjnzdb59YsaFXC0RDNgPK+kbtSNH3Cxf2yv9m7UCwITn6It
HQrto5cBXxcpskA8+utqCdbmTOaj8pNBFUlNVxIp5NOqSnh7VPl36Hw17uHe+MCu
pXaTktOjsWrdTSo/ji4u/Ygo6QibhRmeLcWRScloSn1qudhvsL6Q8aRzrxNx0Fpm
+0YGC1oUFSGBlEOHTSdLGQTf+I2eCX0Dl8OikIKO1b3iypDgyfh9Esg2mT+uvKoR
boRjau9ehJDPCADLLPAei6rlvK86adwWKOFysDeMs3OGMIFTuOeyW6+X1UQjsJ6U
WDlwLRSHlWUiMecZfxy7RaVorDtcupfM90giSWlouSR+BRa+psYsxbcSfUbqdv84
XBqR/hgxyYMZkoNFk+udkq1DaTH6Ohy77fM2Y0Bj4NVtSa0ifa0bZBvWvgUtzevi
iB8SGBvMbrqCi2nxZmXmKgtB4YSCXwMlooZUpAn2ybtRgowbUEtSZw26Axe8vrd6
Hda+IXz2QedFOHhGcZ91iXRw7WGm1EUwCvMCQPTypbYP7ThvvR4IV+IqD4xoFRLH
ddi4+5yLWH1LPhC3oy56fFlGNbgtEswWuQ+NB/oDb2TERqjo6OqDOATmCqrXvotJ
UGaxlR2gUcMcWTzZNlM9SuSxRnnFWtfzC070NKoskojKmGg1wrxU8aECZCpHY9zs
K5ojdVtFux1tNhI/iECnAKwTTmZpW4POyJ96gSjqXpOp5KV8D1bsg1Gxd8ERJrTy
maEbCq0JL0vE7jknUJrq7cPBdHAclledC9rHDkOO8P2hUK4mo+tF0F/IZVK5onwQ
vRpH5/XsrwHMcO7ccXMyKH5beylwYOQbMscPI5gNep6aV1grzKeXqUaFrG0FdEXu
mHUdLMzzh6oxXUIvpJlLmH9wCKXDbBbqpXGqyQ0NR2bsNeoAmARGf0WM/WylfPWJ
BGwEGAEIACAWIQQka8U54o1JBd+Bo0ZfNfJkYg7NJwUCabImvQIbLgJACRBfNfJk
Yg7NJ8F0IAQZAQgAHRYhBHVLkGQ1NkxEA8rb+4ObB2DoSD1FBQJpsia9AAoJEIOb
B2DoSD1FF/cP/jXm7ssd77zPC1GCZx0srX8JE2DrSRNp/utHtvk29YAbD5GK3Dxi
2t7rCMJGYPK0VTImC74TeAvABlPtt/K/vUl/4vVekSX0r9/4fhQH61S5S6BsfYpb
qYUmhRoOPeiEod0iPnE0eLPzS3FZDxENjJ9n29tZv2JgZpgKj1CaOJxGuhLwsE6m
jXFfwQmxSrRXQ+ZwR64Sr7VXG5cuFCYdvgPalyy2/2TzwBMx3CJODwm6tedP978f
St1p+tPLQO9pUcmi+c3bCZ5A8jYvhUIal9FqCkm+tx/c0CkDIZ57v2uznJa9C5Zt
gPtAo93ORf8D4Cc/nZw2BfQ1qxV1n/qyhWSu9fDobHCWznC/R2OTVrfn+BbgQste
fodnydxtgb63B2pwxG4QZ/t8OE6mf9v2dODmibQ299OwbJS6vRpqQiBAIxfLcrBy
DDLTa4946H3EcpJWKdgksKUXY59X8UxvL7XUH4CnSqpj8BK+8CAyWqb6vjgt5Fn2
lCibCfAH30Sq95tlKpIsq+rXI5TRnPz9R9efMxiCdjo1QJF56H13EUdINn5+Cky/
z8eZmJ6MbqmA84INEp3W6VY/31LVqSnkMsEqDQ3+qZ5goZBwazcjd1viMrMZIN5l
mQpGZukWZZle0EfUXrn54WkJKyECGU+31mte9Cl7ZqMvOKgGCEica34EVuUP/j6z
ZntpqxiJjw84b9l82rMvJ20cDLEm4EgjZHMfh3SeeD6ylfLYoSi27qciVGHZ3e6t
4swvutHNFcwbltRzpZXNIJ0Hjs0nE7ePkEAnHaRIkxPaPFuOauXbYdxErms0pTug
GGioZwMADO6zPxjLlTodo4PyRZWGScMRUh7IBLt6boTgUgsbAZO8bJr6U7lZCDZ3
r8wt7qjMkC1zkEs4VgQNDxJKFukVZ/zWkw1VDwlxgIHs2AVz7oEU0NxsBSlUKYRp
qNi7NWKF21zlrhm4T9WlKQTcuo0Ii+cRkTDhC+DIgwlmo3dB27pS9KuEp+BhXH5o
NPrGjir7AoxeJD0ObKChtRiQ0HXJj9mnSyf/C63t2DeIe/oH1Vo/9eRumzFivpRU
IVG/IAfemP7X8uy8+3xhC4ZQEP+xkFHOT9gvtiPBRyTOIxZYccUi1zh4eF0anD3q
I8yQHE6VozEm14cZn4O5cYfUZ4G6IwB9urVdi7Sg5HgAvMlOBIB5uS3UiKLp59q3
KwUt0H+hNxmroTmXfNLhHkv23DrN2kMbhHjQfUCRqslVOMTcKGiID1nG4gE8IbMU
PkJ7eicfM7txaKiF69jxkzIyVeCsfsLNZUThGxyE/qaVz9YEKjpMWGqv7+oIcUUW
rG3zXpnnzYbbLR5r+tEFfW0Hjk1Y08estSXIUE6J
=t/xA
-----END PGP PRIVATE KEY BLOCK-----`
)

func TestGitHubPush_Integration(t *testing.T) {
	repoURL := gptest.RequireEnv(t, "TEST_GITHUB_REPO_URL")

	_, _, repoOwner, repoName, err := ghutil.ParseRepoURL(repoURL)
	require.NoError(t, err)

	appClientID := gptest.RequireEnv(t, "TEST_GITHUB_APP_CLIENT_ID")
	appInstallationID := gptest.RequireEnv(t, "TEST_GITHUB_APP_INSTALLATION_ID")
	appPrivateKeyPath := gptest.RequireEnv(t, "TEST_GITHUB_APP_PRIVATE_KEY_PATH")
	appPrivateKey, err := os.ReadFile(appPrivateKeyPath)
	require.NoError(t, err)

	appInstallationToken := getAppInstallationToken(
		t,
		appClientID,
		appInstallationID,
		string(appPrivateKey),
	)

	ghClient, err := ghutil.NewClient(
		repoURL,
		&ghutil.ClientOptions{Token: appInstallationToken},
	)
	require.NoError(t, err)

	// This is a fake credentials DB that always return the same credentials (the
	// GitHub App's installation token).
	credsDB := &credentials.FakeDB{
		GetFn: func(
			context.Context,
			string,
			credentials.Type,
			string,
		) (*credentials.Credentials, error) {
			return &credentials.Credentials{
				Username: "kargo",
				Password: appInstallationToken,
			}, nil
		},
	}

	// Ensure the test repo has a main branch. All test cases assume it exists.
	ensureMainBranch(t, repoURL, appInstallationToken)

	// Each test case sets up a work tree, optionally manipulates the remote to
	// create the desired target branch state, creates two local commits (A by
	// someone else, B by Kargo), then runs github-push and inspects the resulting
	// commits via the GitHub API.
	testCases := []struct {
		name string

		// signingConfigured indicates whether configuration provided when cloning
		// the repository for the test case should include the test signing key.
		// When it's included, all commits made via the work tree will be signed
		// unless explicitly overridden so as not to. This is important in order
		// to allow for testing scenarios where local commits have been signed by
		// a trusted key (Kargo itself) vs those where they have not been.
		signingConfigured bool

		// integrationPolicy is the policy to use for integrating remote changes
		// into the local branch before pushing. This is relevant for testing how
		// different policies affect the trust level of commits post-integration.
		integrationPolicy git.PushIntegrationPolicy

		// generateTargetBranch indicates whether the test should use the
		// github-push step to push to an existing remote target branch (false) or
		// generate a new random branch name to push to (true). This is important
		// because the GitHub API calls required for these two scenarios differ and
		// we want to test both.
		generateTargetBranch bool

		// verifyUntrustedCommits whether the test should unconditionally withhold
		// author/committer details from API calls such that GitHub will always
		// count the authenticated user (the GitHub App) as the author and committer
		// of every commit, which it will then sign with its own key, establishing
		// every commit as verified.
		//
		// Using this option IRL means playing "security theater," valuing the
		// appearance of verified commits over genuine cryptographic verification.
		// Shame.
		verifyUntrustedCommits bool

		// force indicates whether the push should be forced.
		force bool

		// setupTarget prepares the remote target branch state.
		setupTarget func(
			t *testing.T,
			workTree git.WorkTree,
			targetBranch string,
			token string,
		)

		assertions func(t *testing.T, commits []*github.Commit,
		)
	}{
		//
		// Target doesn't exist
		//
		{
			name: "case01",
			// Description: Target doesn't exist. Signing configured. After replay,
			// verification reflects trust: A (untrusted) is unverified, B (trusted)
			// is verified.
			signingConfigured:    true,
			generateTargetBranch: true,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.False(t, commits[0].GetVerification().GetVerified())
				require.Equal(t, someoneElseName, commits[0].GetAuthor().GetName())
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case02",
			// Description: Target doesn't exist. Signing configured.
			// VerifyUntrustedCommits overrides trust — both verified. A gets
			// Co-authored-by since its author differs from signer.
			signingConfigured:      true,
			generateTargetBranch:   true,
			verifyUntrustedCommits: true,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.True(t, commits[0].GetVerification().GetVerified())
				require.Contains(
					t,
					commits[0].GetMessage(),
					fmt.Sprintf("Co-authored-by: %s <%s>", someoneElseName, someoneElseEmail),
				)
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case03",
			// Description: Target doesn't exist. No signing configured. All commits
			// are untrusted — neither is verified. Original attribution preserved.
			generateTargetBranch: true,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.False(t, commits[0].GetVerification().GetVerified())
				require.Equal(t, someoneElseName, commits[0].GetAuthor().GetName())
				require.False(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case04",
			// Description: Target doesn't exist. No signing configured.
			// VerifyUntrustedCommits overrides trust — both verified. A gets
			// Co-authored-by since its author differs from committer.
			generateTargetBranch:   true,
			verifyUntrustedCommits: true,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.True(t, commits[0].GetVerification().GetVerified())
				require.Contains(
					t,
					commits[0].GetMessage(),
					fmt.Sprintf("Co-authored-by: %s <%s>", someoneElseName, someoneElseEmail),
				)
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		//
		// Local is ahead of target
		//
		{
			name: "case05",
			// Description: Local is ahead of target. Signing configured. After
			// replay, verification reflects trust.
			signingConfigured: true,
			setupTarget:       setupLocalAheadOfTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.False(t, commits[0].GetVerification().GetVerified())
				require.Equal(t, someoneElseName, commits[0].GetAuthor().GetName())
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case06",
			// Description: Local is ahead of target. Signing configured.
			// VerifyUntrustedCommits overrides trust — both verified. A gets
			// Co-authored-by.
			signingConfigured:      true,
			verifyUntrustedCommits: true,
			setupTarget:            setupLocalAheadOfTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.True(t, commits[0].GetVerification().GetVerified())
				require.Contains(
					t,
					commits[0].GetMessage(),
					fmt.Sprintf("Co-authored-by: %s <%s>", someoneElseName, someoneElseEmail),
				)
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case07",
			// Description: Local is ahead of target. No signing configured. All
			// commits are untrusted — neither is verified. Original attribution
			// preserved.
			setupTarget: setupLocalAheadOfTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.False(t, commits[0].GetVerification().GetVerified())
				require.Equal(t, someoneElseName, commits[0].GetAuthor().GetName())
				require.False(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case08",
			// Description: Local is ahead of target. No signing configured.
			// VerifyUntrustedCommits overrides trust — both verified. A gets
			// Co-authored-by.
			verifyUntrustedCommits: true,
			setupTarget:            setupLocalAheadOfTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.True(t, commits[0].GetVerification().GetVerified())
				require.Contains(
					t,
					commits[0].GetMessage(),
					fmt.Sprintf("Co-authored-by: %s <%s>", someoneElseName, someoneElseEmail),
				)
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		//
		// Local has diverged from target
		//
		{
			name: "case09",
			// Description: Local has diverged from target. Signing configured.
			// RebaseOrMerge falls back to merge (untrusted commits present). After
			// replay: A unverified with original attribution, B and merge commit
			// verified.
			signingConfigured: true,
			integrationPolicy: git.PushIntegrationPolicyRebaseOrMerge,
			setupTarget:       setupLocalDivergedFromTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 3)
				require.False(t, commits[0].GetVerification().GetVerified())
				require.Equal(t, someoneElseName, commits[0].GetAuthor().GetName())
				require.True(t, commits[1].GetVerification().GetVerified())
				require.True(t, commits[2].GetVerification().GetVerified())
			},
		},
		{
			name: "case10",
			// Description: Local has diverged from target. Signing configured.
			// AlwaysRebase re-signs all commits. Both verified. A gets Co-authored-by
			// trailer.
			signingConfigured: true,
			integrationPolicy: git.PushIntegrationPolicyAlwaysRebase,
			setupTarget:       setupLocalDivergedFromTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.True(t, commits[0].GetVerification().GetVerified())
				require.Contains(
					t,
					commits[0].GetMessage(),
					fmt.Sprintf("Co-authored-by: %s <%s>", someoneElseName, someoneElseEmail),
				)
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case11",
			// Description: Local has diverged from target. Signing configured. Force
			// is enabled, so integration is skipped. After replay, verification
			// reflects trust. A preserves original attribution.
			signingConfigured: true,
			force:             true,
			setupTarget:       setupLocalDivergedFromTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.False(t, commits[0].GetVerification().GetVerified())
				require.Equal(t, someoneElseName, commits[0].GetAuthor().GetName())
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case12",
			// Description: Local has diverged from target. Signing configured. Force
			// + VerifyUntrustedCommits — both verified. A gets Co-authored-by.
			signingConfigured:      true,
			force:                  true,
			verifyUntrustedCommits: true,
			setupTarget:            setupLocalDivergedFromTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.True(t, commits[0].GetVerification().GetVerified())
				require.Contains(
					t,
					commits[0].GetMessage(),
					fmt.Sprintf("Co-authored-by: %s <%s>", someoneElseName, someoneElseEmail),
				)
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case13",
			// Description: Local has diverged from target. No signing configured.
			// RebaseOrMerge: rebase is safe (unsigned stays unsigned). After replay,
			// neither is verified. Original attribution preserved.
			integrationPolicy: git.PushIntegrationPolicyRebaseOrMerge,
			setupTarget:       setupLocalDivergedFromTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.False(t, commits[0].GetVerification().GetVerified())
				require.Equal(t, someoneElseName, commits[0].GetAuthor().GetName())
				require.False(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case14",
			// Description: Local has diverged from target. No signing configured.
			// Force is enabled. After replay, neither is verified. Original
			// attribution preserved.
			force:       true,
			setupTarget: setupLocalDivergedFromTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.False(t, commits[0].GetVerification().GetVerified())
				require.Equal(t, someoneElseName, commits[0].GetAuthor().GetName())
				require.False(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case15",
			// Description: Local has diverged from target. No signing configured.
			// AlwaysRebase + VerifyUntrustedCommits — both verified. A gets
			// Co-authored-by.
			integrationPolicy:      git.PushIntegrationPolicyAlwaysRebase,
			verifyUntrustedCommits: true,
			setupTarget:            setupLocalDivergedFromTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.True(t, commits[0].GetVerification().GetVerified())
				require.Contains(
					t,
					commits[0].GetMessage(),
					fmt.Sprintf("Co-authored-by: %s <%s>", someoneElseName, someoneElseEmail),
				)
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		{
			name: "case16",
			// Description: Local has diverged from target. No signing configured.
			// Force + VerifyUntrustedCommits — both verified. A gets Co-authored-by.
			force:                  true,
			verifyUntrustedCommits: true,
			setupTarget:            setupLocalDivergedFromTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Len(t, commits, 2)
				require.True(t, commits[0].GetVerification().GetVerified())
				require.Contains(t, commits[0].GetMessage(),
					fmt.Sprintf("Co-authored-by: %s <%s>", someoneElseName, someoneElseEmail),
				)
				require.True(t, commits[1].GetVerification().GetVerified())
			},
		},
		//
		// Local is behind target
		//
		{
			name: "case17",
			// Description: Local is behind target. Signing configured. Integration
			// pulls the target's extra commit into local, making them identical.
			// Nothing to replay. Step succeeds with no new commits.
			signingConfigured: true,
			integrationPolicy: git.PushIntegrationPolicyRebaseOrMerge,
			setupTarget:       setupLocalBehindTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Empty(t, commits)
			},
		},
		{
			name: "case18",
			// Description: Local is behind target. Signing configured. AlwaysRebase
			// integrates the target's extra commit. Same outcome as case17 — nothing
			// to replay.
			signingConfigured: true,
			integrationPolicy: git.PushIntegrationPolicyAlwaysRebase,
			setupTarget:       setupLocalBehindTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Empty(t, commits)
			},
		},
		{
			name: "case19",
			// Description: Local is behind target. Signing configured. Force is
			// enabled, so integration is skipped. Zero source-only commits, so
			// nothing is replayed. Ref is moved backward to match local.
			signingConfigured: true,
			force:             true,
			setupTarget:       setupLocalBehindTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Empty(t, commits)
			},
		},
		{
			name: "case20",
			// Description: Local is behind target. No signing configured.
			// RebaseOrMerge integrates the target's extra commit. Nothing to replay.
			integrationPolicy: git.PushIntegrationPolicyRebaseOrMerge,
			setupTarget:       setupLocalBehindTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Empty(t, commits)
			},
		},
		{
			name: "case21",
			// Description: Local is behind target. No signing configured. Same
			// outcome as case20 with AlwaysRebase.
			integrationPolicy: git.PushIntegrationPolicyAlwaysRebase,
			setupTarget:       setupLocalBehindTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Empty(t, commits)
			},
		},
		{
			name: "case22",
			// Description: Local is behind target. No signing configured. Force moves
			// ref backward, nothing to replay.
			force:       true,
			setupTarget: setupLocalBehindTarget,
			assertions: func(t *testing.T, commits []*github.Commit) {
				require.Empty(t, commits)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			// Use a short temp dir path to avoid exceeding the Unix
			// socket path length limit (~104 chars) for GPG's keyboxd.
			workDir, err := os.MkdirTemp("/tmp", "ghp")
			require.NoError(t, err)
			defer os.RemoveAll(workDir)
			targetBranch := fmt.Sprintf("integration-test-%d", time.Now().UnixNano())

			// Clone bare + add work tree (same as git-clone step).
			gitUser := &git.User{
				Name:  kargoName,
				Email: kargoEmail,
			}
			if testCase.signingConfigured {
				gitUser.SigningKey = testSigningKey
			}
			clientOpts := &git.ClientOptions{
				Credentials: &git.RepoCredentials{
					Username: "kargo",
					Password: appInstallationToken,
				},
				User: gitUser,
			}
			bareRepo, err := git.CloneBare(
				repoURL,
				clientOpts,
				&git.BareCloneOptions{BaseDir: workDir},
			)
			require.NoError(t, err)
			defer bareRepo.Close()

			workTree, err := bareRepo.AddWorkTree(
				filepath.Join(workDir, "main"),
				&git.AddWorkTreeOptions{Ref: "main"},
			)
			require.NoError(t, err)

			// The git library's buildGitCommand sets a minimal env with
			// only HOME — no PATH. git commit needs to find gpg, so we
			// must set gpg.program to its absolute path in the work
			// tree's global gitconfig.
			if testCase.signingConfigured {
				gpgPath, gpgErr := exec.LookPath("gpg")
				require.NoError(t, gpgErr, "gpg not found on PATH")
				cmd := exec.Command(
					"git", "-C", workTree.Dir(),
					"config", "--global", "gpg.program", gpgPath,
				)
				cmd.Env = []string{
					fmt.Sprintf("HOME=%s", workTree.HomeDir()),
				}
				out, gpgErr := cmd.CombinedOutput()
				require.NoError(t, gpgErr, string(out))
			}

			// Commit A: Unsigned commit by someone
			require.NoError(t, os.WriteFile(
				filepath.Join(workTree.Dir(), fmt.Sprintf("a-%d.txt", time.Now().UnixNano())),
				[]byte("commit A by someone else"),
				0o600,
			))
			require.NoError(
				t,
				workTree.AddAllAndCommit(
					"commit A by someone else",
					&git.CommitOptions{
						Author: &git.User{
							Name:  someoneElseName,
							Email: someoneElseEmail,
						},
					},
				),
			)

			// Commit B: by Kargo (signed if signing is configured).
			require.NoError(t, os.WriteFile(
				filepath.Join(workTree.Dir(), fmt.Sprintf("b-%d.txt", time.Now().UnixNano())),
				[]byte("commit B by Kargo"),
				0o600,
			))
			require.NoError(t, workTree.AddAllAndCommit(
				"commit B by Kargo", nil,
			))

			// Set up target branch state if needed.
			if testCase.setupTarget != nil {
				testCase.setupTarget(t, workTree, targetBranch, appInstallationToken)
			}

			// Capture the base SHA before the step runs. For new
			// branches, the base is main's HEAD. For existing branches,
			// it's the target branch's current HEAD.
			baseBranch := targetBranch
			if testCase.generateTargetBranch {
				baseBranch = "main"
			}
			baseRef, _, err := ghClient.Git.GetRef(
				t.Context(), repoOwner, repoName, "heads/"+baseBranch,
			)
			var baseSHA string
			if err == nil {
				baseSHA = baseRef.GetObject().GetSHA()
			} else {
				// Branch doesn't exist yet — use main.
				mainRef, _, mainErr := ghClient.Git.GetRef(
					t.Context(), repoOwner, repoName, "heads/main",
				)
				require.NoError(t, mainErr)
				baseSHA = mainRef.GetObject().GetSHA()
			}

			// Build step config.
			stepCfg := promotion.Config{"path": "main"}
			if testCase.generateTargetBranch {
				stepCfg["generateTargetBranch"] = true
			} else {
				stepCfg["targetBranch"] = targetBranch
			}
			if testCase.force && !testCase.generateTargetBranch {
				stepCfg["force"] = true
			}

			// Run github-push.
			runner := newGitHubPusher(
				promotion.StepRunnerCapabilities{CredsDB: credsDB},
				githubPusherConfig{
					PushIntegrationPolicy:  testCase.integrationPolicy,
					MaxRevisions:           10,
					VerifyUntrustedCommits: testCase.verifyUntrustedCommits,
				},
			)
			promotionID := fmt.Sprintf("test-%d", time.Now().UnixNano())
			result, err := runner.Run(
				t.Context(),
				&promotion.StepContext{
					Project:   "test-project",
					Stage:     "test-stage",
					Promotion: promotionID,
					WorkDir:   workDir,
					Config:    stepCfg,
				},
			)
			require.NoError(t, err)
			require.Equal(t, "Succeeded", string(result.Status))

			// Determine the actual branch that was pushed to.
			pushedBranch := targetBranch
			if testCase.generateTargetBranch {
				b, ok := result.Output[stateKeyBranch].(string)
				require.True(t, ok)
				pushedBranch = b
			}

			// Fetch the commits on the pushed branch from GitHub API.
			headSHA, ok := result.Output[stateKeyCommit].(string)
			require.True(t, ok)

			commits := fetchCommitChain(
				t, ghClient, repoOwner, repoName, baseSHA, headSHA,
			)

			t.Logf("branch commits: https://github.com/%s/%s/commits/%s",
				repoOwner, repoName, pushedBranch,
			)
			// Log for manual inspection.
			for i, c := range commits {
				t.Logf(
					"commit[%d]: sha=%s author=%s verified=%v",
					i, c.GetSHA()[:7], c.GetAuthor().GetName(), c.GetVerification().GetVerified(),
				)
			}

			// Run test-specific assertions.
			testCase.assertions(t, commits)

			// Clean up: delete the remote branch.
			_, _ = ghClient.Git.DeleteRef(
				t.Context(), repoOwner, repoName, "heads/"+pushedBranch,
			)
		})
	}
}

func getAppInstallationToken(
	t *testing.T,
	clientID string,
	installationIDStr string,
	privateKey string,
) string {
	t.Helper()
	t.Log("Using GitHub App installation token")
	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	require.NoError(t, err)
	appSrc, err := githubauth.NewApplicationTokenSource(clientID, []byte(privateKey))
	require.NoError(t, err)
	instSrc := githubauth.NewInstallationTokenSource(installationID, appSrc)
	token, err := instSrc.Token()
	require.NoError(t, err)
	return token.AccessToken
}

func ensureMainBranch(
	t *testing.T,
	repoURL string,
	token string,
) {
	t.Helper()
	clientOpts := &git.ClientOptions{
		Credentials: &git.RepoCredentials{
			Username: "kargo",
			Password: token,
		},
		User: &git.User{
			Name:  "Test",
			Email: "test@test.com",
		},
	}
	// If main already exists, nothing to do.
	if repo, err := git.Clone(
		repoURL, clientOpts,
		&git.CloneOptions{Branch: "main", SingleBranch: true},
	); err == nil {
		repo.Close()
		return
	}
	// Create main with an initial commit.
	repo, err := git.Clone(repoURL, clientOpts, nil)
	require.NoError(t, err)
	defer repo.Close()
	require.NoError(t, os.WriteFile(
		filepath.Join(repo.Dir(), "README.md"),
		[]byte("# integration test repo\n"),
		0o600,
	))
	require.NoError(t, repo.AddAllAndCommit("initial commit", nil))
	require.NoError(t, repo.Push(nil))
}

func setupLocalBehindTarget(
	t *testing.T,
	workTree git.WorkTree,
	targetBranch string,
	token string,
) {
	t.Helper()
	// Push the local branch (which has test commits A + B) to the target
	// branch so that the target contains everything the local has.
	require.NoError(t, workTree.CreateChildBranch(targetBranch))
	require.NoError(t, workTree.Push(
		&git.PushOptions{TargetBranch: targetBranch},
	))
	// Switch back to main so the work tree is in the expected state.
	require.NoError(t, workTree.Checkout("main"))
	// Clone the target branch and add a commit so it's ahead of local.
	repo, err := git.Clone(
		workTree.URL(),
		&git.ClientOptions{
			Credentials: &git.RepoCredentials{
				Username: "kargo",
				Password: token,
			},
			User: &git.User{Name: "Remote Author", Email: "remote@author.com"},
		},
		&git.CloneOptions{Branch: targetBranch, SingleBranch: true},
	)
	require.NoError(t, err)
	defer repo.Close()
	require.NoError(t, os.WriteFile(
		filepath.Join(repo.Dir(),
			fmt.Sprintf("remote-%d.txt", time.Now().UnixNano())),
		[]byte("remote commit"),
		0o600,
	))
	require.NoError(t, repo.AddAllAndCommit("remote commit ahead", nil))
	require.NoError(t, repo.Push(nil))
}

func setupLocalAheadOfTarget(
	t *testing.T,
	workTree git.WorkTree,
	targetBranch string,
	token string,
) {
	t.Helper()
	// Clone main from the remote. This gets us a repo at the remote's
	// current HEAD — before our test commits, which only exist locally.
	repo, err := git.Clone(
		workTree.URL(),
		&git.ClientOptions{
			Credentials: &git.RepoCredentials{
				Username: "kargo",
				Password: token,
			},
			User: &git.User{Name: "Test", Email: "test@test.com"},
		},
		&git.CloneOptions{Branch: "main", SingleBranch: true},
	)
	require.NoError(t, err)
	defer repo.Close()
	// Create the target branch at this point and push it.
	require.NoError(t, repo.CreateChildBranch(targetBranch))
	require.NoError(t, repo.Push(
		&git.PushOptions{TargetBranch: targetBranch},
	))
}

func setupLocalDivergedFromTarget(
	t *testing.T,
	workTree git.WorkTree,
	targetBranch string,
	token string,
) {
	t.Helper()
	// Start by creating the target branch at main's HEAD.
	setupLocalAheadOfTarget(t, workTree, targetBranch, token)
	// Clone the target branch and add a commit.
	repo, err := git.Clone(
		workTree.URL(),
		&git.ClientOptions{
			Credentials: &git.RepoCredentials{
				Username: "kargo",
				Password: token,
			},
			User: &git.User{Name: "Remote Author", Email: "remote@author.com"},
		},
		&git.CloneOptions{Branch: targetBranch, SingleBranch: true},
	)
	require.NoError(t, err)
	defer repo.Close()
	require.NoError(t, os.WriteFile(
		filepath.Join(repo.Dir(),
			fmt.Sprintf("remote-%d.txt", time.Now().UnixNano())),
		[]byte("remote commit"),
		0o600,
	))
	require.NoError(t, repo.AddAllAndCommit("remote commit ahead", nil))
	require.NoError(t, repo.Push(nil))
}

func fetchCommitChain(
	t *testing.T,
	ghClient *github.Client,
	owner, repo, baseSHA, headSHA string,
) []*github.Commit {
	t.Helper()
	comparison, _, err := ghClient.Repositories.CompareCommits(
		t.Context(), owner, repo, baseSHA, headSHA,
		&github.ListOptions{PerPage: 100},
	)
	require.NoError(t, err)

	var commits []*github.Commit
	for _, rc := range comparison.Commits {
		c, _, err := ghClient.Git.GetCommit(
			t.Context(), owner, repo, rc.GetSHA(),
		)
		require.NoError(t, err)
		commits = append(commits, c)
	}
	return commits
}
