import { Table } from 'antd';
import Radio from 'antd/es/radio/radio';

import { ArtifactReference } from '@ui/gen/api/v1alpha1/generated_pb';

type GenericArtifactTableProps = {
  show?: boolean;
  references: ArtifactReference[];
  selected: ArtifactReference | undefined;
  select: (reference?: ArtifactReference) => void;
};

export const GenericArtifactTable = (props: GenericArtifactTableProps) => {
  if (!props.show) {
    return null;
  }

  return (
    <Table
      dataSource={props.references}
      columns={[
        {
          width: '50px',
          render: (_, record) => (
            <Radio
              checked={props.selected?.version === record.version}
              onClick={() =>
                props.select(props.selected?.version === record.version ? undefined : record)
              }
            />
          )
        },
        { title: 'Version', dataIndex: 'version' }
      ]}
    />
  );
};
