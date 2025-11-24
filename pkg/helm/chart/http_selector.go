package chart

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/helm"
)

// httpSelector is an implementation of Selector that interacts with classic
// (http/s-based) Helm chart repositories.
type httpSelector struct {
	*baseSelector
	indexURL  string
	chartName string
	creds     *helm.Credentials
}

func newHTTPSelector(
	sub kargoapi.ChartSubscription,
	creds *helm.Credentials,
) (Selector, error) {
	base, err := newBaseSelector(sub)
	if err != nil {
		return nil, fmt.Errorf("error building base selector: %w", err)
	}
	return &httpSelector{
		baseSelector: base,
		indexURL: fmt.Sprintf(
			"%s/index.yaml",
			strings.TrimSuffix(sub.RepoURL, "/"),
		),
		chartName: sub.Name,
		creds:     creds,
	}, nil
}

// Select implements Selector.
func (h *httpSelector) Select(context.Context) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, h.indexURL, nil)
	if err != nil {
		return nil,
			fmt.Errorf("error preparing HTTP/S request to %q: %w", h.indexURL, err)
	}
	if h.creds != nil {
		req.SetBasicAuth(h.creds.Username, h.creds.Password)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil,
			fmt.Errorf("error querying repository index at %q: %w", h.indexURL, err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"received unexpected HTTP %d when querying repository index at %q",
			res.StatusCode,
			h.indexURL,
		)
	}
	defer res.Body.Close()
	resBodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil,
			fmt.Errorf("error reading repository index from %q: %w", h.indexURL, err)
	}
	index := struct {
		Entries map[string][]struct {
			Version string `json:"version,omitempty"`
		} `json:"entries,omitempty"`
	}{}
	if err = yaml.Unmarshal(resBodyBytes, &index); err != nil {
		return nil, fmt.Errorf(
			"error unmarshaling repository index from %q: %w",
			h.indexURL, err,
		)
	}
	entries, ok := index.Entries[h.chartName]
	if !ok {
		return nil, nil
	}
	semvers := make(semver.Collection, 0, len(entries))
	for _, entry := range entries {
		sv, err := semver.NewVersion(entry.Version)
		if err == nil {
			semvers = append(semvers, sv)
		}
	}
	semvers = h.filterSemvers(semvers)
	h.sort(semvers)
	return h.semversToVersionStrings(semvers), nil
}
