import { Button, Modal } from 'antd';
import yaml from 'yaml';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import { AnalysisTemplate } from '@ui/gen/api/v1alpha1/generated_pb';

type Props = ModalProps & {
  template: AnalysisTemplate;
};

export const PreviewAnalysisTemplateModal = ({ visible, hide, template }: Props) => (
  <Modal
    open={visible}
    onCancel={hide}
    title={template.metadata?.name}
    footer={
      <Button type='primary' onClick={hide}>
        Close
      </Button>
    }
    width={700}
  >
    <YamlEditor value={yaml.stringify(template.toJson())} height='500px' disabled />
  </Modal>
);
