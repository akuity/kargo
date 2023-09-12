import { faDocker } from '@fortawesome/free-brands-svg-icons';
import { faArrowTurnUp, faCodeCommit, faExternalLinkAlt } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMutation } from '@tanstack/react-query';
import { Button, Descriptions, List, Popconfirm, Tooltip, Typography, message } from 'antd';
import { format, formatRelative } from 'date-fns';
import React from 'react';

import { ButtonIcon } from '@ui/features/common';
import { promoteStage } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Stage } from '@ui/gen/v1alpha1/types_pb';

export const AvailableFreight = (props: { stage: Stage; onSuccess?: () => void }) => {
  const [promotingFreightId, setPromotingFreightId] = React.useState<string | null>(null);
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
    setPromotingFreightId(id);
    mutate({
      name: stage.metadata?.name,
      project: stage.metadata?.namespace,
      freight: id
    });
  };

  return (
    <List
      itemLayout='horizontal'
      dataSource={stage?.status?.availableFreight || []}
      renderItem={(freight) => (
        <List.Item
          actions={[
            <Popconfirm
              key='promote'
              title='Are you sure to promote to this freight?'
              onConfirm={() => freight.id && promote(freight.id)}
              okText='Confirm'
              placement='left'
              icon=''
              disabled={stage.status?.currentFreight?.id === freight.id}
            >
              <Button
                type='primary'
                icon={<ButtonIcon icon={faArrowTurnUp} size='1x' />}
                disabled={stage.status?.currentFreight?.id === freight.id}
                loading={isLoadingPromote && promotingFreightId === freight.id}
              >
                Promote
              </Button>
            </Popconfirm>
          ]}
        >
          {freight.commits.map((commit) => (
            <List.Item.Meta
              key={commit.id}
              avatar={<FontAwesomeIcon icon={faCodeCommit} />}
              title={
                <Typography.Link
                  href={`${commit.repoUrl?.replace('.git', '')}/commit/${commit.id}`}
                  target='_blank'
                >
                  {commit?.id?.slice(0, 7)}
                  <FontAwesomeIcon icon={faExternalLinkAlt} style={{ marginLeft: '5px' }} />
                </Typography.Link>
              }
              description={
                <>
                  <Tooltip
                    placement='topLeft'
                    title={
                      freight.firstSeen &&
                      format(freight.firstSeen.toDate(), "HH:mm:ss 'on' MMM do yyyy")
                    }
                  >
                    <div>
                      {freight.firstSeen && formatRelative(freight.firstSeen?.toDate(), new Date())}
                    </div>
                  </Tooltip>
                  <div>
                    {commit.author && `${commit.author} : `}
                    {commit.message}
                  </div>
                </>
              }
            />
          ))}
          {freight.commits.length === 0 && (
            <List.Item.Meta
              avatar={<FontAwesomeIcon icon={faDocker} />}
              title='Image'
              description={
                <Descriptions size='small' column={1}>
                  <Descriptions.Item label='Repo URL'>
                    {freight.images[0]?.repoUrl}
                  </Descriptions.Item>
                  <Descriptions.Item label='Tag'>{freight.images[0]?.tag}</Descriptions.Item>
                </Descriptions>
              }
            />
          )}
        </List.Item>
      )}
    />
  );
};
