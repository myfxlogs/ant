import base from './base';
import trading from './trading';
import dashboard from './dashboard';
import accounts from './accounts';
import ai from './ai';
import aiDebate from './ai_debate';
import aiSettings from './ai_settings';
import aiStore from './ai_store';
import analytics from './analytics';
import logs from './logs';
import strategy from './strategy';
import errors from './errors';

const vi = {
  ...base,
  ...trading,
  ...dashboard,
  ...accounts,
  ...ai,
  ...aiDebate,
  ...aiSettings,
  ...aiStore,
  ...analytics,
  ...logs,
  ...strategy,
  ...errors,
} as const;

export default vi;
