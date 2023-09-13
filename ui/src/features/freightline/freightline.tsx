import { Timestamp } from '@bufbuild/protobuf';
import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faThumbTack, faTimeline } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMutation } from '@tanstack/react-query';
import { Tooltip, message } from 'antd';
import React, { useEffect, useState } from 'react';

import {
  promoteStage,
  promoteSubscribers
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Stage } from '@ui/gen/v1alpha1/types_pb';

export type PromotionType = 'default' | 'subscribers';

import styles from './freightline.module.less';

export const Freightline = (props: {
  freight: Freight[];
  stagesPerFreight: { [key: string]: Stage[] };
  stageColorMap: { [key: string]: string };
  promotingStage?: Stage;
  setPromotingStage: (stage?: Stage) => void;
  promotionType?: PromotionType;
}) => {
  const {
    freight,
    stagesPerFreight,
    stageColorMap,
    promotingStage,
    setPromotingStage,
    promotionType
  } = props;
  const [selected, setSelected] = React.useState<string | null>(null);
  const [promotionEligible, setPromotionEligible] = useState<{ [key: string]: boolean }>({});
  const [confirmingPromotion, setConfirmingPromotion] = useState<string | undefined>();

  const [orderedFreight, setOrderedFreight] = useState(props.freight);

  const getSeconds = (ts?: Timestamp): number => Number(ts?.seconds) || 0;

  const { mutate: promoteSubscribersAction } = useMutation({
    ...promoteSubscribers.useMutation(),
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(
        `All subscribers of "${promotingStage?.metadata?.name}" stage have been promoted.`
      );
    }
  });

  const { mutate: promoteAction } = useMutation({
    ...promoteStage.useMutation(),
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(`Stage "${promotingStage?.metadata?.name}" has been promoted.`);
    }
  });

  useEffect(() => {
    const ordered = (freight || []).sort(
      (a, b) => getSeconds(b.firstSeen) - getSeconds(a.firstSeen)
    );
    setOrderedFreight(ordered);
  }, [freight]);

  useEffect(() => {
    const availableFreight =
      promotionType === 'default'
        ? promotingStage?.status?.availableFreight
        : (promotingStage?.status?.history || []).filter((f) => f.qualified);
    const pe: { [key: string]: boolean } = {};
    (availableFreight || []).map((f) => {
      pe[f.id || ''] = true;
    });
    setPromotionEligible(pe);
  }, [promotingStage, freight, promotionType]);

  return (
    <div className='bg-zinc-900 w-full py-4 px-1 h-56 flex flex-col'>
      <div className='text-gray-300 text-sm ml-12 mb-3'>
        {promotingStage === undefined ? (
          <div className='font-semibold flex items-center'>
            <FontAwesomeIcon icon={faTimeline} className='mr-2' />
            FREIGHTLINE
          </div>
        ) : (
          <div className='flex items-center'>
            PROMOTING {promotionType === 'subscribers' ? 'SUBSCRIBERS' : ''} /
            <div className='font-semibold flex items-center ml-1'>
              STAGE{' '}
              <div
                className='px-2 rounded text-white ml-2'
                style={{ backgroundColor: stageColorMap[promotingStage?.metadata?.uid || ''] }}
              >
                {' '}
                {promotingStage?.metadata?.name?.toUpperCase()}
              </div>
            </div>
            <div
              className='ml-auto mr-4 cursor-pointer px-2 text-white bg-zinc-700 rounded hover:bg-zinc-600 font-semibold'
              onClick={() => promotingStage && setPromotingStage(undefined)}
            >
              CANCEL
            </div>
          </div>
        )}
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
              stages={stagesPerFreight[id] || []}
              stageColorMap={stageColorMap}
              promotable={promotionEligible[id]}
              promoting={promotingStage !== undefined}
              promotionType={promotionType || 'default'}
              confirmingPromotion={confirmingPromotion === f?.id}
              setConfirmingPromotion={(c: boolean) => {
                if (c) {
                  setConfirmingPromotion(f?.id);
                } else {
                  setConfirmingPromotion(undefined);
                }
              }}
              onConfirm={() => {
                const currentData = {
                  project: promotingStage?.metadata?.namespace,
                  freight: f?.id
                };
                if (promotionType === 'default') {
                  promoteAction({
                    name: promotingStage?.metadata?.name,
                    ...currentData
                  });
                } else {
                  promoteSubscribersAction({
                    stage: promotingStage?.metadata?.name,
                    ...currentData
                  });
                }
              }}
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
    <div
      className={`flex flex-col align-center h-full justify-center w-full flex-grow mr-2`}
      style={{ maxWidth: '25px' }}
    >
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
      className={`flex flex-col justify-center items-center font-mono text-sm flex-shrink min-w-min w-full ${
        selected || pinned ? 'text-white' : 'text-gray-300'
      }`}
    >
      {(freight?.commits || []).map((c) => (
        <Tooltip
          key={c.id}
          className='flex items-center my-2'
          overlay={
            <div className='grid grid-cols-2'>
              <div>Repo:</div>
              <div>
                <a href={c.repoUrl}>{c.repoUrl}</a>
              </div>
              <div>Branch:</div>
              <div>{c.branch}</div>
              {c.author && (
                <>
                  <div>Author:</div>
                  <div>{c.author}</div>
                </>
              )}
              {c.message && (
                <>
                  <div>Message:</div>
                  <div>{c.message}</div>
                </>
              )}
            </div>
          }
        >
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
  promotable?: boolean;
  promoting?: boolean;
  promotionType?: PromotionType;
  confirmingPromotion: boolean;
  setConfirmingPromotion: (confirming: boolean) => void;
  onConfirm: () => void;
}) => {
  const {
    freight,
    selected,
    setSelected,
    stages,
    promotable,
    promoting,
    promotionType,
    confirmingPromotion,
    setConfirmingPromotion,
    onConfirm
  } = props;
  const [pinned, setPinned] = useState(false);
  const [conditionalStyles, setConditionalStyles] = useState('');

  useEffect(() => {
    if (promoting) {
      setConditionalStyles(promotable ? styles.promotable : styles.unpromotable);
      if (confirmingPromotion) {
        setConditionalStyles(`${conditionalStyles} ${styles.confirming}`);
      }
    } else {
      setConditionalStyles(selected || pinned ? styles.selected : '');
    }
  }, [promoting, promotable, selected, pinned, confirmingPromotion]);

  return (
    <div
      className={`${styles.freightItem} ${conditionalStyles}`}
      onClick={() => {
        if (promoting) {
          if (promotable) {
            setConfirmingPromotion(!confirmingPromotion);
          } else {
            return;
          }
        }
        setSelected(!selected);
      }}
    >
      <div className='flex w-full h-full mb-1 items-center justify-center'>
        {!promoting && <StageIndicators stages={stages} stageColorMap={props.stageColorMap} />}
        {promoting && confirmingPromotion ? (
          <div>
            <div className='text-sm px-2 mb-3'>
              Are you sure you want to promote
              {promotionType === 'subscribers' ? ' subscribers' : ''}?
            </div>
            <div className='flex items-center w-full justify-center'>
              <div
                className={`${styles.confirmButton} bg-sky-500 mr-4 hover:bg-sky-600`}
                onClick={onConfirm}
              >
                YES
              </div>
              <div className={`${styles.confirmButton} bg-sky-800 hover:bg-sky-900`}>NO</div>
            </div>
          </div>
        ) : (
          <FreightContents
            freight={freight}
            pinned={pinned && !promoting}
            setPinned={setPinned}
            selected={selected && !promoting}
          />
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
