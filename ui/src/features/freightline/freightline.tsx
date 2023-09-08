import { Timestamp } from '@bufbuild/protobuf';
import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faBoxOpen, faTimeline } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import React, { useEffect, useState } from 'react';

import { Freight, Stage } from '@ui/gen/v1alpha1/types_pb';
import { getStageBackground } from '@ui/utils/stages';

export const Freightline = (props: {
  freight: Freight[];
  stagesPerFreight: { [key: string]: Stage[] };
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
            />
          );
        })}
        <div className='rotate-90 text-gray-500 text-sm font-semibold ml-auto'>OLD</div>
      </div>
    </div>
  );
};

const EmptyFreightLabel = () => <div className='w-full rounded-md bg-zinc-700 h-4' />;

const FreightIcon = (props: { icon: IconDefinition; hasStages?: boolean }) => (
  <FontAwesomeIcon
    icon={props.icon}
    className={`${props.hasStages ? 'text-gray-800 opacity-40' : 'text-gray-100'} text-2xl`}
  />
);

const FreightContent = (props: {
  freight?: Freight;
  hasStages: boolean;
  selected: boolean;
  stage?: Stage;
}) => {
  const { freight, hasStages, selected, stage } = props;
  const [hasCommits, setHasCommits] = useState(false);
  const [hasImages, setHasImages] = useState(false);
  useEffect(() => {
    setHasCommits((freight?.commits || []).length > 0);
    setHasImages((freight?.images || []).length > 0);
  }, [freight]);

  const bg = stage ? getStageBackground(stage) : !hasCommits && !hasImages ? '' : 'bg-zinc-800';

  return (
    <div
      className={`${bg} w-full my-1 flex-shrink h-full flex items-center justify-center flex-col ${
        selected ? 'w-3 rounded' : 'w-full rounded-md'
      }`}
    >
      {!selected && (
        <>
          {hasCommits && <FreightIcon icon={faGit} hasStages={hasStages} />}
          {hasImages && <FreightIcon icon={faDocker} hasStages={hasStages} />}
          {!hasStages && !hasCommits && !hasImages && (
            <FontAwesomeIcon icon={faBoxOpen} className='text-gray-600 text-2xl' />
          )}
        </>
      )}
    </div>
  );
};

const FreightItem = (props: {
  freight?: Freight;
  setSelected: (selected: boolean) => void;
  selected: boolean;
  stages: Stage[];
}) => {
  const { freight, selected, stages } = props;

  return (
    <div
      className={`transition-all p-2 cursor-pointer h-full mr-4 rounded-lg border-solid border-4 text-white flex flex-col items-center ${
        selected ? 'w-36 border-gray-400' : 'w-20 border-gray-600 hover:border-gray-500'
      }`}
      onClick={() => props.setSelected(!selected)}
    >
      <div className='flex w-full h-full mb-1'>
        <div
          className={`flex flex-col align-center h-full ${
            selected ? 'justify-start' : 'justify-center w-full'
          }`}
        >
          {(stages || []).map((s) => (
            <FreightContent
              freight={freight}
              hasStages={true}
              selected={selected}
              key={s?.metadata?.uid}
              stage={s}
            />
          ))}
          {(stages || []).length == 0 && (
            <FreightContent freight={freight} hasStages={false} selected={selected} />
          )}
        </div>
        {selected && (
          <div className='flex flex-col justify-center items-start w-full ml-4 font-mono text-sm'>
            {(freight?.commits || []).map((c) => (
              <div key={c.id}>{c.id.substring(0, 6)}</div>
            ))}
            {(freight?.images || []).map((i) => (
              <Tooltip
                className='flex items-center'
                key={`${i.repoUrl}:${i.tag}`}
                title={`${i.repoUrl}:${i.tag}`}
              >
                <FontAwesomeIcon icon={faDocker} className='mr-2' />
                <div>{i.tag}</div>
              </Tooltip>
            ))}
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
