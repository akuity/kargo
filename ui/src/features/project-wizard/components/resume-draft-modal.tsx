import { Alert, Button, Modal } from 'antd';

import { useGetProject } from '@ui/gen/api/v2/core/core';

type ResumeDraftModalProps = {
  open: boolean;
  // Whether the saved draft contains any credentials, so the resume prompt can
  // warn that their secrets were not saved and must be re-entered.
  hasCredentials?: boolean;
  // Looked up so the prompt can say when the name is taken - most often by
  // what an interrupted run left behind.
  projectName: string;
  onResume: () => void;
  onStartFresh: () => void;
};

// Shown on wizard entry when a saved draft with real data exists, so the user
// chooses between resuming it and starting a new project.
export const ResumeDraftModal = ({
  open,
  hasCredentials,
  projectName,
  onResume,
  onStartFresh
}: ResumeDraftModalProps) => {
  // Not found is the expected answer here, so skip the global 404 notification.
  // Any other failure stays unsuccessful and shows no warning.
  const { isSuccess: projectExists } = useGetProject(projectName, {
    query: { enabled: open && !!projectName, meta: { silent404: true } }
  });

  return (
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
      You have a saved project draft from a previous session. Resume where you left off, or discard
      it and start a new project?
      {projectExists && (
        <Alert
          className='mt-4'
          type='warning'
          showIcon
          message={
            `A project named ${projectName} already exists, so creating this draft will fail. ` +
            `If an earlier attempt was interrupted it may be incomplete - delete it, or give ` +
            `this draft a new name.`
          }
        />
      )}
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
};
