import { Timestamp } from '@bufbuild/protobuf';
import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faThumbTack, faTimeline } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import React, { useEffect } from 'react';

import { Freight, Stage } from '@ui/gen/v1alpha1/types_pb';

export const Freightline = (props: {
  freight: Freight[];
  stagesPerFreight: { [key: string]: Stage[] };
  stageColorMap: { [key: string]: string };
}) => {
  const [selected, setSelected] = React.useState<string | null>(null);

  const [orderedFreight, setOrderedFreight] = React.useState(props.freight);

  const getSeconds = (ts?: Timestamp): number => Number(ts?.seconds) || 0;

  useEffect(() => {
    const ordered = (props.freight || []).sort(
      (a, b) => getSeconds(b.firstSeen) - getSeconds(a.firstSeen)
    );
    setOrderedFreight(ordered);
  }, [props.freight]);

  return (
    <div className='bg-zinc-900 w-full py-4 px-1 h-56 flex flex-col'>
      <div className='mb-4 text-gray-300 text-sm font-semibold flex items-center ml-12'>
        <FontAwesomeIcon icon={faTimeline} className='mr-2' />
        FREIGHTLINE
      </div>
      <div className='flex w-full h-full items-center'>
        <div className='-rotate-90 text-gray-500 text-sm font-semibold mr-2'>NEW</div>
        {(orderedFreight || []).map((f, i) => {
          const id = f?.id || `${i}`;
          return (
            <FreightItem
              freight={f || undefined}
              key={id}
              setSelected={(s: boolean) => setSelected(s ? id : null)}
              selected={selected == id}
              stages={props.stagesPerFreight[id] || []}
              stageColorMap={props.stageColorMap}
            />
          );
        })}
        <div className='rotate-90 text-gray-500 text-sm font-semibold ml-auto'>OLD</div>
      </div>
    </div>
  );
};

const EmptyFreightLabel = () => <div className='w-full rounded-md bg-zinc-700 h-4' />;

const StageIndicator = (props: { stage: Stage; backgroundColor: string }) => {
  const { stage, backgroundColor } = props;
  return (
    <Tooltip title={stage ? stage.metadata?.name : null} placement='right'>
      <div
        className={`my-1 flex-shrink h-full flex items-center justify-center flex-col w-full rounded`}
        style={{ backgroundColor }}
      />
    </Tooltip>
  );
};

const StageIndicators = (props: { stages: Stage[]; stageColorMap: { [key: string]: string } }) =>
  (props.stages || []).length > 0 ? (
    <div className={`flex flex-col align-center h-full justify-center w-full flex-grow mr-3`}>
      {(props.stages || []).map((s) => (
        <StageIndicator
          stage={s}
          backgroundColor={props.stageColorMap[s?.metadata?.uid || '']}
          key={s?.metadata?.uid}
        />
      ))}
    </div>
  ) : (
    <></>
  );

const FreightContents = (props: {
  freight?: Freight;
  pinned: boolean;
  setPinned: (pinned: boolean) => void;
  selected: boolean;
}) => {
  const { freight, pinned, setPinned, selected } = props;

  const Icon = (props: { icon: IconDefinition }) => (
    <FontAwesomeIcon icon={props.icon} className={`px-1 ${selected || pinned ? 'mr-2' : ''}`} />
  );

  return (
    <div
      className={`flex flex-col justify-center items-start font-mono text-sm flex-shrink min-w-min ${
        selected || pinned ? 'text-white' : 'text-gray-300'
      }`}
    >
      {(freight?.commits || []).map((c) => (
        <Tooltip key={c.id} className='flex items-center my-2' title={`${c.repoUrl} (${c.branch})`}>
          <Icon icon={faGit} />
          {(selected || pinned) && (
            <a
              href={`${c.repoUrl.replace('.git', '')}/commit/${c.id}`}
              target='_blank'
              className='text-blue-200 hover:text-blue-400'
            >
              {c.id.substring(0, 6)}
            </a>
          )}
        </Tooltip>
      ))}
      {(freight?.images || []).map((i) => (
        <Tooltip
          className='flex items-center my-2'
          key={`${i.repoUrl}:${i.tag}`}
          title={`${i.repoUrl}:${i.tag}`}
        >
          <Icon icon={faDocker} />
          {(selected || pinned) && <div>{i.tag}</div>}
        </Tooltip>
      ))}
      {(selected || pinned) && (
        <FontAwesomeIcon
          onClick={() => setPinned(!pinned)}
          icon={faThumbTack}
          size='lg'
          className={`${
            pinned ? 'text-gray-200' : 'text-gray-600'
          } cursor-pointer mx-auto mt-2 hover:text-gray-300`}
        />
      )}
    </div>
  );
};

const FreightItem = (props: {
  freight?: Freight;
  setSelected: (selected: boolean) => void;
  selected: boolean;
  stages: Stage[];
  stageColorMap: { [key: string]: string };
}) => {
  const { freight, selected, stages } = props;
  const [pinned, setPinned] = React.useState(false);

  return (
    <div
      className={`transition-all p-2 cursor-pointer h-full mr-5 rounded-lg border-solid border-2 text-white flex flex-col items-center ${
        selected ? 'border-gray-400' : 'border-gray-700 hover:border-gray-500'
      } ${selected || pinned ? 'w-40' : 'w-20'}`}
      onClick={() => props.setSelected(!selected)}
    >
      <div className='flex w-full h-full mb-1 items-center justify-center'>
        <StageIndicators stages={stages} stageColorMap={props.stageColorMap} />
        <FreightContents
          freight={freight}
          pinned={pinned}
          setPinned={setPinned}
          selected={selected}
        />
      </div>
      <div className='mt-auto w-full'>
        {!freight ? (
          <EmptyFreightLabel />
        ) : (
          <div className='w-full text-center font-mono text-sm'>{freight.id?.substring(0, 6)}</div>
        )}
      </div>
    </div>
  );
};
