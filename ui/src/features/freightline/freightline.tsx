import { Timestamp } from '@bufbuild/protobuf';
import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import {
  IconDefinition,
  faBoxOpen,
  faThumbTack,
  faTimeline
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import React, { useEffect, useState } from 'react';

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

const FreightIcon = (props: { icon: IconDefinition; hasStages?: boolean; className?: string }) => (
  <FontAwesomeIcon
    icon={props.icon}
    className={`${
      props.hasStages ? 'text-gray-800 opacity-30' : 'text-gray-400'
    } text-base my-auto`}
  />
);

const FreightContent = (props: {
  freight?: Freight;
  hasStages: boolean;
  selected: boolean;
  stage?: Stage;
  stageBackground?: string;
  multi?: boolean;
}) => {
  const { freight, hasStages, selected, stageBackground } = props;
  const [hasCommits, setHasCommits] = useState(false);
  const [hasImages, setHasImages] = useState(false);
  useEffect(() => {
    setHasCommits((freight?.commits || []).length > 0);
    setHasImages((freight?.images || []).length > 0);
  }, [freight]);

  const emptyGray = '#2d3748'; // bg-zinc-800
  const bg = stageBackground ? stageBackground : !hasCommits && !hasImages ? '' : emptyGray;

  return (
    <Tooltip title={props.stage && !selected ? props.stage.metadata?.name : null} placement='right'>
      <div
        className={`my-1 flex-shrink h-full flex items-center justify-center flex-col ${
          selected ? 'w-3 rounded' : 'w-full rounded-md'
        }`}
        style={{ backgroundColor: bg }}
      >
        {!selected && (
          <>
            {hasCommits && <FreightIcon icon={faGit} hasStages={hasStages} />}
            {hasImages && <FreightIcon icon={faDocker} hasStages={hasStages} />}
            {!hasStages && !hasCommits && !hasImages && (
              <FontAwesomeIcon icon={faBoxOpen} className='text-gray-600 text-lg' />
            )}
          </>
        )}
      </div>
    </Tooltip>
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
      } ${selected || pinned ? 'w-36' : 'w-20'}`}
      onClick={() => props.setSelected(!selected)}
    >
      <div className='flex w-full h-full mb-1'>
        <div
          className={`flex flex-col align-center h-full ${
            selected || pinned ? 'justify-start' : 'justify-center w-full'
          }`}
        >
          {(stages || []).map((s) => (
            <FreightContent
              stage={s}
              freight={freight}
              hasStages={true}
              selected={selected || pinned}
              key={s?.metadata?.uid}
              multi={(stages || []).length > 1}
              stageBackground={props.stageColorMap[s?.metadata?.uid || '']}
            />
          ))}
          {(stages || []).length == 0 && (
            <FreightContent freight={freight} hasStages={false} selected={selected || pinned} />
          )}
        </div>
        {(selected || pinned) && (
          <div className='flex flex-col justify-center items-start w-full ml-2 font-mono text-sm'>
            {(freight?.commits || []).map((c) => (
              <div key={c.id} className='flex items center'>
                <FontAwesomeIcon icon={faGit} className='w-10' />
                {c.id.substring(0, 6)}
              </div>
            ))}
            {(freight?.images || []).map((i) => (
              <Tooltip
                className='flex items-center'
                key={`${i.repoUrl}:${i.tag}`}
                title={`${i.repoUrl}:${i.tag}`}
              >
                <FontAwesomeIcon icon={faDocker} className='w-10' />
                <div>{i.tag}</div>
              </Tooltip>
            ))}
            <FontAwesomeIcon
              onClick={() => setPinned(!pinned)}
              icon={faThumbTack}
              size='lg'
              className={`${
                pinned ? 'text-gray-200' : 'text-gray-600'
              } cursor-pointer mx-auto mt-4 hover:text-gray-300`}
            />
          </div>
        )}
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
