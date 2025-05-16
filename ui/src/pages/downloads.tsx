import { IconDefinition, faApple, faLinux, faWindows } from '@fortawesome/free-brands-svg-icons';
import { faCircleDown, faCodeCommit, faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

import { PageTitle } from '@ui/features/common';

const DownloadLink = ({ url, children }: { url: string; children: React.ReactNode }) => (
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

const DownloadItem = ({
  title,
  icon,
  links
}: {
  title: string;
  icon: IconDefinition;
  links: { url: string; title?: string }[];
}) => (
  <div className='p-4 rounded bg-gray-100 flex flex-col' style={{ width: '250px' }}>
    <div className='flex items-center mb-4 text-gray-800 font-medium text-lg mx-auto'>
      <FontAwesomeIcon icon={icon} className='mr-2' />
      {title}
    </div>
    <div className='flex items-center gap-2 justify-center'>
      {links.map((link, index) => (
        <DownloadLink key={index} url={link.url}>
          {link.title || 'Download'}
        </DownloadLink>
      ))}
    </div>
  </div>
);

const constructUrl = (platform: string, version?: string) =>
  `https://github.com/akuity/kargo/releases/${version ? '' : 'latest/'}download/${version ? `${version}/` : ''}${platform}`;

export const Downloads = () => (
  <div className='p-6'>
    <PageTitle title='CLI Downloads' />
    <div className='text-2xl mb-2 font-semibold flex items-center'>
      <FontAwesomeIcon icon={faCodeCommit} className='mr-2' />
      {__UI_VERSION__ === 'development' ? 'Latest version' : __UI_VERSION__}
    </div>
    <a
      href='https://github.com/akuity/kargo/releases'
      target='_blank'
      rel='noreferrer'
      className='mb-6 flex text-xs items-center text-blue-500 uppercase'
    >
      <FontAwesomeIcon icon={faExternalLink} className='mr-2' />
      View all releases
    </a>
    <div className='flex items-center gap-4 flex-wrap'>
      <DownloadItem
        title='Mac'
        icon={faApple}
        links={[
          { url: constructUrl('kargo-darwin-arm64'), title: 'Apple Silicon' },
          { url: constructUrl('kargo-darwin-amd64'), title: 'Intel' }
        ]}
      />
      <DownloadItem
        title='Windows'
        icon={faWindows}
        links={[
          { url: `${constructUrl('kargo-windows-arm64')}.exe`, title: 'ARM' },
          { url: `${constructUrl('kargo-windows-amd64')}.exe`, title: 'x86' }
        ]}
      />
      <DownloadItem
        title='Linux'
        icon={faLinux}
        links={[
          { url: constructUrl('kargo-linux-arm64'), title: 'ARM' },
          { url: constructUrl('kargo-linux-amd64'), title: 'x86' }
        ]}
      />
    </div>
  </div>
);
