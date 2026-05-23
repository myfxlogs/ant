import base from './base';
import trading from './trading';
import dashboard from './dashboard';
import accounts from './accounts';
import aiCore from './ai';
import aiDebate from './ai_debate';
import aiWizard from './ai_wizard';
import aiSettings from './ai_settings';
import aiStore from './ai_store';
import analytics from './analytics';
import logs from './logs';
import strategy from './strategy';
import { mergeResources } from '../merge';

const en = mergeResources(
  base,
  dashboard,
  trading,
  accounts,
  aiCore,
  aiDebate,
  aiWizard,
  aiSettings,
  aiStore,
  analytics,
  logs,
  strategy,
) as const;

export default en;
