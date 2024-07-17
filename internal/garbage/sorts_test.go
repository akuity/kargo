package garbage

import (
	"math/rand"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPromosByCreation(t *testing.T) {
	// Build a list of Promotions with random CreationTimestamps
	const numPromos = 100
	now := time.Now()
	rnd := rand.New(rand.NewSource(now.UnixNano())) // nolint: gosec
	promos := make(promosByCreation, numPromos)
	for i := range promos {
		promos[i].ObjectMeta.CreationTimestamp =
			metav1.NewTime(now.Add(time.Duration(rnd.Intn(100)) * time.Minute))
	}
	sort.Sort(promos)
	for i := 0; i < numPromos-1; i++ {
		curTime := promos[i].ObjectMeta.CreationTimestamp.Time
		nextTime := promos[i+1].ObjectMeta.CreationTimestamp.Time
		require.True(
			t,
			curTime.After(nextTime) || curTime.Equal(nextTime),
		)
	}
}

func TestFreightByCreation(t *testing.T) {
	// Build a list of Freight with random CreationTimestamps
	const numFreight = 100
	now := time.Now()
	rnd := rand.New(rand.NewSource(now.UnixNano())) // nolint: gosec
	freight := make(freightByCreation, numFreight)
	for i := range freight {
		freight[i].ObjectMeta.CreationTimestamp =
			metav1.NewTime(now.Add(time.Duration(rnd.Intn(100)) * time.Minute))
	}
	sort.Sort(freight)
	for i := 0; i < numFreight-1; i++ {
		curTime := freight[i].ObjectMeta.CreationTimestamp.Time
		nextTime := freight[i+1].ObjectMeta.CreationTimestamp.Time
		require.True(
			t,
			curTime.After(nextTime) || curTime.Equal(nextTime),
		)
	}
}
