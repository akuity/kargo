import {
  faCircleCheck,
  faCircleExclamation,
  faCircleNotch,
  faCircleQuestion,
  faHourglassStart
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { theme } from 'antd';

export const VerificationIcon = ({ phase }: { phase: string }) => {
  switch (phase) {
    case 'Successful':
      return (
        <FontAwesomeIcon color={theme.defaultSeed.colorSuccess} icon={faCircleCheck} size='lg' />
      );
    case 'Failed':
    case 'Error':
    case 'Aborted':
      return (
        <FontAwesomeIcon
          color={theme.defaultSeed.colorError}
          icon={faCircleExclamation}
          size='lg'
        />
      );
    case 'Running':
      return <FontAwesomeIcon icon={faCircleNotch} spin size='lg' />;
    case 'Pending':
      return <FontAwesomeIcon color='#aaa' icon={faHourglassStart} size='lg' />;
    default:
      return <FontAwesomeIcon color='#aaa' icon={faCircleQuestion} size='lg' />;
  }
};
