import Link from 'antd/es/typography/Link';

import { ArtifactReference, Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

import { ArtifactIcon } from './artifact-icon';
import { getArtifactDisplay } from './freight-artifact-utils';

// One blue for every artifact type: the value is a chip in it, and the icon
// takes it too. Light values are AntD's geekblue shades (1 for fill, 7 for
// text); dark values come from the freightline design.
const iconClassName = 'shrink-0 w-3.5 mr-1.5 text-[#1d39c4] dark:text-[#8ab8ff]';
const chipClassName =
  'inline-block max-w-full truncate rounded px-1.5 font-mono text-[12px] leading-[18px] bg-[#f0f5ff] text-[#1d39c4] dark:bg-[#1b2d4a] dark:text-[#8ab8ff]';

type FreightArtifactProps = {
  artifact: GitCommit | Chart | Image | ArtifactReference;
};

// FreightArtifact renders one artifact: its type icon, then its version as a
// chip, linked to its source where there is one.
export const FreightArtifact = (props: FreightArtifactProps) => {
  const { value, title, url } = getArtifactDisplay(props.artifact);

  const chip = <span className={chipClassName}>{value}</span>;

  return (
    <div className='flex items-center h-5 min-w-0 max-w-full' title={title}>
      <ArtifactIcon artifact={props.artifact} className={iconClassName} />
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
