import { faBoxes, faUserGear } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

export const IconSetByKargoTerminology = {
  subscription: () => (
    <div className='flex items-center gap-2'>
      <FontAwesomeIcon icon={faBoxes} />
      <FontAwesomeIcon icon={faUserGear} />
    </div>
  )
};
