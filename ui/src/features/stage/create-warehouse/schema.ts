/* eslint-disable @typescript-eslint/ban-ts-comment */
// @ts-nocheck
/**
 * transforms Warehouse CRD with only properties that user needs to care of when creating Warehouse
 */
import defaultWarehouseFormJSONSchema from '@ui/gen/schema/warehouses.kargo.akuity.io_v1alpha1.json';

const warehouseCreateFormJSONSchema = { ...defaultWarehouseFormJSONSchema };

warehouseCreateFormJSONSchema.properties = warehouseCreateFormJSONSchema.properties.spec.properties;

delete warehouseCreateFormJSONSchema.required;
delete warehouseCreateFormJSONSchema.type;
delete warehouseCreateFormJSONSchema.description;
delete warehouseCreateFormJSONSchema.properties.shard;

export { warehouseCreateFormJSONSchema };
