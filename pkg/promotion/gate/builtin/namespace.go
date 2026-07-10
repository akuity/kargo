package builtin

import (
	"context"
	"errors"
	"fmt"

	"github.com/akuity/kargo/pkg/promotion/gate/types"
)

// NamespaceGateName is the name of the Promotion creation gate that enforces
// that Freight and its target Stage reside in the same namespace (Project).
const NamespaceGateName = "namespace"

type namespaceGate struct{}

// NewNamespaceGate returns a PromotionGate that denies Freight whose namespace
// differs from the target Stage's namespace.
func NewNamespaceGate() types.PromotionGate {
	return &namespaceGate{}
}

func (g *namespaceGate) Name() string {
	return NamespaceGateName
}

func (g *namespaceGate) Evaluate(
	_ context.Context,
	input types.PromotionInput,
) (*types.Decision, error) {
	if input.Stage == nil {
		return nil, errors.New("stage is nil")
	}
	if input.Freight == nil {
		return nil, errors.New("freight is nil")
	}
	if input.Stage.Namespace != input.Freight.Namespace {
		return types.NewDenyDecision().WithMessage(fmt.Sprintf(
			"Freight %q is in namespace %q, but Stage %q is in namespace %q",
			input.Freight.Name,
			input.Freight.Namespace,
			input.Stage.Name,
			input.Stage.Namespace,
		)), nil
	}
	return types.NewAllowDecision(), nil
}
