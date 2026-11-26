import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';

import { ControllerStatus, getControllerStatus } from './controller-status';

// useControllerStatus reports whether the controller responsible for the
// given shard is alive, from the heartbeats the pipeline page loads. See
// getControllerStatus for how a missing heartbeat is treated.
export const useControllerStatus = (shard: string | undefined): ControllerStatus => {
  const dictionaryContext = useDictionaryContext();

  return getControllerStatus(
    shard,
    dictionaryContext?.heartbeatsByController,
    dictionaryContext?.defaultControllerName
  );
};
