/* eslint-disable @typescript-eslint/ban-ts-comment */
// @ts-nocheck
/**
 * transforms Warehouse CRD with only properties that user needs to care of when creating Warehouse
 */
import defaultWarehouseFormJSONSchema from '@ui/gen/schema/warehouses.kargo.akuity.io_v1alpha1.json';

let warehouseCreateFormJSONSchema = { ...defaultWarehouseFormJSONSchema };

warehouseCreateFormJSONSchema.properties = warehouseCreateFormJSONSchema.properties.spec.properties;

delete warehouseCreateFormJSONSchema.required;
delete warehouseCreateFormJSONSchema.type;
delete warehouseCreateFormJSONSchema.description;
delete warehouseCreateFormJSONSchema.properties.shard;

const removePropertiesRecursively = (schema, props) => {
  // remove keys
  for (const prop of props) {
    if (schema?.[prop]) {
      delete schema[prop];
    }
  }

  // recurse
  for (const [key, value] of Object.entries(schema || {})) {
    if (typeof value === 'object') {
      schema[key] = removePropertiesRecursively(value, props);
    }
  }

  return schema;
};

warehouseCreateFormJSONSchema = removePropertiesRecursively(warehouseCreateFormJSONSchema, [
  'default'
]);

export { warehouseCreateFormJSONSchema };
