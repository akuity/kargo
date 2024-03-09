package garbage

import kargoapi "github.com/akuity/kargo/api/v1alpha1"

// promosByCreation implements sort.Interface for []kargoapi.Promotion.
type promosByCreation []kargoapi.Promotion

func (p promosByCreation) Len() int {
	return len(p)
}

func (p promosByCreation) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p promosByCreation) Less(i, j int) bool {
	return p[i].ObjectMeta.CreationTimestamp.Time.After(
		p[j].ObjectMeta.CreationTimestamp.Time,
	)
}

// freightByCreation implements sort.Interface for []kargoapi.Freight.
type freightByCreation []kargoapi.Freight

func (p freightByCreation) Len() int {
	return len(p)
}

func (p freightByCreation) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p freightByCreation) Less(i, j int) bool {
	return p[i].ObjectMeta.CreationTimestamp.Time.After(
		p[j].ObjectMeta.CreationTimestamp.Time,
	)
}
