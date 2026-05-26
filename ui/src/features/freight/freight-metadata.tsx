import { faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Descriptions, Popover } from 'antd';

import { Freight } from '@ui/gen/api/v2/models';

type FreightMetadataProps = {
  freight: Freight;
  className?: string;
};

export const FreightMetadata = (props: FreightMetadataProps) => {
  if (
    !props?.freight?.status?.metadata ||
    Object.keys(props.freight.status?.metadata || {}).length === 0
  ) {
    return null;
  }

  return (
    <Descriptions
      className={props.className}
      title={
        <>
          Metadata{' '}
          <Popover
            content={
              <>
                Freight metadata is set by{' '}
                <a
                  href='https://docs.kargo.io/user-guide/reference-docs/promotion-steps/set-metadata'
                  target='_blank'
                >
                  set-metadata
                </a>{' '}
                step
              </>
            }
          >
            <FontAwesomeIcon icon={faQuestionCircle} />
          </Popover>
        </>
      }
      items={Object.entries(props.freight.status?.metadata || {}).map(([key, value]) => {
        // TODO(Marvin9): verify
        console.log({ value });
        return {
          key,
          label: key,
          children: ''
        };
      })}
      bordered
    />
  );
};
