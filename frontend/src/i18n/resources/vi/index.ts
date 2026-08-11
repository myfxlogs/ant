import base from './base';
import trading from './trading';
import dashboard from './dashboard';
import accounts from './accounts';
import aiCore from './ai';
import aiWizard from './ai_wizard';
import aiSettings from './ai_settings';
import aiStore from './ai_store';
import analytics from './analytics';
import logs from './logs';
import strategy from './strategy';
import strategyWorkspace from './strategy_workspace';
import errors from './errors';
import admin from './admin';
import StrategyBacktestParams from './strategy_backtest_params';
import StrategyTuning from './strategy_tuning';
import StrategyAi from './strategy_ai';
import StrategyBacktest from './strategy_backtest';
import StrategyCodeAssist from './strategy_code_assist';
import StrategyCodeQuality from './strategy_code_quality';
import StrategyCodeEditor from './strategy_code_editor';
import StrategyChartTools from './strategy_chart_tools';
import StrategyQuickTradeSection from './strategy_quick_trade_section';
import StrategyLibrary from './strategy_library';
import StrategyTemplates from './strategy_templates';
import StrategyExperiment from './strategy_experiment';
import StrategyMarketRegime from './strategy_market_regime';
import StrategyAsset from './strategy_asset';
import StrategyAssetAnalysis from './strategy_asset_analysis';
import StrategyBacktestRun from './strategy_backtest_run';
import StrategySchedules from './strategy_schedules';
import StrategyScheduleLogs from './strategy_schedule_logs';
import StrategyGen from './strategy_gen';
import StrategyAiChat from './strategy_ai_chat';
import StrategyPaper from './strategy_paper';
import StrategyDefaultTemplates from './strategy_default_templates';
import AiCore from './ai_core';
import Marketplace from './marketplace';
import landing from './landing';
import IndicatorCatalog from './indicator_catalog';
import { mergeResources } from '../merge';

import Menu from './menu';
import Notifications from './notifications';
import Onboarding from './onboarding';
import Common from './common';
import ImportAnalysis from './importAnalysis';
import Theme from './theme';
import Share from './share';
import Monitoring from './monitoring';
import Sre from './sre';
import Auth from './auth';
import SharePage from './sharePage';
import Subscription from './subscription';
import ErrorsExtra from './errors_extra';
import StrategyExtra from './strategy_extra';
import AccountsExtra from './accounts_extra';
import AdminExtra from './admin_extra';
import DashboardExtra from './dashboard_extra';
import MarketplaceExtra from './marketplace_extra';
import LandingExtra from './landing_extra';

const vi = mergeResources(
  base,
  dashboard,
  trading,
  accounts,
  aiCore,
  aiWizard,
  aiSettings,
  aiStore,
  analytics,
  logs,
  strategy,
  StrategyDefaultTemplates,
  StrategyPaper,
  StrategyAiChat,
  StrategyGen,
  StrategyScheduleLogs,
  StrategySchedules,
  StrategyBacktestRun,
  StrategyAssetAnalysis,
  StrategyAsset,
  StrategyMarketRegime,
  StrategyExperiment,
  StrategyTemplates,
  StrategyLibrary,
  StrategyQuickTradeSection,
  StrategyChartTools,
  StrategyCodeEditor,
  StrategyCodeQuality,
  StrategyCodeAssist,
  StrategyBacktest,
  StrategyAi,
  StrategyTuning,
  StrategyBacktestParams,
  strategyWorkspace,
  errors,
  admin,
  AiCore,
  Marketplace,
  IndicatorCatalog,
  landing,  Menu,
  Notifications,
  Onboarding,
  Common,
  ImportAnalysis,
  Theme,
  Share,
  Monitoring,
  Sre,
  Auth,
  SharePage,
  Subscription,
  ErrorsExtra,
  StrategyExtra,
  AccountsExtra,
  AdminExtra,
  DashboardExtra,
  MarketplaceExtra,
  LandingExtra,
);

export default vi;
