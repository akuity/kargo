import { Tooltip, Typography } from 'antd';

// The hint is a name, which reads as one thing only on one line, so the
// tooltip is let grow to fit it rather than wrap it.
const hintStyles = { root: { maxWidth: 'none' } };

// RoundIdentityLabel says which round this is: a PromotionRequest's name, or a
// PromotionRollout's number with its name one hover away. A number leads the
// line it is on, so its hint opens above it rather than over the words that
// follow. className sets the size, which is the line's own.
export const RoundIdentityLabel = ({
  label,
  hint,
  className = 'text-xs'
}: {
  label: string;
  hint?: string;
  className?: string;
}) => (
  <Tooltip title={hint} placement='topLeft' styles={hintStyles}>
    <Typography.Text type='secondary' className={`font-mono ${className}`}>
      {label}
    </Typography.Text>
  </Tooltip>
);
