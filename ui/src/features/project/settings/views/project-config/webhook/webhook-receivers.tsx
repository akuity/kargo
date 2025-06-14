import { faBitbucket, faGithub, faGitlab, faRedhat } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { ReactNode } from 'react';

type WebhookReceiverSecretT = {
  dataKey: string;
  description?: ReactNode;
};

export type WebhookReceiverT = {
  key: string;
  label: string;
  icon?: IconDefinition;
  secrets: WebhookReceiverSecretT[];
};

// information manually ported from api/v1/project_config_types.go

const bitbucket: WebhookReceiverT = {
  key: 'bitbucket',
  label: 'Bitbucket',
  icon: faBitbucket,
  secrets: [
    {
      dataKey: 'secret',
      description: (
        <>
          The Secret's data map is expected to contain a `secret` key whose value is the shared
          secret used to authenticate the webhook requests sent by Bitbucket. For more information
          please refer to the{' '}
          <a
            href='https://support.atlassian.com/bitbucket-cloud/docs/manage-webhooks/'
            target='_blank'
          >
            Bitbucket documentation
          </a>
        </>
      )
    }
  ]
};

const github: WebhookReceiverT = {
  key: 'github',
  label: 'Github',
  icon: faGithub,
  secrets: [
    {
      dataKey: 'secret',
      description: (
        <>
          The Secret's data map is expected to contain a `secret` key whose value is the shared
          secret used to authenticate the webhook requests sent by GitHub. For more information
          please refer to{' '}
          <a
            href='https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries'
            target='_blank'
          >
            GitHub documentation
          </a>
        </>
      )
    }
  ]
};

const gitlab: WebhookReceiverT = {
  key: 'gitlab',
  label: 'Gitlab',
  icon: faGitlab,
  secrets: [
    {
      dataKey: 'secret-token',
      description: (
        <>
          The secret is expected to contain a `secret-token` key containing the shared secret
          specified when registering the webhook in GitLab. For more information about this token,
          please refer to the{' '}
          <a href='https://docs.gitlab.com/user/project/integrations/webhooks/' target='_blank'>
            GitLab documentation
          </a>
        </>
      )
    }
  ]
};

const quay: WebhookReceiverT = {
  key: 'quay',
  label: 'Quay',
  icon: faRedhat,
  secrets: [
    {
      dataKey: 'secret',
      description: (
        <>
          The Secret's data map is expected to contain a `secret` key whose value does NOT need to
          be shared directly with Quay when registering a webhook. It is used only by Kargo to
          create a complex, hard-to-guess URL, which implicitly serves as a shared secret. For more
          information about // Quay webhooks, please refer to the{' '}
          <a href='https://docs.quay.io/guides/notifications.html' target='_blank'>
            Quay documentation
          </a>
        </>
      )
    }
  ]
};

export const webhookReceivers: WebhookReceiverT[] = [bitbucket, github, gitlab, quay];
