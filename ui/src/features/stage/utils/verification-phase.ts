// read more in stage_types.go
enum VerificationPhase {
  Pending = 'Pending',
  Running = 'Running',
  Successful = 'Successful',
  Failed = 'Failed',
  Error = 'Error',
  Aborted = 'Aborted',
  Inconclusive = 'Inconclusive'
}

export const verificationPhaseIsTerminal = (phase: string) => {
  switch (phase) {
    case VerificationPhase.Successful:
    case VerificationPhase.Failed:
    case VerificationPhase.Error:
    case VerificationPhase.Aborted:
    case VerificationPhase.Inconclusive:
      return true;
    default:
      return false;
  }
};
