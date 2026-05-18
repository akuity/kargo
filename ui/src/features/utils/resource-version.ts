import type { ObjectMeta } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';

export const isSameOrOlderResourceVersion = <T extends { metadata?: ObjectMeta }>(
  current: T | undefined,
  next: T | undefined
): boolean => {
  const currentResourceVersion = current?.metadata?.resourceVersion;
  const nextResourceVersion = next?.metadata?.resourceVersion;
  if (!currentResourceVersion || !nextResourceVersion) {
    return false;
  }
  if (currentResourceVersion === nextResourceVersion) {
    return true;
  }
  try {
    return BigInt(nextResourceVersion) <= BigInt(currentResourceVersion);
  } catch {
    return false;
  }
};
