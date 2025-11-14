import { Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

export const getWarehouseError = (warehouse: Warehouse): string | null => {
  let message: string | null = null;

  const conditions = warehouse?.status?.conditions || [];

  for (const condition of conditions) {
    if (condition?.type === 'Healthy' && condition?.status === 'False') {
      message = condition?.message;
    }

    if (condition?.type === 'Ready' && condition?.status === 'False') {
      message = condition?.message;
    }
  }

  return message;
};
