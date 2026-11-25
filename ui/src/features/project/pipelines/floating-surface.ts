import classNames from 'classnames';

// The surface of what floats over the pipeline graph -- the freight timeline
// and the toolbar: translucent, blurring what is underneath so what sits on it
// stays legible. White in the light theme, the page background in the dark
// one. Important, so that it overrides the background of an AntD Card.
export const floatingSurfaceClassName = classNames(
  '!bg-[color:color-mix(in_srgb,var(--kargo-color-bg-container)_40%,transparent)]',
  'dark:!bg-[color:color-mix(in_srgb,var(--kargo-color-bg-layout)_40%,transparent)]',
  'backdrop-blur-md'
);

// The same surface, opaque, for when what is underneath is dimmed: through the
// translucent one, the dimming would grey it. It is the color the translucent
// surface takes over the bare graph canvas (bg-layout), so the mix is kept in
// step with the one above; in the dark theme that is the canvas itself.
export const solidSurfaceClassName = classNames(
  'bg-[color:color-mix(in_srgb,var(--kargo-color-bg-container)_40%,var(--kargo-color-bg-layout))]',
  'dark:bg-[color:var(--kargo-color-bg-layout)]'
);

// The surface's border is translucent too, so the blur shows through it
// rather than a hard line. The light theme takes it from the stronger border
// token, as the lighter one all but vanishes on the white surface; the dark
// value is set in app.less.
export const floatingSurfaceBorderColor =
  'var(--kargo-floating-surface-border, color-mix(in srgb, var(--kargo-color-border, #d9d9d9) 50%, transparent))';
