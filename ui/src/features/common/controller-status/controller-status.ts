import { Heartbeat, HeartbeatStatus } from '@ui/gen/api/v2/models';

export type ControllerStatus = {
  controllerName: string;
  isControllerDead: boolean;
};

// getControllerStatus resolves which controller reconciles a resource on the
// given shard (the shard itself, or — when empty — the cluster's default
// controller name) and reports whether that controller's heartbeat indicates
// it is dead.
//
// Treatment of missing heartbeats: Kargo controllers always produce
// heartbeats, so once the heartbeats query has resolved, absence of an entry
// for a controller name means that controller is dead or nonexistent. While
// the query is still in flight `heartbeats` is undefined, and this reports
// isControllerDead=false to avoid flashing every resource as failed during
// initial page load.
export const getControllerStatus = (
  shard: string | undefined,
  heartbeats: Record<string, Heartbeat> | undefined,
  defaultControllerName: string | undefined
): ControllerStatus => {
  const controllerName = shard || defaultControllerName || '';

  if (!heartbeats) {
    return { controllerName, isControllerDead: false };
  }

  const heartbeat = heartbeats[controllerName];

  return {
    controllerName,
    isControllerDead: !heartbeat || heartbeat.status === HeartbeatStatus.StatusDead
  };
};

export const deadControllerReason = (controllerName: string) =>
  controllerName
    ? `Controller '${controllerName}' is dead or nonexistent`
    : 'The default controller is dead or nonexistent';
