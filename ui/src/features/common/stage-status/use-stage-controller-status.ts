import { ControllerStatus } from '@ui/features/common/controller-status/controller-status';
import { useControllerStatus } from '@ui/features/common/controller-status/use-controller-status';
import { Stage } from '@ui/gen/api/v2/models';

// useStageControllerStatus reports whether the controller that reconciles a
// Stage (its `spec.shard`, or — when empty — the cluster's default
// controller) is dead or nonexistent.
export const useStageControllerStatus = (stage: Stage): ControllerStatus =>
  useControllerStatus(stage?.spec?.shard);
