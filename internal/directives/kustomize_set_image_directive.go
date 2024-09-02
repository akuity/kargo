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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	intyaml "github.com/akuity/kargo/internal/yaml"
)

// preserveSeparator is the separator used to preserve values in the
// Kustomization image field.
const preserveSeparator = "*"

func init() {
	// Register the kustomize-set-image directive with the builtins registry.
	builtins.RegisterDirective(
		newKustomizeSetImageDirective(),
		&DirectivePermissions{
			AllowKargoClient: true,
		},
	)
}

// kustomizeSetImageDirective is a directive that sets images in a Kustomization
// file.
type kustomizeSetImageDirective struct {
	schemaLoader gojsonschema.JSONLoader
}

// newKustomizeSetImageDirective creates a new kustomize-set-image directive.
func newKustomizeSetImageDirective() Directive {
	return &kustomizeSetImageDirective{
		schemaLoader: getConfigSchemaLoader("kustomize-set-image"),
	}
}

func (d *kustomizeSetImageDirective) Name() string {
	return "kustomize-set-image"
}

func (d *kustomizeSetImageDirective) Run(ctx context.Context, stepCtx *StepContext) (Result, error) {
	// Validate the configuration against the JSON Schema.
	if err := validate(d.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), d.Name()); err != nil {
		return Result{Status: StatusFailure}, err
	}

	// Convert the configuration into a typed object.
	cfg, err := configToStruct[KustomizeSetImageConfig](stepCtx.Config)
	if err != nil {
		return Result{Status: StatusFailure},
			fmt.Errorf("could not convert config into kustomize-set-image config: %w", err)
	}

	return d.run(ctx, stepCtx, cfg)
}

func (d *kustomizeSetImageDirective) run(
	ctx context.Context,
	stepCtx *StepContext,
	cfg KustomizeSetImageConfig,
) (Result, error) {
	// Find the Kustomization file.
	kusPath, err := findKustomization(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return Result{Status: StatusFailure}, fmt.Errorf("could not discover kustomization file: %w", err)
	}

	// Discover image origins and collect target images.
	images, err := discoverImages(ctx, stepCtx.KargoClient, stepCtx.Project, cfg.Images, stepCtx.FreightRequests)
	if err != nil {
		return Result{Status: StatusFailure}, err
	}
	targetImages, err := buildTargetImages(images, stepCtx.Freight.Freight)
	if err != nil {
		return Result{Status: StatusFailure}, err
	}

	// Update the Kustomization file with the new images.
	if err := updateKustomizationFile(kusPath, targetImages); err != nil {
		return Result{Status: StatusFailure}, err
	}

	return Result{Status: StatusSuccess}, nil
}

func discoverImages(
	ctx context.Context,
	c client.Client,
	namespace string,
	images []KustomizeSetImageConfigImage,
	freight []kargoapi.FreightRequest,
) ([]KustomizeSetImageConfigImage, error) {
	discoveredImages := slices.Clone(images)
	for i, img := range discoveredImages {
		discoveredImage, err := discoverImage(ctx, c, namespace, img, freight)
		if err != nil {
			return nil, fmt.Errorf("unable to discover image for %q: %w", img.Image, err)
		}
		if discoveredImage == nil {
			return nil, fmt.Errorf("no image found for %q", img.Image)
		}
		discoveredImages[i] = *discoveredImage
	}
	return discoveredImages, nil
}

func discoverImage(
	ctx context.Context,
	c client.Client,
	namespace string,
	image KustomizeSetImageConfigImage,
	requestedFreight []kargoapi.FreightRequest,
) (*KustomizeSetImageConfigImage, error) {
	if image.FromOrigin != nil {
		return &image, nil
	}

	var discoverdImage *KustomizeSetImageConfigImage
	for _, req := range requestedFreight {
		warehouse, err := kargoapi.GetWarehouse(ctx, c, types.NamespacedName{
			Name:      req.Origin.Name,
			Namespace: namespace,
		})
		if err != nil {
			return nil, fmt.Errorf("error getting Warehouse %q in namespace %q: %w", req.Origin.Name, namespace, err)
		}
		if warehouse == nil {
			return nil, fmt.Errorf("Warehouse %q not found in namespace %q", req.Origin.Name, namespace)
		}
		for _, sub := range warehouse.Spec.Subscriptions {
			if sub.Image != nil && sub.Image.RepoURL == image.Image {
				if discoverdImage != nil {
					return nil, fmt.Errorf(
						"multiple requested Freight could provide a container image from repository %q: "+
							"please provide an origin manually to disambiguate", image.Image)
				}
				image.FromOrigin = &ChartFromOrigin{
					Kind: Kind(warehouse.Kind),
					Name: warehouse.Name,
				}
				discoverdImage = &image
			}
		}
	}
	return discoverdImage, nil
}

func buildTargetImages(
	images []KustomizeSetImageConfigImage,
	freight map[string]kargoapi.FreightReference,
) (map[string]kustypes.Image, error) {
	targetImages := make(map[string]kustypes.Image, len(images))

	for _, img := range images {
		targetImage, err := buildTargetImage(img, freight)
		if err != nil {
			return nil, err
		}
		targetImages[targetImage.Name] = targetImage
	}

	return targetImages, nil
}

func buildTargetImage(
	img KustomizeSetImageConfigImage,
	freight map[string]kargoapi.FreightReference,
) (kustypes.Image, error) {
	if img.FromOrigin == nil {
		return kustypes.Image{}, fmt.Errorf("image %q has no origin specified", img.Image)
	}

	for _, f := range freight {
		if !f.Origin.Equals(&kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKind(img.FromOrigin.Kind),
			Name: img.FromOrigin.Name,
		}) {
			continue
		}

		for _, i := range f.Images {
			if i.RepoURL == img.Image {
				targetImage := kustypes.Image{
					Name:    img.Image,
					NewName: img.NewName,
					NewTag:  i.Tag,
				}
				if img.Name != "" {
					targetImage.Name = img.Name
				}
				if img.UseDigest {
					targetImage.Digest = i.Digest
				}
				return targetImage, nil
			}
		}
	}

	return kustypes.Image{}, fmt.Errorf("no matching image found in freight for %q", img.Image)
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
