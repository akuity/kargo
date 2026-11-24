// Kind string representations mirrored from the Kargo API
// (api/v1alpha1/stage_types.go). The AnalysisRunReference.kind field carries one
// of these to indicate which kind of run a VerificationInfo references.
export const analysisRunKindRun = 'argoproj.io/v1alpha1, Kind=AnalysisRun';

// isAnalysisRun reports whether an AnalysisRunReference points at an Argo Rollouts AnalysisRun
// rather than a Kargo EE extension.
export const isAnalysisRun = (kind?: string): boolean =>
  kind === analysisRunKindRun || kind == '' || kind == null;
