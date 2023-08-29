import { faDocker } from '@fortawesome/free-brands-svg-icons';
import { faArrowTurnUp, faCodeCommit, faExternalLinkAlt } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMutation } from '@tanstack/react-query';
import { Button, Descriptions, List, Popconfirm, Tooltip, Typography, message } from 'antd';
import { format, formatRelative } from 'date-fns';
import React from 'react';

import { ButtonIcon } from '@ui/features/common';
import { promoteStage } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';

export const AvailableStates = (props: { stage: Stage; onSuccess?: () => void }) => {
  const [promotingStateId, setPromotingStateId] = React.useState<string | null>(null);
  const { stage, onSuccess } = props;
  const { mutate, isLoading: isLoadingPromote } = useMutation({
    ...promoteStage.useMutation(),
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(`The "${stage.metadata?.name}" stage has been promoted.`);
      onSuccess?.();
    }
  });

  const promote = (id: string) => {
    setPromotingStateId(id);
    mutate({
      name: stage.metadata?.name,
      project: stage.metadata?.namespace,
      state: id
    });
  };

  return (
    <div>
      <Typography.Title level={3}>Available States</Typography.Title>
      <List
        itemLayout='horizontal'
        dataSource={stage?.status?.availableStates || []}
        renderItem={(state) => (
          <List.Item
            actions={[
              <Popconfirm
                key='promote'
                title='Are you sure to promote this state?'
                onConfirm={() => state.id && promote(state.id)}
                okText='Confirm'
                placement='left'
                icon=''
                disabled={stage.status?.currentState?.id === state.id}
              >
                <Button
                  type='primary'
                  icon={<ButtonIcon icon={faArrowTurnUp} size='1x' />}
                  disabled={stage.status?.currentState?.id === state.id}
                  loading={isLoadingPromote && promotingStateId === state.id}
                >
                  Promote
                </Button>
              </Popconfirm>
            ]}
          >
            {state.commits.map((commit) => (
              <List.Item.Meta
                key={commit.id}
                avatar={<FontAwesomeIcon icon={faCodeCommit} />}
                title={
                  <Typography.Link
                    href={`${commit.repoURL?.replace('.git', '')}/commit/${commit.id}`}
                    target='_blank'
                  >
                    {commit?.id?.slice(0, 7)}
                    <FontAwesomeIcon icon={faExternalLinkAlt} style={{ marginLeft: '5px' }} />
                  </Typography.Link>
                }
                description={
                  <Tooltip
                    title={
                      state.firstSeen &&
                      format(state.firstSeen.toDate(), "HH:mm:ss 'on' MMM do yyyy")
                    }
                  >
                    {state.firstSeen && formatRelative(state.firstSeen?.toDate(), new Date())}
                  </Tooltip>
                }
              />
            ))}
            {state.commits.length === 0 && (
              <List.Item.Meta
                avatar={<FontAwesomeIcon icon={faDocker} />}
                title='Image'
                description={
                  <Descriptions size='small' column={1}>
                    <Descriptions.Item label='Repo URL'>
                      {state.images[0]?.repoURL}
                    </Descriptions.Item>
                    <Descriptions.Item label='Tag'>{state.images[0]?.tag}</Descriptions.Item>
                  </Descriptions>
                }
              />
            )}
          </List.Item>
        )}
      />
    </div>
  );
};
