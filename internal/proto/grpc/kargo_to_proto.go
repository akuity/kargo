package conversion

import (
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	freightv1 "github.com/akuity/kargo/pkg/generated/freight/v1"
)

// KargoFreightCollectionToProto converts a Kargo FreightCollection to protobuf format
func KargoFreightCollectionToProto(kargoFC kargoapi.FreightCollection) *freightv1.FreightCollection {
	if kargoFC.Freight == nil {
		return &freightv1.FreightCollection{
			Id:      kargoFC.ID,
			Freight: make(map[string]*freightv1.FreightReference),
		}
	}

	protoFC := &freightv1.FreightCollection{
		Id:      kargoFC.ID,
		Freight: make(map[string]*freightv1.FreightReference, len(kargoFC.Freight)),
	}

	for origin, freightRef := range kargoFC.Freight {
		protoFC.Freight[origin] = KargoFreightReferenceToProto(freightRef)
	}

	return protoFC
}

// KargoFreightReferenceToProto converts a Kargo FreightReference to protobuf format
func KargoFreightReferenceToProto(kargoFR kargoapi.FreightReference) *freightv1.FreightReference {
	protoFR := &freightv1.FreightReference{
		Name:     kargoFR.Name,
		Origin:   KargoFreightOriginToProto(kargoFR.Origin),
		Commits:  make([]*freightv1.GitCommit, len(kargoFR.Commits)),
		Images:   make([]*freightv1.Image, len(kargoFR.Images)),
		Charts:   make([]*freightv1.Chart, len(kargoFR.Charts)),
		Metadata: make(map[string]string),
	}

	// Convert commits
	for i, commit := range kargoFR.Commits {
		protoFR.Commits[i] = KargoGitCommitToProto(commit)
	}

	// Convert images
	for i, image := range kargoFR.Images {
		protoFR.Images[i] = KargoImageToProto(image)
	}

	// Convert charts
	for i, chart := range kargoFR.Charts {
		protoFR.Charts[i] = KargoChartToProto(chart)
	}

	// Note: Kargo's FreightReference doesn't have a metadata field,
	// but our protobuf does. We initialize it as empty for future extensibility.

	return protoFR
}

// KargoFreightOriginToProto converts a Kargo FreightOrigin to protobuf format
func KargoFreightOriginToProto(kargoFO kargoapi.FreightOrigin) *freightv1.FreightOrigin {
	return &freightv1.FreightOrigin{
		Kind: string(kargoFO.Kind),
		Name: kargoFO.Name,
	}
}

// KargoGitCommitToProto converts a Kargo GitCommit to protobuf format
func KargoGitCommitToProto(kargoGC kargoapi.GitCommit) *freightv1.GitCommit {
	protoGC := &freightv1.GitCommit{
		RepoUrl: kargoGC.RepoURL,
		Id:      kargoGC.ID,
		Branch:  kargoGC.Branch,
		Tag:     kargoGC.Tag,
		Message: kargoGC.Message,
	}

	// Convert author string to GitUser structure
	if kargoGC.Author != "" {
		protoGC.Author = &freightv1.GitUser{
			Name:  kargoGC.Author,
			Email: "",  // Kargo doesn't separate name/email for author
			When:  nil, // Kargo doesn't store author timestamp separately
		}
	}

	// Convert committer string to GitUser structure
	if kargoGC.Committer != "" {
		protoGC.Committer = &freightv1.GitUser{
			Name:  kargoGC.Committer,
			Email: "",  // Kargo doesn't separate name/email for committer
			When:  nil, // Kargo doesn't store committer timestamp separately
		}
	}

	// Note: Kargo's GitCommit doesn't have a timestamp field,
	// but our protobuf does. We leave it nil for now.

	return protoGC
}

// KargoImageToProto converts a Kargo Image to protobuf format
func KargoImageToProto(kargoImg kargoapi.Image) *freightv1.Image {
	protoImg := &freightv1.Image{
		RepoUrl:     kargoImg.RepoURL,
		Tag:         kargoImg.Tag,
		Digest:      kargoImg.Digest,
		Annotations: make(map[string]string),
	}

	// Copy annotations
	if kargoImg.Annotations != nil {
		for k, v := range kargoImg.Annotations {
			protoImg.Annotations[k] = v
		}
	}

	// Note: Kargo's Image doesn't have a CreatedAt field,
	// but our protobuf does. We leave it nil for now.

	return protoImg
}

// KargoChartToProto converts a Kargo Chart to protobuf format
func KargoChartToProto(kargoChart kargoapi.Chart) *freightv1.Chart {
	return &freightv1.Chart{
		RepoUrl:  kargoChart.RepoURL,
		Name:     kargoChart.Name,
		Version:  kargoChart.Version,
		Metadata: make(map[string]string), // Kargo Chart doesn't have metadata, but our protobuf does
	}
}

