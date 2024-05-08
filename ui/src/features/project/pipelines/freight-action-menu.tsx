import {
  faCircleCheck,
  faClipboard,
  faCopy,
  faEllipsisV,
  faPencil
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown, message } from 'antd';

import { useModal } from '@ui/features/common/modal/use-modal';
import { getAlias } from '@ui/features/common/utils';
import { Freight } from '@ui/gen/v1alpha1/generated_pb';

import { UpdateFreightAliasModal } from './update-freight-alias-modal';

export const FreightActionMenu = ({
  freight,
  approveAction,
  refetchFreight
}: {
  freight: Freight;
  approveAction: () => void;
  refetchFreight: () => void;
}) => {
  const { show } = useModal();

  return (
    <Dropdown
      className='absolute top-2 right-2 pl-2'
      trigger={['click']}
      dropdownRender={(menu) => {
        return <div onClick={(e) => e.stopPropagation()}>{menu}</div>;
      }}
      menu={{
        items: [
          {
            key: '1',
            label: (
              <>
                <FontAwesomeIcon icon={faCircleCheck} className='mr-2' />
                Manually Approve
              </>
            ),
            onClick: approveAction
          },
          {
            key: '2',
            label: (
              <>
                <FontAwesomeIcon icon={faClipboard} className='mr-2' /> Copy ID
              </>
            ),
            onClick: () => {
              navigator.clipboard.writeText(freight.metadata?.uid || '');
              message.success('Copied Freight ID to clipboard');
            }
          },
          getAlias(freight)
            ? {
                key: '3',
                label: (
                  <>
                    <FontAwesomeIcon icon={faCopy} className='mr-2' /> Copy Alias
                  </>
                ),
                onClick: () => {
                  navigator.clipboard.writeText(getAlias(freight) || '');
                  message.success('Copied Freight Alias to clipboard');
                }
              }
            : null,
          {
            key: '4',
            label: (
              <>
                <FontAwesomeIcon icon={faPencil} className='mr-2' /> Change Alias
              </>
            ),
            onClick: async () => {
              show((p) => (
                <UpdateFreightAliasModal
                  {...p}
                  freight={freight || undefined}
                  project={freight?.metadata?.namespace || ''}
                  onSubmit={() => {
                    refetchFreight();
                    p.hide();
                  }}
                />
              ));
            }
          }
        ]
      }}
    >
      <FontAwesomeIcon
        onClick={(e) => e.stopPropagation()}
        icon={faEllipsisV}
        className='cursor-pointer text-gray-500 hover:text-white'
      />
    </Dropdown>
  );
};
