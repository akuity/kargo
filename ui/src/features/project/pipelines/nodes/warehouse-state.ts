import { RepoSubscription, WarehouseExpanded } from '@ui/extend/types';

export type WarehousePhase = 'ready' | 'discovering' | 'failed';

export type WarehouseState = {
  phase: WarehousePhase;
  // What a failed Warehouse heads its error with, and the error itself.
  errorTitle?: string;
  errorMessage?: string;
  // The condition message as the controller wrote it. A failing subscription
  // is named in here, if anywhere.
  rawErrorMessage?: string;
  discoveredAt?: Date;
};

const refreshAnnotation = 'kargo.akuity.io/refresh';

// Reasons the controller gives for a Warehouse that discovered its artifacts
// but could not make Freight of them. Every other reason is a discovery
// failure of one kind or another.
const freightReasons = [
  'CriteriaEvaluationFailed',
  'FreightBuildFailure',
  'FreightCreationFailure'
];

// The prefixes the controller puts ahead of a discovery error. The node heads
// the message with "Discovery failed" already, so they would only say it twice.
const redundantPrefixes = ['Unable to discover artifacts: ', 'error discovering artifacts: '];

const trimMessage = (message: string) => {
  let trimmed = message;
  for (const prefix of redundantPrefixes) {
    if (trimmed.startsWith(prefix)) {
      trimmed = trimmed.slice(prefix.length);
    }
  }
  return trimmed;
};

export const getWarehouseState = (warehouse?: WarehouseExpanded): WarehouseState => {
  const conditions = warehouse?.status?.conditions || [];

  const reconciling = conditions.some((c) => c.type === 'Reconciling' && c.status === 'True');
  const hasReconcilingCondition = conditions.some((c) => c.type === 'Reconciling');

  // A refresh has been asked for and the controller has not got to it yet.
  const refreshToken = warehouse?.metadata?.annotations?.[refreshAnnotation];
  const refreshing =
    typeof refreshToken === 'string' && refreshToken !== warehouse?.status?.lastHandledRefresh;

  const discoveredAtValue = warehouse?.status?.discoveredArtifacts?.discoveredAt;
  const discoveredAt = discoveredAtValue ? new Date(discoveredAtValue) : undefined;

  // A refresh asked for since the last failure is a retry, and it is under way
  // whatever the failure on record says.
  if (refreshing) {
    return { phase: 'discovering', discoveredAt };
  }

  const unhealthy = conditions.find((c) => c.type === 'Healthy' && c.status === 'False');
  const notReady = conditions.find((c) => c.type === 'Ready' && c.status === 'False');

  // Unhealthy outranks reconciling. A failed discovery, or a failed attempt
  // to build Freight, returns without clearing the Reconciling condition it
  // set on the way in, so a failed Warehouse goes on reporting that it is
  // reconciling. Work that really is under way first sets Healthy to Unknown.
  //
  // Not ready on its own is only a failure once reconciling has stopped
  // reporting: while it reports, not ready means not ready yet.
  const failure = unhealthy || (!hasReconcilingCondition ? notReady : undefined);

  if (!failure) {
    return { phase: reconciling ? 'discovering' : 'ready', discoveredAt };
  }

  return {
    phase: 'failed',
    errorTitle: freightReasons.includes(failure.reason)
      ? 'Freight creation failed'
      : 'Discovery failed',
    errorMessage: trimMessage(failure.message || ''),
    rawErrorMessage: failure.message || '',
    discoveredAt
  };
};

// The least time the loader and the edge animation are shown for, so that a
// quick discovery does not blink.
export const minDiscoveringMs = 2000;

export const subscriptionRepoURL = (subscription: RepoSubscription) =>
  subscription?.git?.repoURL || subscription?.chart?.repoURL || subscription?.image?.repoURL || '';

// Discovery stops at the first subscription it fails on, and every subscriber
// quotes the repository it failed on in its error. A subscription is the
// failing one if its own repository is the one quoted.
export const isSubscriptionFailing = (state: WarehouseState, subscription: RepoSubscription) => {
  const repoURL = subscriptionRepoURL(subscription);

  return (
    state.phase === 'failed' &&
    Boolean(repoURL) &&
    Boolean(state.rawErrorMessage?.includes(`"${repoURL}"`))
  );
};
