import { theme } from 'antd';

// A tone is the whole of how a status is colored. Every status the UI states
// -- a Stage's health, its phase, what a promotion is waiting on -- picks one
// of these, and nothing picks a color.
export type Tone = 'success' | 'error' | 'warning' | 'processing' | 'muted';

// useToneColor resolves a tone against the live theme. The colors are AntD's
// status text tokens, overridden in themeConfig.ts, so a status reads the same
// wherever it is stated and in either theme, and retoning the app is a change
// to the theme rather than to each of the places that state a status.
export const useToneColor = (tone?: Tone): string | undefined => {
  const { token } = theme.useToken();

  const colors: Record<Tone, string> = {
    success: token.colorSuccessText,
    error: token.colorErrorText,
    warning: token.colorWarningText,
    processing: token.colorInfoText,
    // Nothing to report is not a verdict, so it steps back from the statuses
    // it stands among rather than competing with them.
    muted: token.colorTextTertiary
  };

  return tone ? colors[tone] : undefined;
};
