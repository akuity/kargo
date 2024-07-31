import { faChevronDown, faPencil, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown } from 'antd';

import { useModal } from '@ui/features/common/modal/use-modal';

import { DeleteProjectModal } from './components/delete-project-modal';
import { EditProjectModal } from './components/edit-project-modal';

export const ProjectSettings = () => {
  const { show: showEditModal } = useModal(EditProjectModal);
  const { show: showDeleteModal } = useModal(DeleteProjectModal);

  return (
    <Dropdown
      menu={{
        items: [
          {
            key: '1',
            label: (
              <>
                <FontAwesomeIcon icon={faPencil} size='xs' className='mr-2' /> Edit
              </>
            ),
            onClick: () => showEditModal()
          },
          {
            key: '2',
            danger: true,
            label: (
              <>
                <FontAwesomeIcon icon={faTrash} size='xs' className='mr-2' /> Delete
              </>
            ),
            onClick: () => showDeleteModal()
          }
        ]
      }}
      placement='bottomRight'
      trigger={['click']}
    >
      <FontAwesomeIcon icon={faChevronDown} className='cursor-pointer text-sm ml-2 text-gray-400' />
    </Dropdown>
  );
};
