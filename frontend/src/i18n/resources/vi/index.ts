import base from './base';
import trading from './trading';
import dashboard from './dashboard';
import accounts from './accounts';
import ai from './ai';
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
  ...analytics,
  ...logs,
  ...strategy,
  ...errors,
} as const;

export default vi;
