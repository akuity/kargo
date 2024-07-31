import { faCheck, faClipboard } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useState } from 'react';

export const TruncatedCopyable = ({ text }: { text?: string }) => {
  const [copied, setCopied] = useState(false);
  return (
    text && (
      <div
        className='flex items-center cursor-pointer hover:text-blue-500'
        onClick={() => {
          navigator.clipboard.writeText(text);
          setCopied(true);
          setTimeout(() => setCopied(false), 1000);
        }}
      >
        <FontAwesomeIcon icon={copied ? faCheck : faClipboard} className='mr-2 w-3 text-gray-400' />
        <div className='truncate font-mono text-xs' style={{ maxWidth: '200px' }}>
          {text}
        </div>
      </div>
    )
  );
};
