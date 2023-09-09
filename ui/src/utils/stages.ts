import { Stage } from '@ui/gen/v1alpha1/types_pb';

const COLOR_ANNOTATION = 'ui.kargo.akuity.io/color';

export const parseColorAnnotation = (stage: Stage) => {
  const annotations = stage?.metadata?.annotations;
  if (!annotations) {
    return null;
  }
  const color = annotations[COLOR_ANNOTATION];
  if (!color) {
    return null;
  }
  return color;
};

export const ColorMap = {
  red: 'bg-red-500',
  orange: 'bg-orange-400',
  yellow: 'bg-yellow-400',
  lime: 'bg-lime-400',
  green: 'bg-green-400',
  teal: 'bg-teal-500',
  cyan: 'bg-cyan-400',
  sky: 'bg-sky-500',
  blue: 'bg-blue-500',
  violet: 'bg-violet-500',
  purple: 'bg-purple-500',
  fuchsia: 'bg-fuchsia-500',
  pink: 'bg-pink-500',
  rose: 'bg-rose-400'
};

export const getBackground = (n: number) => {
  if (n < 0 || n >= Object.keys(ColorMap).length) {
    return 'bg-gray-500';
  }
  return Object.values(ColorMap)[n];
};

export const getStageColors = (stages: Stage[]) => {
  const colors = Object.values(ColorMap);
  const map: { [key: string]: string } = {};
  let i = 0;
  for (const stage of stages) {
    const id = stage?.metadata?.uid;
    if (!id) {
      continue;
    }
    map[id] = colors[i];
    i = i + (1 % colors.length);
  }
  return map;
};
