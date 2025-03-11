package directives

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	intyaml "github.com/akuity/kargo/internal/yaml"
	"github.com/akuity/kargo/pkg/x/directive/builtin"
)

// preserveSeparator is the separator used to preserve values in the
// Kustomization image field.
const preserveSeparator = "*"

// kustomizeImageSetter is an implementation  of the PromotionStepRunner
// interface that sets images in a Kustomization file.
type kustomizeImageSetter struct {
	schemaLoader gojsonschema.JSONLoader
	kargoClient  client.Client
}

// newKustomizeImageSetter returns an implementation  of the PromotionStepRunner
// interface that sets images in a Kustomization file.
func newKustomizeImageSetter(kargoClient client.Client) PromotionStepRunner {
	return &kustomizeImageSetter{
		schemaLoader: getConfigSchemaLoader("kustomize-set-image"),
		kargoClient:  kargoClient,
	}
}

// Name implements the PromotionStepRunner interface.
func (k *kustomizeImageSetter) Name() string {
	return "kustomize-set-image"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (k *kustomizeImageSetter) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}

	if err := k.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed object.
	cfg, err := ConfigToStruct[builtin.KustomizeSetImageConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into kustomize-set-image config: %w", err)
	}

	return k.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates kustomizeImageSetter configuration against a JSON schema.
func (k *kustomizeImageSetter) validate(cfg Config) error {
	return validate(k.schemaLoader, gojsonschema.NewGoLoader(cfg), k.Name())
}

func (k *kustomizeImageSetter) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg builtin.KustomizeSetImageConfig,
) (PromotionStepResult, error) {
	// Find the Kustomization file.
	kusPath, err := findKustomization(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not discover kustomization file: %w", err)
	}

	var targetImages map[string]kustypes.Image
	switch {
	case len(cfg.Images) > 0:
		// Discover image origins and collect target images.
		targetImages = k.buildTargetImagesFromConfig(cfg.Images)
	default:
		// Attempt to automatically set target images based on the Freight references.
		targetImages, err = k.buildTargetImagesAutomatically(ctx, stepCtx)
	}
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}

	// Update the Kustomization file with the new images.
	if err = updateKustomizationFile(kusPath, targetImages); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}

	result := PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}
	if commitMsg := k.generateCommitMessage(cfg.Path, targetImages); commitMsg != "" {
		result.Output = map[string]any{
			"commitMessage": commitMsg,
		}
	}
	return result, nil
}

func (k *kustomizeImageSetter) buildTargetImagesFromConfig(
	images []builtin.Image,
) map[string]kustypes.Image {
	targetImages := make(map[string]kustypes.Image, len(images))
	for _, img := range images {
		targetImage := kustypes.Image{
			Name:    img.Image,
			NewName: img.NewName,
		}
		if img.Name != "" {
			targetImage.Name = img.Name
		}
		if img.Digest != "" {
			targetImage.Digest = img.Digest
		} else if img.Tag != "" {
			targetImage.NewTag = img.Tag
		}
		targetImages[targetImage.Name] = targetImage
	}
	return targetImages
}

