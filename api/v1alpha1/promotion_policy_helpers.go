package v1alpha1

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPromotionPolicy returns a pointer to the PromotionPolicy resource
// specified by the namespacedName argument. If no such resource is found, nil
// is returned instead.
func GetPromotionPolicy(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*PromotionPolicy, error) {
	policy := PromotionPolicy{}
	if err := c.Get(ctx, namespacedName, &policy); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting PromotionPolicy %q in namespace %q",
			namespacedName.Name,
			namespacedName.Namespace,
		)
	}
	return &policy, nil
}
