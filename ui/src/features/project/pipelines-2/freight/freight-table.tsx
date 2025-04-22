import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Table } from 'antd';
import classNames from 'classnames';

import { ArtifactMetadata } from '@ui/features/freight/artifact-metadata';
import { flattenFreightOrigin } from '@ui/features/freight/flatten-freight-origin-utils';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

type FreightTableProps = {
  freight: Freight;
  className?: string;
};

export const FreightTable = (props: FreightTableProps) => {
  return (
    <Table
      className={classNames(props.className)}
      pagination={{ pageSize: 5 }}
      dataSource={flattenFreightOrigin(props.freight)}
      columns={[
        {
          title: 'Source',
          width: '5%',
          render: (_, { type }) => {
            let icon: IconProp = faGitAlt;

            switch (type) {
              case 'helm':
                icon = faAnchor;
                break;
              case 'image':
                icon = faDocker;
                break;
            }

            return <FontAwesomeIcon icon={icon} />;
          }
        },
        {
          title: 'Repo',
          dataIndex: 'repoURL',
          width: '30%'
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
  );
};
