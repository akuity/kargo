package freight

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func toContents(freight *kargoapi.Freight) (database.FreightContents, error) {
	result := database.FreightContents{}
	id := string(freight.UID)
	for i, commit := range freight.Commits {
		result.Commits = append(result.Commits, database.UpsertFreightCommitParams{
			FreightID: id, Ordinal: int64(i), RepoURL: commit.RepoURL, CommitID: commit.ID,
			Branch: commit.Branch, Tag: commit.Tag, Message: commit.Message,
			Author: commit.Author, Committer: commit.Committer, SubscriptionName: commit.SubscriptionName,
		})
	}
	for i, image := range freight.Images {
		annotations := []byte("{}")
		if len(image.Annotations) > 0 {
			var err error
			annotations, err = json.Marshal(image.Annotations)
			if err != nil {
				return database.FreightContents{}, fmt.Errorf("error encoding image annotations: %w", err)
			}
		}
		result.Images = append(result.Images, database.UpsertFreightImageParams{
			FreightID: id, Ordinal: int64(i), RepoURL: image.RepoURL,
			Tag: image.Tag, Digest: image.Digest, SubscriptionName: image.SubscriptionName, Annotations: annotations,
		})
	}
	for i, chart := range freight.Charts {
		result.Charts = append(result.Charts, database.UpsertFreightChartParams{
			FreightID: id, Ordinal: int64(i), RepoURL: chart.RepoURL,
			Name: chart.Name, Version: chart.Version, SubscriptionName: chart.SubscriptionName,
		})
	}
	for i, artifact := range freight.Artifacts {
		var metadata []byte
		if artifact.Metadata != nil {
			// Keep opaque JSON numbers exact rather than decoding through float64.
			var compact bytes.Buffer
			if err := json.Compact(&compact, artifact.Metadata.Raw); err != nil {
				return database.FreightContents{}, fmt.Errorf("error encoding artifact metadata: %w", err)
			}
			metadata = compact.Bytes()
			if len(metadata) == 0 || metadata[0] != '{' {
				return database.FreightContents{}, fmt.Errorf("artifact metadata must be a JSON object")
			}
		}
		result.Artifacts = append(result.Artifacts, database.UpsertFreightArtifactParams{
			FreightID: id, Ordinal: int64(i), ArtifactType: artifact.ArtifactType,
			SubscriptionName: artifact.SubscriptionName, Version: artifact.Version, Metadata: metadata,
		})
	}
	return result, nil
}

func equalContents(a, b database.FreightContents) bool {
	return slices.Equal(a.Commits, b.Commits) && slices.Equal(a.Charts, b.Charts) &&
		slices.EqualFunc(a.Images, b.Images, func(left, right database.UpsertFreightImageParams) bool {
			annotationsMatch := equalJSON(left.Annotations, right.Annotations)
			left.Annotations, right.Annotations = nil, nil
			return annotationsMatch && reflect.DeepEqual(left, right)
		}) &&
		slices.EqualFunc(a.Artifacts, b.Artifacts, func(left, right database.UpsertFreightArtifactParams) bool {
			metadataMatch := equalJSON(left.Metadata, right.Metadata)
			left.Metadata, right.Metadata = nil, nil
			return metadataMatch && reflect.DeepEqual(left, right)
		})
}