// ProtoFreightCollectionToKargo converts a protobuf FreightCollection back to Kargo format
func ProtoFreightCollectionToKargo(protoFC *freightv1.FreightCollection) kargoapi.FreightCollection {
	if protoFC == nil {
		return kargoapi.FreightCollection{}
	}

	kargoFC := kargoapi.FreightCollection{
		ID:      protoFC.Id,
		Freight: make(map[string]kargoapi.FreightReference, len(protoFC.Freight)),
	}

	for origin, protoFR := range protoFC.Freight {
		kargoFC.Freight[origin] = ProtoFreightReferenceToKargo(protoFR)
	}

	return kargoFC
}

// ProtoFreightReferenceToKargo converts a protobuf FreightReference back to Kargo format
func ProtoFreightReferenceToKargo(protoFR *freightv1.FreightReference) kargoapi.FreightReference {
	if protoFR == nil {
		return kargoapi.FreightReference{}
	}

	kargoFR := kargoapi.FreightReference{
		Name:    protoFR.Name,
		Origin:  ProtoFreightOriginToKargo(protoFR.Origin),
		Commits: make([]kargoapi.GitCommit, len(protoFR.Commits)),
		Images:  make([]kargoapi.Image, len(protoFR.Images)),
		Charts:  make([]kargoapi.Chart, len(protoFR.Charts)),
	}

	// Convert commits
	for i, protoCommit := range protoFR.Commits {
		kargoFR.Commits[i] = ProtoGitCommitToKargo(protoCommit)
	}

	// Convert images
	for i, protoImage := range protoFR.Images {
		kargoFR.Images[i] = ProtoImageToKargo(protoImage)
	}

	// Convert charts
	for i, protoChart := range protoFR.Charts {
		kargoFR.Charts[i] = ProtoChartToKargo(protoChart)
	}

	// Note: We ignore the metadata field from protobuf since Kargo's FreightReference doesn't have it

	return kargoFR
}

// ProtoFreightOriginToKargo converts a protobuf FreightOrigin back to Kargo format
func ProtoFreightOriginToKargo(protoFO *freightv1.FreightOrigin) kargoapi.FreightOrigin {
	if protoFO == nil {
		return kargoapi.FreightOrigin{}
	}

	return kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKind(protoFO.Kind),
		Name: protoFO.Name,
	}
}

// ProtoGitCommitToKargo converts a protobuf GitCommit back to Kargo format
func ProtoGitCommitToKargo(protoGC *freightv1.GitCommit) kargoapi.GitCommit {
	if protoGC == nil {
		return kargoapi.GitCommit{}
	}

	kargoGC := kargoapi.GitCommit{
		RepoURL: protoGC.RepoUrl,
		ID:      protoGC.Id,
		Branch:  protoGC.Branch,
		Tag:     protoGC.Tag,
		Message: protoGC.Message,
	}

	// Convert GitUser back to string
	if protoGC.Author != nil {
		kargoGC.Author = protoGC.Author.Name
	}

	if protoGC.Committer != nil {
		kargoGC.Committer = protoGC.Committer.Name
	}

	// Note: We ignore the timestamp field from protobuf since Kargo's GitCommit doesn't have it

	return kargoGC
}

// ProtoImageToKargo converts a protobuf Image back to Kargo format
func ProtoImageToKargo(protoImg *freightv1.Image) kargoapi.Image {
	if protoImg == nil {
		return kargoapi.Image{}
	}

	kargoImg := kargoapi.Image{
		RepoURL: protoImg.RepoUrl,
		Tag:     protoImg.Tag,
		Digest:  protoImg.Digest,
	}

	// Copy annotations
	if len(protoImg.Annotations) > 0 {
		kargoImg.Annotations = make(map[string]string, len(protoImg.Annotations))
		for k, v := range protoImg.Annotations {
			kargoImg.Annotations[k] = v
		}
	}

	// Note: We ignore the CreatedAt field from protobuf since Kargo's Image doesn't have it

	return kargoImg
}

// ProtoChartToKargo converts a protobuf Chart back to Kargo format
func ProtoChartToKargo(protoChart *freightv1.Chart) kargoapi.Chart {
	if protoChart == nil {
		return kargoapi.Chart{}
	}

	return kargoapi.Chart{
		RepoURL: protoChart.RepoUrl,
		Name:    protoChart.Name,
		Version: protoChart.Version,
	}
	// Note: We ignore the metadata field from protobuf since Kargo's Chart doesn't have it
}
