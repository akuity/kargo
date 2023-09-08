import { Stage } from '@ui/gen/v1alpha1/types_pb';

export const getStageKey = (stage: Stage): number => {
  for (const c of stage?.metadata?.uid?.split('') || []) {
    if (parseInt(c)) {
      return parseInt(c);
    }
  }
  return 0;
};

export const getStageBackground = (stage: Stage) => {
  const key = getStageKey(stage);
  switch (key) {
    case 0:
      return 'bg-red-400';
    case 1:
      return 'bg-orange-400';
    case 2:
      return 'bg-amber-400';
    case 3:
      return 'bg-yellow-400';
    case 4:
      return 'bg-green-400';
    case 5:
      return 'bg-teal-400';
    case 6:
      return 'bg-blue-400';
    case 7:
      return 'bg-indigo-400';
    case 8:
      return 'bg-violet-400';
    case 9:
      return 'bg-pink-500';
  }
};
