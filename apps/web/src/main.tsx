import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import 'reactflow/dist/style.css';
import '@vscode/codicons/dist/codicon.css';

import { App } from './App';
import './styles/vscode.css';

const rootEl = document.getElementById('root');
if (!rootEl) {
  throw new Error('Root element #root not found');
}

createRoot(rootEl).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
