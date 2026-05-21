import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import '@vscode/codicons/dist/codicon.css';

import { App } from './App';
import { getStoredTheme, initTheme } from './lib/theme';
import { QueryProvider } from './providers/QueryProvider';
import { useStore } from './store';
import './styles/themes.css';
import './styles/vscode.css';

initTheme();
useStore.setState({ theme: getStoredTheme() });

const rootEl = document.getElementById('root');
if (!rootEl) {
  throw new Error('Root element #root not found');
}

createRoot(rootEl).render(
  <StrictMode>
    <QueryProvider>
      <App />
    </QueryProvider>
  </StrictMode>,
);
