import { Stage } from '@ui/gen/v1alpha1/generated_pb';

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

export type ColorMap = { [key: string]: string };

export const ColorMapHex: { [key: string]: string } = {
  red: '#ED204E',
  salmon: '#FD5352',
  orange: '#FE7537',
  amber: '#e78a00',
  yellow: '#DFC546',
  lime: '#9bce22',
  avocado: '#84DF75',
  green: '#1CAC77',
  teal: '#1bc1a7',
  cyan: '#1DCECA',
  sky: '#0DAFD3',
  blue: '#3882EA',
  indigo: '#2D5EDC',
  periwinkle: '#6380E1',
  violet: '#7851AA',
  purple: '#A9499D',
  fuchsia: '#D0469D',
  pink: '#E573A2',
  rose: '#f1619b',
  dragonfruit: '#FE43A3',
  gray: '#6a7382'
};

export const getBackgroundKey = (n: number) => {
  if (n < 0 || n >= Object.keys(ColorMapHex).length) {
    return 'gray';
  }
  return Object.keys(ColorMapHex)[n];
};

export const getStageColors = (project: string, stages: Stage[]): ColorMap => {
  // check local storage
  const colors = localStorage.getItem(`${project}/colors`);
  const count = parseInt(localStorage.getItem(`${project}/stageCount`) || '0');
  if (colors) {
    const m = JSON.parse(colors);
    if (count === stages.length) {
      for (const stage of stages) {
        const color = parseColorAnnotation(stage);
        if (color) {
          m[stage?.metadata?.name || ''] = ColorMapHex[color];
        }
      }
      return m;
    } else {
      return setStageColors(project, stages, m);
    }
  } else {
    return setStageColors(project, stages);
  }
};

export const setStageColors = (project: string, stages: Stage[], prevMap?: ColorMap): ColorMap => {
  const colors = generateStageColors(stages, prevMap);
  localStorage.setItem(`${project}/colors`, JSON.stringify(colors));
  localStorage.setItem(`${project}/stageCount`, `${stages.length}`);
  return colors;
};

export const clearColors = (project: string) => {
  localStorage.removeItem(`${project}/colors`);
  localStorage.removeItem(`${project}/stageCount`);
};

export const generateStageColors = (sortedStages: Stage[], prevMap?: ColorMap) => {
  const curColors = { ...ColorMapHex };
  let finalMap: { [key: string]: string } = {};

  if (prevMap) {
    for (const color of Object.values(prevMap)) {
      delete curColors[color];
    }
    finalMap = { ...prevMap };
  }

  for (const stage of sortedStages) {
    const color = parseColorAnnotation(stage);
    if (color) {
      delete curColors[color];
      finalMap[stage?.metadata?.name || ''] = ColorMapHex[color];
    }
  }
  const colors = Object.values(curColors);
  let step = Math.floor(colors.length / sortedStages.length);
  if (step < 1) {
    step = 1;
  }
  let i = 0;
  for (const stage of sortedStages) {
    const id = stage?.metadata?.name;
    if (!id || finalMap[id]) {
      continue;
    }
    finalMap[id] = colors[i];
    i = i + step;
  }
  return finalMap;
};
