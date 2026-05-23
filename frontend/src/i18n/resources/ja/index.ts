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
import zhCN from '../zh-cn/index';
import { mergeResources } from '../merge';

const ja = mergeResources(
  base,
  trading,
  dashboard,
  accounts,
  aiCore,
  aiDebate,
  aiWizard,
  aiSettings,
  aiStore,
  analytics,
  logs,
  strategy,
  { admin: zhCN.admin },
) as const;

export default ja;
