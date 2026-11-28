// Ant Design closes an open Drawer or Modal on Escape through its own window
// listener, which runs alongside ours and cannot stop it. While one is open,
// leave Escape to it so a single press closes one layer at a time.
const openLayerSelector = '.ant-drawer-open, .ant-modal-wrap:not([style*="display: none"])';

export const escapeCancelsAction = (
  event: Pick<KeyboardEvent, 'key' | 'isComposing' | 'defaultPrevented'>,
  root: Pick<ParentNode, 'querySelector'>
) =>
  event.key === 'Escape' &&
  !event.isComposing &&
  !event.defaultPrevented &&
  !root.querySelector(openLayerSelector);
