package git

// FilterBlobless is a filter that excludes blobs from the clone. When using
// this filter, the initial Git clone will download all reachable commits and
// trees, and only download the blobs for commits when you do a Git checkout
// (including the first checkout during the clone).
//
// When using a blobless clone, you can still explore the commits of the
// repository without downloading additional data. This means that you can
// perform commands like `git log`, or even `git log -- <path>` with the same
// performance as a full clone.
//
// Commands like `git diff` or `git blame <path>` require the contents of the
// paths to compute diffs, so these will trigger blob downloads the first time
// they are run.
const FilterBlobless = "blob:none"
