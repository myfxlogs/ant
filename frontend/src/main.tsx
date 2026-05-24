import { createRoot } from 'react-dom/client';
import './i18n';
import i18n from './i18n';
import './index.css';
import './styles/message.css';

// Ensure i18n is fully initialized before React renders.
// Without this, lazy-loaded components using useTranslation()
// may receive a non-function `t` placeholder.
async function bootstrap() {
  if (!i18n.isInitialized) {
    await i18n.init();
  }
  await Promise.resolve();
  const { default: App } = await import('./App');
  createRoot(document.getElementById('root')!).render(<App />);
}

bootstrap();
