import { createContext, useContext } from 'react';

import { ArgoCDShard } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { Heartbeat } from '@ui/gen/api/v2/models';

export type DictionaryContextType = {
  freightInStages: Record<string, Stage[]>;
  freightById: Record<string, Freight>;
  stageAutoPromotionMap: Record<string, boolean>;
  subscribersByStage: Record<string, Set<string>>;
  stageByName: Record<string, Stage>;
  argocdShards?: Record<string, ArgoCDShard>;
  // heartbeatsByController is undefined while the heartbeats query is still
  // loading, and a map (possibly empty) once it has resolved. Consumers must
  // distinguish these states; absence of a key in the map after load means
  // the named controller is dead or nonexistent.
  heartbeatsByController?: Record<string, Heartbeat>;
  defaultControllerName?: string;
  hasAnalysisRunLogsUrlTemplate?: boolean;
};

export const DictionaryContext = createContext<DictionaryContextType | null>(null);

export const useDictionaryContext = () => useContext(DictionaryContext);
