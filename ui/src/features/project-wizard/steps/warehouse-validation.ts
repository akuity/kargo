import { dnsRegex } from '@ui/features/common/utils';

import { WarehouseDraft } from '../types';

// Warehouse names are Kubernetes resource names (DNS subdomain). Matches the
// leniency of the app's CreateWarehouseWizard, which does not otherwise gate.
export const isValidWarehouse = (w: WarehouseDraft) => dnsRegex.test(w.name);
