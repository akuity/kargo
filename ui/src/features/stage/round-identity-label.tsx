import { Tooltip, Typography } from 'antd';

// The hint is a name, which reads as one thing only on one line, so the
// tooltip is let grow to fit it rather than wrap it.
const hintStyles = { root: { maxWidth: 'none' } };

// RoundIdentityLabel is the line that says which round this is: a
// PromotionRequest's name, or a PromotionRollout's number with its name one
// hover away. The hint opens beside the label, where it covers nothing else
// that is being read.
export const RoundIdentityLabel = ({
  label,
  hint,
  className
}: {
  label: string;
  hint?: string;
  className?: string;
}) => (
  <Tooltip title={hint} placement='right' styles={hintStyles}>
    <Typography.Text type='secondary' className={`text-xs font-mono ${className || ''}`}>
      {label}
    </Typography.Text>
  </Tooltip>
);
