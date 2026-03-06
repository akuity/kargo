import { faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Descriptions, Popover } from 'antd';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

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
        const decodedValue = decodeRawData({ result: { case: 'raw', value: value.raw } });
        return {
          key,
          label: key,
          children: decodedValue
        };
      })}
      bordered
    />
  );
};
