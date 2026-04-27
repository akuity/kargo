import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { HeartbeatStatus } from '@ui/gen/api/v2/models';

// useStageControllerStatus resolves which controller a Stage is reconciled
// by (its `spec.shard`, or — when empty — the cluster's default controller
// name) and reports whether that controller's heartbeat indicates it is
// dead.
//
// Treatment of missing heartbeats: Kargo controllers always produce
// heartbeats, so once the heartbeats query has resolved, absence of an
// entry for a controller name means that controller is dead or
// nonexistent. While the query is still in flight the result is undefined,
// in which case this hook reports isControllerDead=false to avoid flashing
// every Stage as Failed during initial page load.
export const useStageControllerStatus = (
  stage: Stage
): {
  controllerName: string;
  isControllerDead: boolean;
} => {
  const dictionaryContext = useDictionaryContext();
  const controllerName = stage?.spec?.shard || dictionaryContext?.defaultControllerName || '';

  const heartbeats = dictionaryContext?.heartbeatsByController;
  if (!heartbeats) {
    // Query has not resolved yet; do not synthesize Failed.
    return { controllerName, isControllerDead: false };
  }

  const heartbeat = heartbeats[controllerName];
  return {
    controllerName,
    isControllerDead: !heartbeat || heartbeat.status === HeartbeatStatus.StatusDead
  };
};
