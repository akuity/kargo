// GVK string representations mirrored from the Kargo API
// (api/v1alpha1/stage_types.go). The AnalysisRunReference.gvk field carries one
// of these to indicate which kind of run a VerificationInfo references.
export const analysisRunGVKRun = 'argoproj.io/v1alpha1, Kind=AnalysisRun';
export const analysisRunGVKRequest = 'kargo.akuity.io/v1alpha1, Kind=AnalysisRunRequest';

// isAnalysisRunRequest reports whether an AnalysisRunReference points at a Kargo
// AnalysisRunRequest rather than a Rollouts AnalysisRun. Legacy references with
// an empty gvk are treated as Rollouts AnalysisRuns.
export const isAnalysisRunRequest = (gvk?: string): boolean => gvk === analysisRunGVKRequest;
