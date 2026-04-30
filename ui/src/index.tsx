import '@ant-design/v5-patch-for-react-19';

import { createRoot } from 'react-dom/client';

import { App } from './app';

const container = document.getElementById('root');
if (!container) throw new Error('Failed to find the root element');

const root = createRoot(container);
root.render(<App />);
