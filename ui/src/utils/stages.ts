import { Stage } from '@ui/gen/v1alpha1/types_pb';

const COLOR_ANNOTATION = 'kargo.akuity.io/color';

export const parseColorAnnotation = (stage: Stage): string | null => {
  const annotations = stage?.metadata?.annotations;
  if (!annotations) {
    return null;
  }
  const color = annotations[COLOR_ANNOTATION] as string;
  if (!color) {
    return null;
  }
  return color;
};

export const ColorMapHex: { [key: string]: string } = {
  red: '#EF4444', // 'bg-red-500',
  orange: '#F97316', // 'bg-orange-400',
  yellow: '#FCD34D', // 'bg-yellow-400',
  lime: '#84CC16', // 'bg-lime-400',
  green: '#22C55E', // 'bg-green-400',
  teal: '#06B6D4', // 'bg-teal-500',
  cyan: '#22D3EE', // 'bg-cyan-400',
  sky: '#60A5FA', // 'bg-sky-500',
  blue: '#3B82F6', // 'bg-blue-500',
  violet: '#8B5CF6', // 'bg-violet-500',
  purple: '#A855F7', // 'bg-purple-500',
  fuchsia: '#D946EF', // 'bg-fuchsia-500',
  pink: '#EC4899', // 'bg-pink-500',
  rose: '#F43F5E' // 'bg-rose-400'
};

export const ColorMap: { [key: string]: string } = {
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
  const curColors = { ...ColorMapHex };
  const sorted = stages.sort((a, b) => {
    return (a?.metadata?.name || '') > (b?.metadata?.name || '') ? 1 : -1;
  });
  const finalMap: { [key: string]: string } = {};

  for (const stage of sorted) {
    const color = parseColorAnnotation(stage);
    if (color) {
      delete curColors[color];
      finalMap[stage?.metadata?.uid || ''] = ColorMapHex[color];
    }
  }
  const colors = Object.values(curColors);
  let i = 0;
  for (const stage of sorted) {
    const id = stage?.metadata?.uid;
    if (!id || finalMap[id]) {
      continue;
    }
    finalMap[id] = colors[i];
    i = i + (1 % colors.length);
  }
  return finalMap;
};
