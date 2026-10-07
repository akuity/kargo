package list

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		source  *Source[int]
		want    []int
		wantErr string
	}{
		{name: "no list function", source: &Source[int]{}, wantErr: "list function is required"},
		{
			name:    "negative jitter",
			source:  New(func(context.Context) ([]int, error) { return nil, nil }).Jitter(-0.1),
			wantErr: "jitter must not be negative",
		},
		{
			name: "failed initial list fails the source",
			source: New(func(context.Context) ([]int, error) {
				return []int{1}, errors.New("store offline")
			}),
			wantErr: "initial list failed: store offline",
		},
		{
			name:   "initial list is pushed before Start returns",
			source: New(func(context.Context) ([]int, error) { return []int{1, 2, 3}, nil }),
			want:   []int{1, 2, 3},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			out := make(chan int, 8)
			err := testCase.source.Start(context.Background(), out)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
			var got []int
			for len(out) > 0 {
				got = append(got, <-out)
			}
			require.Equal(t, testCase.want, got)
		})
	}
}

// TestEvery checks that the list repeats on the interval, keeps going after a
// failed list, and stops with the context.
func TestEvery(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	source := New(func(context.Context) ([]int, error) {
		n := calls.Add(1)
		if n == 2 {
			return nil, errors.New("transient")
		}
		return []int{int(n)}, nil
	}).Every(5 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan int, 64)
	require.NoError(t, source.Start(ctx, out))
	require.Equal(t, 1, <-out) // The initial list.
	require.Eventually(t, func() bool { return calls.Load() >= 4 }, 5*time.Second, time.Millisecond)
	cancel()
	// No further lists once the context is done.
	time.Sleep(20 * time.Millisecond)
	settled := calls.Load()
	time.Sleep(20 * time.Millisecond)
	require.Equal(t, settled, calls.Load())
	// The failed list pushed nothing; the ones around it did.
	var got []int
	for len(out) > 0 {
		got = append(got, <-out)
	}
	require.NotContains(t, got, 2)
	require.Contains(t, got, 3)
}

func TestNextInterval(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		every  time.Duration
		jitter float64
		random float64
		want   time.Duration
	}{
		{name: "no jitter", every: time.Minute, random: 0.99, want: time.Minute},
		{name: "jitter at the low end adds nothing", every: time.Minute, jitter: 0.5, random: 0, want: time.Minute},
		{name: "jitter scales with the random draw", every: time.Minute, jitter: 0.5, random: 0.5, want: 75 * time.Second},
		{
			name: "jitter stays below the factor", every: time.Minute, jitter: 0.5, random: 0.999,
			want: 89*time.Second + 970*time.Millisecond,
		},
		{name: "jitter above one is allowed", every: 10 * time.Second, jitter: 2, random: 0.5, want: 20 * time.Second},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			source := New(func(context.Context) ([]int, error) { return nil, nil }).
				Every(testCase.every).
				Jitter(testCase.jitter)
			source.random = func() float64 { return testCase.random }
			require.Equal(t, testCase.want, source.nextInterval())
		})
	}
}

// TestJitterIsRandom checks that the real random source produces intervals
// that vary within the configured bounds.
func TestJitterIsRandom(t *testing.T) {
	t.Parallel()
	source := New(func(context.Context) ([]int, error) { return nil, nil }).
		Every(time.Minute).
		Jitter(0.5)
	seen := make(map[time.Duration]struct{})
	for range 100 {
		d := source.nextInterval()
		require.GreaterOrEqual(t, d, time.Minute)
		require.Less(t, d, 90*time.Second)
		seen[d] = struct{}{}
	}
	require.Greater(t, len(seen), 1)
}
