import { createContext, useContext } from 'react';

import { ArgoCDShard } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { ShardInfo } from '@ui/gen/api/v2/models';

export type DictionaryContextType = {
  freightInStages: Record<string, Stage[]>;
  freightById: Record<string, Freight>;
  stageAutoPromotionMap: Record<string, boolean>;
  subscribersByStage: Record<string, Set<string>>;
  stageByName: Record<string, Stage>;
  argocdShards?: Record<string, ArgoCDShard>;
  shardsByName?: Record<string, ShardInfo>;
  defaultShardName?: string;
  hasAnalysisRunLogsUrlTemplate?: boolean;
};

export const DictionaryContext = createContext<DictionaryContextType | null>(null);

export const useDictionaryContext = () => useContext(DictionaryContext);
