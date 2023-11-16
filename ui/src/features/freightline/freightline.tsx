import { Timestamp } from '@bufbuild/protobuf';
import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faTimeline } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Tooltip, message } from 'antd';
import { formatDistance } from 'date-fns';
import React, { useContext, useEffect, useState } from 'react';

import { ColorContext } from '@ui/context/colors';
import {
  promoteStage,
  promoteSubscribers,
  queryFreight
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Freight, GitCommit, Stage } from '@ui/gen/v1alpha1/types_pb';

export type PromotionType = 'default' | 'subscribers';

import styles from './freightline.module.less';
import { StageIndicators } from './stage-indicators';

export const Freightline = (props: {
  freight: Freight[];
  stagesPerFreight: { [key: string]: Stage[] };
  promotingStage?: Stage;
  setPromotingStage: (stage?: Stage) => void;
  promotionType?: PromotionType;
  confirmingPromotion?: string;
  setConfirmingPromotion: (confirming?: string) => void;
  project: string;
}) => {
  const {
    freight,
    stagesPerFreight,
    promotingStage,
    setPromotingStage,
    promotionType,
    confirmingPromotion,
    setConfirmingPromotion,
    project
  } = props;
  const [promotionEligible, setPromotionEligible] = useState<{ [key: string]: boolean }>({});

  const [orderedFreight, setOrderedFreight] = useState(props.freight);

  const getSeconds = (ts?: Timestamp): number => Number(ts?.seconds) || 0;

  const stageColorMap = useContext(ColorContext);

  const {
    data: availableFreightData,
    refetch,
    isLoading
  } = useQuery(queryFreight.useQuery({ project, stage: promotingStage?.metadata?.name || '' }));

  const { mutate: promoteSubscribersAction } = useMutation({
    ...promoteSubscribers.useMutation(),
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(
        `All subscribers of "${promotingStage?.metadata?.name}" stage have been promoted.`
      );
      setPromotingStage(undefined);
    }
  });

  const { mutate: promoteAction } = useMutation({
    ...promoteStage.useMutation(),
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(`Stage "${promotingStage?.metadata?.name}" has been promoted.`);
      setPromotingStage(undefined);
    }
  });

  useEffect(() => {
    const ordered = (freight || []).sort(
      (a, b) =>
        getSeconds(b.metadata?.creationTimestamp) - getSeconds(a.metadata?.creationTimestamp)
    );
    setOrderedFreight(ordered);
  }, [freight]);

  useEffect(() => {
    refetch();
  }, [promotingStage, freight, promotionType]);

  useEffect(() => {
    if (!isLoading) {
      const availableFreight =
        promotionType === 'default'
          ? availableFreightData?.groups['']?.freight || []
          : promotingStage?.status?.history || [];
      const pe: { [key: string]: boolean } = {};
      ((availableFreight as Freight[]) || []).map((f: Freight) => {
        const name = promotionType === 'default' ? f?.metadata?.name : f?.id;
        pe[name || ''] = true;
      });
      setPromotionEligible(pe);
    }
  }, [availableFreightData]);

  return (
    <div
      className='w-full py-4 px-1 h-56 flex flex-col overflow-hidden'
      style={{ backgroundColor: '#222' }}
    >
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
              onClick={() => setPromotingStage(undefined)}
            >
              CANCEL
            </div>
          </div>
        )}
      </div>
      <div className='flex h-full w-full items-center'>
        <div
          className='text-gray-500 text-sm font-semibold mb-2 w-min h-min'
          style={{ transform: 'rotate(-0.25turn)' }}
        >
          NEW
        </div>
        <div className='flex items-center h-full overflow-x-auto'>
          {(orderedFreight || []).map((f, i) => {
            const id = f?.metadata?.name || `${i}`;
            return (
              <FreightItem
                freight={f || undefined}
                key={id}
                stages={stagesPerFreight[id] || []}
                promotable={promotionEligible[id]}
                promoting={promotingStage}
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
        </div>
        <div className='rotate-90 text-gray-500 text-sm font-semibold ml-auto'>OLD</div>
      </div>
    </div>
  );
};

const CommitInfo = ({ commit }: { commit: GitCommit }) => (
  <div className='grid grid-cols-2'>
    <div>Repo:</div>
    <div>
      <a href={commit.repoUrl}>{commit.repoUrl}</a>
    </div>
    <div>Branch:</div>
    <div>{commit.branch}</div>
    {commit.author && (
      <>
        <div>Author:</div>
        <div>{commit.author}</div>
      </>
    )}
    {commit.message && (
      <>
        <div>Message:</div>
        <div>{commit.message}</div>
      </>
    )}
  </div>
);

const FreightContents = (props: {
  freight?: Freight;
  highlighted: boolean;
  promoting: boolean;
}) => {
  const { freight, highlighted, promoting } = props;

  const FreightContentItem = (
    props: {
      icon: IconDefinition;
      overlay?: React.ReactNode;
      title?: string;
    } & React.PropsWithChildren
  ) => (
    <Tooltip
      className={`${styles.freightContentItem} ${promoting && highlighted ? 'bg-transparent' : ''}`}
      overlay={props.overlay}
      title={props.title}
    >
      <FontAwesomeIcon icon={props.icon} className={`px-1 text-lg mb-2`} />
      {props.children}
    </Tooltip>
  );

  return (
    <div
      className={`hover:text-white flex flex-col justify-center items-center font-mono text-xs flex-shrink min-w-min w-full ${
        highlighted ? 'text-white' : 'text-gray-500'
      }`}
    >
      {(freight?.commits || []).map((c) => (
        <FreightContentItem key={c.id} overlay={<CommitInfo commit={c} />} icon={faGit}>
          <a
            href={`${c.repoUrl.replace('.git', '')}/commit/${c.id}`}
            target='_blank'
            className={`${highlighted ? 'text-blue-200' : 'text-gray-500'} hover:text-blue-300`}
          >
            {c.id.substring(0, 6)}
          </a>
        </FreightContentItem>
      ))}
      {(freight?.images || []).map((i) => (
        <FreightContentItem
          key={`${i.repoUrl}:${i.tag}`}
          title={`${i.repoUrl}:${i.tag}`}
          icon={faDocker}
        >
          <div>{i.tag}</div>
        </FreightContentItem>
      ))}
    </div>
  );
};

const FreightItem = (props: {
  freight?: Freight;
  stages: Stage[];
  promotable?: boolean;
  promoting?: Stage;
  promotionType?: PromotionType;
  confirmingPromotion: boolean;
  setConfirmingPromotion: (confirming: boolean) => void;
  onConfirm: () => void;
}) => {
  const {
    freight,
    stages,
    promotable,
    promoting,
    promotionType,
    confirmingPromotion,
    setConfirmingPromotion,
    onConfirm
  } = props;
  const [conditionalStyles, setConditionalStyles] = useState('');

  useEffect(() => {
    if (promoting) {
      setConditionalStyles(promotable ? styles.promotable : styles.unpromotable);
      if (confirmingPromotion) {
        setConditionalStyles(`${conditionalStyles} ${styles.confirming}`);
      }
    } else {
      setConditionalStyles('');
    }
  }, [promoting, promotable, confirmingPromotion]);

  return (
    <div
      className={`${styles.freightItem} ${conditionalStyles} ${
        (stages || []).length > 0 && !confirmingPromotion ? 'w-32' : ''
      } ${!promoting && (stages || []).length > 0 ? 'border-gray-500' : ''}`}
      onClick={() => {
        if (promoting) {
          if (promotable) {
            setConfirmingPromotion(!confirmingPromotion);
          } else {
            return;
          }
        }
      }}
    >
      <div className='flex w-full h-full mb-1 items-center justify-center'>
        <StageIndicators stages={stages} />
        <FreightContents
          highlighted={
            ((stages || []).length > 0 && !promoting) || (promoting && promotable) || false
          }
          promoting={!!promoting}
          freight={freight}
        />
        {promoting && confirmingPromotion && (
          <div>
            <div className='text-xs px-2 mb-3'>
              Are you sure you want to promote{' '}
              {promotionType === 'subscribers' ? (
                ' subscribers'
              ) : (
                <b>{promoting?.metadata?.name}</b>
              )}
              ?
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
        )}
      </div>
      <div className='mt-auto w-full'>
        <div
          className={`w-full text-center font-mono text-xs ${
            confirmingPromotion ? 'text-white' : 'text-gray-400'
          }`}
        >
          <Tooltip
            title={
              freight?.metadata?.creationTimestamp &&
              formatDistance(freight?.metadata?.creationTimestamp?.toDate(), new Date(), {
                addSuffix: true
              })
            }
            placement='bottom'
          >
            {freight?.metadata?.name?.substring(0, 7)}
          </Tooltip>
        </div>
      </div>
    </div>
  );
};
