import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faArrowDownShortWide } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Table } from 'antd';
import classNames from 'classnames';
import { useMemo } from 'react';

import { ArtifactMetadata } from '@ui/features/freight/artifact-metadata';
import { flattenFreightOrigin } from '@ui/features/freight/flatten-freight-origin-utils';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

type FreightTableProps = {
  freight: Freight;
  className?: string;
};

export const FreightTable = (props: FreightTableProps) => {
  const freightSource = useMemo(() => flattenFreightOrigin(props.freight), [props.freight]);

  return (
    <>
      {freightSource?.length > 0 && (
        <Table
          className={classNames(props.className)}
          pagination={{ pageSize: 5, hideOnSinglePage: true }}
          dataSource={freightSource}
          columns={[
            {
              title: 'Type',
              width: '10%',
              render: (_, record) => {
                if (record.type === 'other') {
                  return record.artifactType || '-';
                }

                let icon: IconProp = faArrowDownShortWide;

                switch (record.type) {
                  case 'helm':
                    icon = faAnchor;
                    break;
                  case 'image':
                    icon = faDocker;
                    break;
                  case 'git':
                    icon = faGitAlt;
                }

                return <FontAwesomeIcon icon={icon} />;
              }
            },
            {
              title: 'Repo / Name',
              width: '30%',
              render: (_, record) => {
                if (record.type === 'other') {
                  return record.subscriptionName || '-';
                }

                return record.repoURL;
              }
            },
            {
              title: 'Version',
              render: (_, record) => {
                switch (record.type) {
                  case 'git':
                    return record.id;
                  case 'helm':
                    return record.version;
                  case 'image':
                    return record.tag;
                  default:
                    return record.version || '-';
                }
              }
            },
            {
              title: 'Metadata',
              width: '600px',
              render: (_, record) => {
                return <ArtifactMetadata {...record} />;
              }
            }
          ]}
        />
      )}
    </>
  );
};