func (k *kustomizeImageSetter) buildTargetImagesAutomatically(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (map[string]kustypes.Image, error) {
	// Check if there are any ambiguous image requests.
	//
	// We do this based on the request because the Freight references may not
	// contain all the images that are requested, which could lead eventually
	// to an ambiguous result.
	if ambiguous, ambErr := freight.HasAmbiguousImageRequest(
		ctx, k.kargoClient, stepCtx.Project, stepCtx.FreightRequests,
	); ambErr != nil || ambiguous {
		err := errors.New("manual configuration required due to ambiguous result")
		if ambErr != nil {
			err = fmt.Errorf("%w: %v", err, ambErr)
		}
		return nil, err
	}

	var images = make(map[string]kustypes.Image)
	for _, freightRef := range stepCtx.Freight.References() {
		if len(freightRef.Images) == 0 {
			continue
		}

		for _, img := range freightRef.Images {
			images[img.RepoURL] = kustypes.Image{
				Name:   img.RepoURL,
				NewTag: img.Tag,
				Digest: img.Digest,
			}
		}
	}
	return images, nil
}

func (k *kustomizeImageSetter) generateCommitMessage(path string, images map[string]kustypes.Image) string {
	if len(images) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString(fmt.Sprintf("Updated %s to use new image", path))
	if len(images) > 1 {
		_, _ = commitMsg.WriteString("s")
	}
	_, _ = commitMsg.WriteString("\n")

	// Sort the images by name, in descending order for consistency.
	imageNames := make([]string, 0, len(images))
	for name := range images {
		imageNames = append(imageNames, name)
	}
	slices.Sort(imageNames)

	// Append each image to the commit message.
	for _, name := range imageNames {
		i := images[name]

		ref := i.Name
		if i.NewName != "" {
			ref = i.NewName
		}
		if i.NewTag != "" {
			ref = fmt.Sprintf("%s:%s", ref, i.NewTag)
		}
		if i.Digest != "" {
			ref = fmt.Sprintf("%s@%s", ref, i.Digest)
		}
		_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s", ref))
	}

	return commitMsg.String()
}

func updateKustomizationFile(kusPath string, targetImages map[string]kustypes.Image) error {
	// Read the Kustomization file, and unmarshal it.
	node, err := readKustomizationFile(kusPath)
	if err != nil {
		return err
	}

	// Decode the Kustomization file into a typed object to work with.
	currentImages, err := getCurrentImages(node)
	if err != nil {
		return err
	}

	// Merge existing images with new images.
	newImages := mergeImages(currentImages, targetImages)

	// Update the images field in the Kustomization file.
	if err = intyaml.UpdateField(node, "images", newImages); err != nil {
		return fmt.Errorf("could not update images field in Kustomization file: %w", err)
	}

	// Write the updated Kustomization file.
	return writeKustomizationFile(kusPath, node)
}

func readKustomizationFile(kusPath string) (*yaml.Node, error) {
	b, err := os.ReadFile(kusPath)
	if err != nil {
		return nil, fmt.Errorf("could not read Kustomization file: %w", err)
	}
	var node yaml.Node
	if err = yaml.Unmarshal(b, &node); err != nil {
		return nil, fmt.Errorf("could not unmarshal Kustomization file: %w", err)
	}
	return &node, nil
}

func getCurrentImages(node *yaml.Node) ([]kustypes.Image, error) {
	var currentImages []kustypes.Image
	if err := intyaml.DecodeField(node, "images", &currentImages); err != nil {
		var fieldErr intyaml.FieldNotFoundErr
		if !errors.As(err, &fieldErr) {
			return nil, fmt.Errorf("could not decode images field in Kustomization file: %w", err)
		}
	}
	return currentImages, nil
}

func mergeImages(currentImages []kustypes.Image, targetImages map[string]kustypes.Image) []kustypes.Image {
	for _, img := range currentImages {
		if targetImg, ok := targetImages[img.Name]; ok {
			// Reuse the existing new name when asterisk new name is passed
			if targetImg.NewName == preserveSeparator {
				targetImg = replaceNewName(targetImg, img.NewName)
			}

			// Reuse the existing new tag when asterisk new tag is passed
			if targetImg.NewTag == preserveSeparator {
				targetImg = replaceNewTag(targetImg, img.NewTag)
			}

			// Reuse the existing digest when asterisk digest is passed
			if targetImg.Digest == preserveSeparator {
				targetImg = replaceDigest(targetImg, img.Digest)
			}

			targetImages[img.Name] = targetImg

			continue
		}

		targetImages[img.Name] = img
	}

	var newImages []kustypes.Image
	for _, v := range targetImages {
		if v.NewName == preserveSeparator {
			v = replaceNewName(v, "")
		}

		if v.NewTag == preserveSeparator {
			v = replaceNewTag(v, "")
		}

		if v.Digest == preserveSeparator {
			v = replaceDigest(v, "")
		}

		newImages = append(newImages, v)
	}

	// Sort the images by name, in descending order.
	slices.SortFunc(newImages, func(a, b kustypes.Image) int {
		return strings.Compare(a.Name, b.Name)
	})

	return newImages
}

func writeKustomizationFile(kusPath string, node *yaml.Node) error {
	b, err := kyaml.Marshal(node)
	if err != nil {
		return fmt.Errorf("could not marshal updated Kustomization file: %w", err)
	}
	if err = os.WriteFile(kusPath, b, fs.ModePerm); err != nil {
		return fmt.Errorf("could not write updated Kustomization file: %w", err)
	}
	return nil
}

func findKustomization(workDir, path string) (string, error) {
	secureDir, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return "", fmt.Errorf("could not secure join path %q: %w", path, err)
	}

	var candidates []string
	for _, name := range konfig.RecognizedKustomizationFileNames() {
		p := filepath.Join(secureDir, name)
		fi, err := os.Lstat(p)
		if err != nil {
			continue
		}
		if !fi.Mode().IsRegular() {
			continue
		}
		candidates = append(candidates, p)
	}

	switch len(candidates) {
	case 0:
		return "", fmt.Errorf("could not find any Kustomization files in %q", path)
	case 1:
		return candidates[0], nil
	default:
		return "", fmt.Errorf("ambiguous result: found multiple Kustomization files in %q: %v", path, candidates)
	}
}

func replaceNewName(image kustypes.Image, newName string) kustypes.Image {
	return kustypes.Image{
		Name:    image.Name,
		NewName: newName,
		NewTag:  image.NewTag,
		Digest:  image.Digest,
	}
}

func replaceNewTag(image kustypes.Image, newTag string) kustypes.Image {
	return kustypes.Image{
		Name:    image.Name,
		NewName: image.NewName,
		NewTag:  newTag,
		Digest:  image.Digest,
	}
}

func replaceDigest(image kustypes.Image, digest string) kustypes.Image {
	return kustypes.Image{
		Name:    image.Name,
		NewName: image.NewName,
		NewTag:  image.NewTag,
		Digest:  digest,
	}
}
