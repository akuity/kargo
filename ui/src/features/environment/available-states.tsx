import { transport } from '@config/transport';
import { ButtonIcon } from '@features/ui';
import { faDocker } from '@fortawesome/free-brands-svg-icons';
import { faArrowTurnUp, faCodeCommit, faExternalLinkAlt } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { promoteEnvironment } from '@gen/service/v1alpha1/service-KargoService_connectquery';
import { Environment } from '@gen/v1alpha1/generated_pb';
import { useMutation } from '@tanstack/react-query';
import { Button, List, Tooltip, Typography } from 'antd';
import { format, formatRelative } from 'date-fns';
import React from 'react';

export const AvailableStates = (props: { environment: Environment }) => {
  const [promotingStateId, setPromotingStateId] = React.useState<string | null>(null);
  const { environment } = props;
  const { mutate, isLoading: isLoadingPromote } = useMutation(
    promoteEnvironment.useMutation({ transport })
  );

  const promote = (id: string) => {
    setPromotingStateId(id);
    mutate({
      name: environment.metadata?.name,
      project: environment.metadata?.namespace,
      state: id
    });
  };

  return (
    <div>
      <Typography.Title level={3}>Available States</Typography.Title>
      <List
        itemLayout='horizontal'
        dataSource={environment?.status?.availableStates || []}
        renderItem={(state) => (
          <List.Item
            actions={[
              <Button
                key='promote'
                type='primary'
                icon={<ButtonIcon icon={faArrowTurnUp} size='1x' />}
                onClick={() => state.id && promote(state.id)}
                disabled={environment.status?.currentState?.id === state.id}
                loading={isLoadingPromote && promotingStateId === state.id}
              >
                Promote
              </Button>
            ]}
          >
            {state.commits.map((commit) => (
              <List.Item.Meta
                key={commit.id}
                avatar={<FontAwesomeIcon icon={faCodeCommit} />}
                title={
                  <a
                    href={`${commit.repoURL?.replace('.git', '')}/commit/${commit.id}`}
                    target='_blank'
                  >
                    {commit?.id?.slice(0, 7)}
                    <FontAwesomeIcon icon={faExternalLinkAlt} style={{ marginLeft: '5px' }} />
                  </a>
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
              <List.Item.Meta avatar={<FontAwesomeIcon icon={faDocker} />} title='Image' />
            )}
          </List.Item>
        )}
      />
    </div>
  );
};
