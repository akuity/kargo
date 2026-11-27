import Link from 'antd/es/typography/Link';
import classNames from 'classnames';

import { ArtifactReference, Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

import { ArtifactIcon } from './artifact-icon';
import { getArtifactDisplay } from './freight-artifact-utils';

// One blue for every artifact type: the value is a chip in it, and the icon
// takes it too. Light values are AntD's geekblue shades (1 for fill, 7 for
// text); dark values come from the freightline design.
const iconClassName = 'shrink-0 w-3.5 mr-1.5 text-[#1d39c4] dark:text-[#8ab8ff]';
const valueClassName =
  'inline-block max-w-full truncate rounded px-1.5 font-mono text-[12px] leading-[18px]';
const chipClassName = 'bg-[#f0f5ff] text-[#1d39c4] dark:bg-[#1b2d4a] dark:text-[#8ab8ff]';
// Grey and without the fill, but in the same box, so muted values line up
// with chips.
const mutedClassName = 'text-[#8c8c8c]';

// The color says it at a glance; the words say it on hover and to screen
// readers.
const CHANGED_HINT = 'Changed since the previous freight';
const SAME_HINT = 'Same as in the previous freight';

type FreightArtifactProps = {
  artifact: GitCommit | Chart | Image | ArtifactReference;
  // Whether the artifact changed since the previous freight from its
  // Warehouse. One that did not fades back, so the ones that did stand out.
  // Unset where there is nothing to compare with.
  changed?: boolean;
};

// FreightArtifact renders one artifact: its type icon, then its version as a
// chip, linked to its source where there is one.
export const FreightArtifact = (props: FreightArtifactProps) => {
  const { value, title, url } = getArtifactDisplay(props.artifact);
  const hint = props.changed === undefined ? undefined : props.changed ? CHANGED_HINT : SAME_HINT;

  // Each part says its own thing on hover: the icon which artifact it is, the
  // version how it compares with the previous freight.
  const chip = (
    <span
      className={classNames(
        valueClassName,
        props.changed === false ? mutedClassName : chipClassName
      )}
      title={hint}
    >
      {value}
      {hint && <span className='sr-only'>, {hint.toLowerCase()}</span>}
    </span>
  );

  return (
    <div className='flex items-center h-5 min-w-0 max-w-full'>
      <span className='flex shrink-0' title={title}>
        <ArtifactIcon artifact={props.artifact} className={iconClassName} />
      </span>
      {url ? (
        <Link
          href={url}
          target='_blank'
          className='flex min-w-0'
          onClick={(e) => e.stopPropagation()}
        >
          {chip}
        </Link>
      ) : (
        chip
      )}
    </div>
  );
};
