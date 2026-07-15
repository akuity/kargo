import { faArrowRotateRight, faSpinner } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

function range(start: number, end?: number, step = 1): number[] {
  let _start = start,
    _end = end,
    _step = step;
  const result: number[] = [];
  if (_end === undefined) {
    _end = _start;
    _start = 0;
  }
  if (_step === 0) throw new Error('Step cannot be zero');
  const ascending = _step > 0;
  if (ascending) {
    for (let i = _start; i < _end; i += _step) {
      result.push(i);
    }
  } else {
    for (let i = _start; i > (_end as number); i += _step) {
      result.push(i);
    }
  }
  return result;
}

type FleetTarget = {
  name: string;
  version: string;
  promotionInProgress: boolean;
  isHealthy: boolean;
};

const useGetFleetTargets = (): FleetTarget[] => {
  return range(1, 300)
    .map((number) => ({
      name: `target-${number}`,
      version: `1.0.${number % 10}`,
      promotionInProgress: number % 7 === 0,
      isHealthy: number % 5 === 0
    }))
    .sort((a, b) => b.version.localeCompare(a.version));
};

const versionColors = [
  'bg-blue-400',
  'bg-cyan-400',
  'bg-emerald-400',
  'bg-green-400',
  'bg-lime-400',
  'bg-amber-400',
  'bg-orange-400',
  'bg-rose-400',
  'bg-fuchsia-400',
  'bg-violet-400'
];

const getVersionColor = (version: string): string => {
  const hash = Array.from(version).reduce((total, character) => total + character.charCodeAt(0), 0);
  return versionColors[hash % versionColors.length];
};

type VersionDistribution = {
  version: string;
  targetCount: number;
  percentage: number;
};

const getVersionDistribution = (targets: FleetTarget[]): VersionDistribution[] => {
  if (targets.length === 0) {
    return [];
  }

  const targetsByVersion = targets.reduce<Record<string, number>>((distribution, target) => {
    distribution[target.version] = (distribution[target.version] || 0) + 1;
    return distribution;
  }, {});

  return Object.entries(targetsByVersion).map(([version, targetCount]) => ({
    version,
    targetCount,
    percentage: (targetCount / targets.length) * 100
  }));
};

const VersionBadge = ({ version }: { version: string }) => {
  return (
    <div key={version} className='flex items-center gap-1'>
      <span className={`${getVersionColor(version)} h-2 w-2 rounded-full`} />
      <span>{version}</span>
    </div>
  );
};

const FleetVersionProgress = ({ targets }: { targets: FleetTarget[] }) => {
  const distribution = getVersionDistribution(targets);

  return (
    <div className='mb-3'>
      <div
        className='flex h-3 overflow-hidden rounded-sm bg-gray-200'
        role='progressbar'
        aria-label='Fleet target versions'
      >
        {distribution.map(({ version, targetCount, percentage }) => (
          <div
            key={version}
            className={`${getVersionColor(version)} first:rounded-l-sm last:rounded-r-sm`}
            style={{ width: `${percentage}%` }}
            title={`${version}: ${percentage.toFixed(1)}% (${targetCount} targets)`}
          />
        ))}
      </div>

      <div className='mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-600'>
        {distribution.map(({ version }) => (
          <VersionBadge version={version} />
        ))}
      </div>
    </div>
  );
};

export const Fleet = () => {
  const targets = useGetFleetTargets();
  return (
    <div>
      <FleetVersionProgress targets={targets} />
      <div className='grid grid-cols-[repeat(50,minmax(0,1fr))] md:gap-1 gap-0.5'>
        {targets.map((t) => (
          <div
            key={t.name}
            className={`${getVersionColor(t.version)} flex items-center justify-center rounded-md text-xs text-center aspect-square`}
            title={t.version}
          >
            {t.promotionInProgress && (
              <FontAwesomeIcon
                icon={faArrowRotateRight}
                className='h-1/2 w-1/2 animate-spin text-white'
              />
            )}
          </div>
        ))}
      </div>
    </div>
  );
};
