import Link from 'antd/es/typography/Link';
import classNames from 'classnames';

import { ArtifactReference, Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

import { ArtifactIcon } from './artifact-icon';
import { getArtifactDisplay } from './freight-artifact-utils';
import { OverflowText } from './overflow-text';

// One blue for every artifact type: the value is a chip in it, and the icon
// takes it too. Light values are AntD's geekblue shades (1 for fill, 7 for
// text); dark values come from the freightline design.
const iconClassName = 'shrink-0 w-3.5 mr-1.5 text-[#1d39c4] dark:text-[#8ab8ff]';
const valueClassName = 'inline-block max-w-full truncate rounded px-1 text-[12px] leading-[18px]';
const chipClassName = 'bg-[#f0f5ff] text-[#1d39c4] dark:bg-[#1b2d4a] dark:text-[#8ab8ff]';
// Grey and without the fill, but in the same box, so muted values line up
// with chips.
const mutedClassName = 'text-[#8c8c8c]';
// A subscription label too long for the row fades out at its right edge, and
// slides left on hover to show the rest, like a long alias.
const subscriptionLabelClassName =
  'flex-1 min-w-0 mr-0.5 text-[12px] leading-[18px] text-[color:var(--kargo-color-text-secondary)]';

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
  // When set, the Warehouse subscription the artifact came from, named between
  // the icon and the version.
  subscriptionLabel?: string;
};

// FreightArtifact renders one artifact: its type icon, then its version as a
// chip, linked to its source where there is one. With a subscription label,
// the label follows the icon and the version sits on the right.
export const FreightArtifact = (props: FreightArtifactProps) => {
  const { value, title, url } = getArtifactDisplay(props.artifact);
  const hint = props.changed === undefined ? undefined : props.changed ? CHANGED_HINT : SAME_HINT;

  // Each part says its own thing on hover: the icon which artifact it is, the
  // version how it compares with the previous freight.
  const chip = (
    <span
      className={classNames(
        valueClassName,
        // Next to a subscription label the version has less room; the system
        // UI font is narrower than monospace, so more of it fits.
        props.subscriptionLabel ? 'font-sans' : 'font-mono',
        props.changed === false ? mutedClassName : chipClassName
      )}
      title={hint}
    >
      {value}
      {hint && <span className='sr-only'>, {hint.toLowerCase()}</span>}
    </span>
  );

  // With a subscription label, the version keeps its width on the right and the
  // label gives way first. It is pulled right by its chip's padding, so its text
  // lines up with the card's right edge.
  const valueContainerClassName = classNames(
    'flex min-w-0',
    props.subscriptionLabel && 'shrink-0 max-w-[60%] -mr-1'
  );

  return (
    <div className='flex items-center h-5 min-w-0 max-w-full'>
      <span className='flex shrink-0' title={title}>
        <ArtifactIcon artifact={props.artifact} className={iconClassName} />
      </span>
      {props.subscriptionLabel && (
        // the version's own padding adds to the label's margin, so it is small
        <OverflowText text={props.subscriptionLabel} className={subscriptionLabelClassName} />
      )}
      {url ? (
        <Link
          href={url}
          target='_blank'
          className={valueContainerClassName}
          onClick={(e) => e.stopPropagation()}
        >
          {chip}
        </Link>
      ) : (
        <span className={valueContainerClassName}>{chip}</span>
      )}
    </div>
  );
};
