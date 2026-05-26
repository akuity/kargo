// extended types for protobufs
// some protobufs have generic types ie. JSON but we know the exact types

import { Warehouse } from '@ui/gen/api/v2/models';

// export type RepoSubscription = {
//   [key: string]:
// };

export type WarehouseExpanded = Omit<Warehouse, 'spec'> & {
  spec?: Omit<Warehouse['spec'], 'subscriptions'> & {
    subscriptions?: RepoSubscription[];
  };
};
