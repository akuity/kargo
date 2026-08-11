import { Alert, Button, Modal } from 'antd';

type ResumeDraftModalProps = {
  open: boolean;
  // Whether the saved draft contains any credentials, so the resume prompt can
  // warn that their secrets were not saved and must be re-entered.
  hasCredentials?: boolean;
  onResume: () => void;
  onStartFresh: () => void;
};

// Shown on wizard entry when a saved draft with real data exists, so the user
// chooses between resuming it and starting a new project.
export const ResumeDraftModal = ({
  open,
  hasCredentials,
  onResume,
  onStartFresh
}: ResumeDraftModalProps) => (
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
    {hasCredentials && (
      <Alert
        className='mt-4'
        type='info'
        showIcon
        message="For your security, saved drafts don't include credential secrets - you'll need to re-enter any passwords or keys before creating the project."
      />
    )}
  </Modal>
);
