import MDXComponents from '@theme-original/MDXComponents';
import Highlight from '@site/src/components/Highlight';

// TODO: Use css
const AkuityHighlight = ({children}) => {
  return Highlight({children, color: '#3F4C60' })
}
export default {
  ...MDXComponents,

  hlt: AkuityHighlight,
};
