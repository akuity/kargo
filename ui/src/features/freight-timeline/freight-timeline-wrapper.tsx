import { faCaretLeft, faCaretRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex } from 'antd';
import { useLayoutEffect, useRef, useState } from 'react';

import { headerButtonStyle } from './utils';

export const FreightTimelineWrapper = ({ children }: { children: React.ReactNode }) => {
  const timeline = useRef<HTMLDivElement>(null);
  const [showScrollbars, setShowScrollbars] = useState(false);

  useLayoutEffect(() => {
    const handleResize = () => {
      if (timeline.current) {
        const { scrollWidth, clientWidth } = timeline.current;
        setShowScrollbars(scrollWidth > clientWidth);
      }
    };

    handleResize();
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, [timeline]);

  return (
    <div className='w-full py-3 flex flex-col overflow-hidden'>
      <div className='flex h-48 w-full items-center px-1'>
        <div
          className='text-gray-500 text-sm font-semibold mb-2 w-min h-min'
          style={{ transform: 'rotate(-0.25turn)' }}
        >
          NEW
        </div>
        <div className='flex items-center h-full overflow-x-auto w-full' ref={timeline}>
          {children}
        </div>
        <div className='rotate-90 text-gray-500 text-sm font-semibold ml-auto'>OLD</div>
      </div>
      {showScrollbars && (
        <Flex align='center' className='mt-2 pr-2' justify='end'>
          <Button
            onClick={() => {
              timeline.current?.scrollTo({
                left: timeline.current?.scrollLeft - 200,
                behavior: 'smooth'
              });
            }}
            size='small'
            className={headerButtonStyle(false)}
            icon={<FontAwesomeIcon icon={faCaretLeft} />}
          />
          <Button
            onClick={() => {
              timeline.current?.scrollTo({
                left: timeline.current?.scrollLeft + 200,
                behavior: 'smooth'
              });
            }}
            size='small'
            className={headerButtonStyle(false)}
            icon={<FontAwesomeIcon icon={faCaretRight} />}
          />
        </Flex>
      )}
    </div>
  );
};
