// extended types for protobufs
// some protobufs have generic types ie. JSON but we know the exact types

import { RepoSubscription, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

export type WarehouseExpanded = Omit<Warehouse, 'spec'> & {
  spec?: Omit<Warehouse['spec'], 'subscriptions'> & {
    subscriptions?: RepoSubscription[];
  };
};
