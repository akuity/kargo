import { Modal } from 'antd';

import { ManifestPreview } from '@ui/features/common/manifest-preview';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { Target } from '@ui/gen/api/v2/models';

/**
 * TargetYamlModal shows a Target's manifest. The modal stack keeps the Target
 * it was opened with, so this is a snapshot rather than a live view -- which
 * is what reading a manifest wants anyway: the text does not move under the
 * cursor while it is being read or copied.
 */
export const TargetYamlModal = ({ target, ...props }: ModalComponentProps & { target: Target }) => (
  <Modal
    open={props.visible}
    onCancel={props.hide}
    footer={null}
    width='min(1000px, 90vw)'
    title={`Target ${target.metadata?.name || ''}`}
  >
    <ManifestPreview object={target} height='60vh' />
  </Modal>
);
