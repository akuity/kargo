import { faExternalLinkAlt } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';

export const PrLinks = ({
  repoUrls,
  metadata
}: {
  repoUrls?: string[];
  metadata?: { [key: string]: string };
}) => (
  <div className='flex gap-2 flex-wrap'>
    {(repoUrls || []).map((repo, i) => {
      const url = metadata && metadata[`pr-url:${repo}`];
      return url ? (
        <Tooltip title={url} key={i}>
          <a href={url} className='cursor-pointer' target='_blank'>
            <FontAwesomeIcon icon={faExternalLinkAlt} />
          </a>
        </Tooltip>
      ) : null;
    })}
  </div>
);
