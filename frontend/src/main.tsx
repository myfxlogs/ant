import { createRoot } from 'react-dom/client';
import './i18n';
import './index.css';
import './styles/message.css';
import App from './App';

createRoot(document.getElementById('root')!).render(
  <App />
);
