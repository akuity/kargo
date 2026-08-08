import { Button, Modal } from 'antd';

type ResumeDraftModalProps = {
  open: boolean;
  onResume: () => void;
  onStartFresh: () => void;
};

// Shown on wizard entry when a saved draft with real data exists, so the user
// chooses between resuming it and starting a new project.
export const ResumeDraftModal = ({ open, onResume, onStartFresh }: ResumeDraftModalProps) => (
  <Modal
    open={open}
    title='Resume your draft?'
    closable={false}
    maskClosable={false}
    footer={[
      <Button key='fresh' danger onClick={onStartFresh}>
        Start new project
      </Button>,
      <Button key='resume' type='primary' onClick={onResume}>
        Resume
      </Button>
    ]}
  >
    You have a saved project draft from a previous session. Resume where you left off, or discard it
    and start a new project?
  </Modal>
);
