import classNames from 'classnames';
import { CSSProperties, useLayoutEffect, useRef, useState } from 'react';

import { getOverflow, getSlideDuration } from './overflow-text-utils';
import styles from './overflow-text.module.less';

type OverflowTextProps = {
  text: string;
  title?: string;
  className?: string;
};

// OverflowText shows text on one line, e.g. a freight's alias or an artifact's
// subscription. If it is too long, it fades out where it is cut off, and
// hovering it slides it left to show the rest.
export const OverflowText = ({ text, title, className }: OverflowTextProps) => {
  const boxRef = useRef<HTMLSpanElement>(null);
  const textRef = useRef<HTMLSpanElement>(null);
  // how much of the text is cut off, in pixels
  const [overflow, setOverflow] = useState(0);

  useLayoutEffect(() => {
    const boxElement = boxRef.current;
    const textElement = textRef.current;
    if (!boxElement || !textElement) return;

    // The card stretches with the freightline, so measure on every resize.
    const measure = () =>
      setOverflow(
        getOverflow(
          textElement.getBoundingClientRect().width,
          boxElement.getBoundingClientRect().width
        )
      );

    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(boxElement);
    observer.observe(textElement);
    return () => observer.disconnect();
  }, []);

  const overflowing = overflow > 0;

  const style = overflowing
    ? ({
        '--overflow-shift': `-${overflow}px`,
        '--overflow-duration': `${getSlideDuration(overflow)}ms`
      } as CSSProperties)
    : undefined;

  return (
    <span
      ref={boxRef}
      className={classNames(styles.box, overflowing && styles.overflowing, className)}
      style={style}
      title={title}
    >
      <span ref={textRef} className={styles.text}>
        {text}
      </span>
    </span>
  );
};
