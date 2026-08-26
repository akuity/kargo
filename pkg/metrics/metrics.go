// Package metrics is a small helper package containing constants and other shared behaviors for
// Kargo metrics. Actual metric definitions should be done in the packages that define and/or
// implement the resources being measured
package metrics

import "time"

const (
	// Namespace is the prefix shared by all Kargo metric names.
	Namespace = "kargo"

	// ProjectLabel is the label under which the name of the Kargo Project a resource
	// belongs to is recorded.
	ProjectLabel = "project"

	// NOTE(thomastaylor312): These are the main constants used for long duration steps and
	// promotions for native histograms. Because of the wide range of time values, native is a
	// better fit. Each of the values below has documentation for why it was chosen.

	// NativeHistogramBucketFactor is the factor passed to HistogramOpts for our long-running
	// metrics. This is 10% growth between buckets, shouldn't give us too many. According to the
	// source code, this is 8 buckets per doubling of the duration
	NativeHistogramBucketFactor = 1.1
	// NativeHistogramMaxBucketNumber is the max number of buckets we support. Assuming we run
	// anywhere from about 2s to 6h, the amount of doubling to cover that range of seconds (i.e.
	// log2(21600s)) gives us about 14 doublings, which is 14 * 8 = 112 buckets. If for some reason
	// a 12 hour value comes in, the same formula gives us about 15 and a half doublings, which is
	// 15.5 * 8 = 124 buckets. We set this to 200 to give us plenty of headroom over that, but not
	// anything that would cause runaway buckets
	//
	// NOTE(thomastaylor312): As far as I could tell from my reading of how this all
	// works (which could be flawed), this should be safe from a memory perspective. I
	// think each max bucket number will be bounded per unique combination of labels at
	// 200. Because each unique combo will have a smaller data set it is unlikely that
	// we will hit the max bucket number. But if we ever start having memory issues with
	// this, this is a good place to start looking
	NativeHistogramMaxBucketNumber = 200
	// NativeHistogramMinResetDuration is the minimum time the histogram can collect before it
	// resets. This is set so we don't avoid resetting the histogram too early in case we hit the
	// max bucket number or any other default limit. Given that some steps can run for a while, we
	// want to make sure we don't reset the histogram too early and lose data.
	NativeHistogramMinResetDuration = time.Hour
)
