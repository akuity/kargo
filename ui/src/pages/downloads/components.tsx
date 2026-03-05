import { faCircleDown } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

import { Platform, Release } from './types';

export const DownloadLink = ({ url, children }: { url: string; children: React.ReactNode }) => (
  <a
    href={url}
    target='_blank'
    rel='noreferrer'
    className='bg-blue-500 inline-flex items-center text-white px-2 py-1 rounded text-sm font-medium no-underline'
  >
    <FontAwesomeIcon icon={faCircleDown} className='mr-2' />
    {children}
  </a>
);

export const DownloadItem = ({ title, icon, links, release }: Platform & { release?: Release }) => (
  <div className='p-4 rounded bg-gray-100 flex flex-col' style={{ width: '250px' }}>
    <div className='flex items-center mb-4 text-gray-800 font-medium text-lg mx-auto'>
      <FontAwesomeIcon icon={icon} className='mr-2' />
      {title}
    </div>
    <div className='flex items-center gap-2 justify-center'>
      {links.map((link) => (
        <DownloadLink key={link.title} url={link.getUrl(release)}>
          {link.title}
        </DownloadLink>
      ))}
    </div>
  </div>
);
