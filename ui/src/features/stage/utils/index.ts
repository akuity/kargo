import { ObjectMeta } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';

const COLOR_ANNOTATION = 'kargo.akuity.io/color';

interface HasMetadata {
  metadata?: ObjectMeta;
}

export function parseColorAnnotation<T extends HasMetadata>(object: T): string | null {
  const annotations = object?.metadata?.annotations;
  if (!annotations) {
    return null;
  }
  const color = annotations[COLOR_ANNOTATION] as string;
  if (!color) {
    return null;
  }
  return color;
}

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

export const WarehouseColorMapHex: { [key: string]: string } = {
  red: '#D70015',
  orange: '#C93500',
  yellow: '#B24F01',
  green: '#248A3D',
  mint: '#0E817C',
  teal: '#028299',
  cyan: '#0471A4',
  blue: '#013fDC',
  indigo: '#3634A3',
  purple: '#8944AA',
  pink: '#D21043',
  brown: '#7F6545'
};

export function getColors<T extends HasMetadata>(
  project: string,
  objects: T[],
  key?: string
): ColorMap {
  // check local storage
  const colors = localStorage.getItem(`${project}/colors${key ? `/${key}` : ''}`);
  if (colors) {
    const m = JSON.parse(colors);
    if (Object.keys(m).length === objects.length) {
      for (const stage of objects) {
        const color = parseColorAnnotation(stage);
        if (color) {
          m[stage?.metadata?.name || ''] = ColorMapHex[color];
        }
      }
      return m;
    } else {
      return setColors(project, objects, m, key);
    }
  } else {
    return setColors(project, objects, undefined, key);
  }
}

export function setColors<T extends HasMetadata>(
  project: string,
  stages: T[],
  prevMap?: ColorMap,
  key?: string
): ColorMap {
  const colors = generateStageColors(stages, prevMap);
  localStorage.setItem(`${project}/colors${key ? `/${key}` : ''}`, JSON.stringify(colors));
  return colors;
}

export const clearColors = (project: string, key?: string) => {
  localStorage.removeItem(`${project}/colors${key ? `/${key}` : ''}`);
};

export function generateStageColors<T extends HasMetadata>(sortedObjects: T[], prevMap?: ColorMap) {
  const curColors = { ...ColorMapHex };
  let finalMap: { [key: string]: string } = {};

  if (prevMap && Object.keys(prevMap).length > 0) {
    for (const color of Object.values(prevMap)) {
      delete curColors[color];
    }
    finalMap = { ...prevMap };
  }

  for (const stage of sortedObjects) {
    const color = parseColorAnnotation(stage);
    if (color) {
      delete curColors[color];
      finalMap[stage?.metadata?.name || ''] = ColorMapHex[color];
    }
  }
  const colors = Object.values(curColors);
  let step = Math.floor(colors.length / sortedObjects.length);
  if (step < 1) {
    step = 1;
  }
  let i = 0;
  for (const object of sortedObjects) {
    const id = object?.metadata?.name;
    if (!id || finalMap[id]) {
      continue;
    }
    finalMap[id] = colors[i];
    i = i + step;
  }
  return finalMap;
}
