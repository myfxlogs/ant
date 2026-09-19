// i18n-key-presence.test.ts — I18N-MIXED-1 返修 R6 回归守卫（2026-09-19）。
//
// 43 个历史"生成文件私货"key（strategy.live.diag.* x38 + logs.signalType.* x5）
// 已吸收进 textproto/map SSOT（返修 R3）。本测试断言 5 个 locale 的合并资源
// 树中全部 43 路径解析为非空字符串——从 textproto 删任一 key → i18n-build
// 重生成 → 对应断言 RED（即 R3 的对抗证明载体）。
//
// 另断言：strategy_gen.execFeedback.placeholder zh-cn 存在（S3 修复）与
// 5 条 R4 译文在 zh-cn 资源中生效。
import { describe, expect, it } from 'vitest';
import BaseZhCn from '../i18n/resources/zh-cn/base';
import BaseZhTw from '../i18n/resources/zh-tw/base';
import BaseJa from '../i18n/resources/ja/base';
import BaseVi from '../i18n/resources/vi/base';
import BaseEn from '../i18n/resources/en/base';
import LogsZhCn from '../i18n/resources/zh-cn/logs';
import LogsZhTw from '../i18n/resources/zh-tw/logs';
import LogsJa from '../i18n/resources/ja/logs';
import LogsVi from '../i18n/resources/vi/logs';
import LogsEn from '../i18n/resources/en/logs';
import GenZhCn from '../i18n/resources/zh-cn/strategy_gen';
import GenZhTw from '../i18n/resources/zh-tw/strategy_gen';

type Tree = Record<string, unknown>;

function at(tree: Tree, path: string): string | undefined {
  let cur: unknown = tree;
  for (const seg of path.split('.')) {
    if (typeof cur !== 'object' || cur === null) return undefined;
    cur = (cur as Tree)[seg];
  }
  return typeof cur === 'string' && cur.length > 0 ? cur : undefined;
}

const DIAG_PATHS = [
  'strategy.live.diag.state.warning',
  'strategy.live.diag.orderTruth',
  'strategy.live.diag.vmOrdersTotal',
  'strategy.live.diag.brokerAccountOrders',
  'strategy.live.diag.strategyMagicOrders',
  'strategy.live.diag.pendingBrokerOrders',
  'strategy.live.diag.scheduleMagic',
  'strategy.live.diag.lastBrokerTicket',
  'strategy.live.diag.vmBrokerMismatch',
  'strategy.live.diag.execution',
  'strategy.live.diag.executionState',
  'strategy.live.diag.orderLifecycle',
  'strategy.live.diag.freshness',
  'strategy.live.diag.financialSource',
  'strategy.live.diag.financialAge',
  'strategy.live.diag.financialFresh',
  'strategy.live.diag.positionsSource',
  'strategy.live.diag.positionsAge',
  'strategy.live.diag.positionsFresh',
  'strategy.live.diag.fresh',
  'strategy.live.diag.stale',
  'strategy.live.diag.na',
  'strategy.live.diag.lifecycle.signal_generated',
  'strategy.live.diag.lifecycle.order_submitting',
  'strategy.live.diag.lifecycle.order_submitted',
  'strategy.live.diag.lifecycle.order_confirmed',
  'strategy.live.diag.lifecycle.order_rejected',
  'strategy.live.diag.lifecycle.order_outcome_unknown',
  'strategy.live.diag.execState.idle',
  'strategy.live.diag.execState.submitting',
  'strategy.live.diag.execState.accepted_unconfirmed',
  'strategy.live.diag.execState.confirmed',
  'strategy.live.diag.execState.deterministic_rejected',
  'strategy.live.diag.execState.outcome_unknown',
  'strategy.live.diag.source.account_summary',
  'strategy.live.diag.source.profit_stream',
  'strategy.live.diag.source.order_update',
  'strategy.live.diag.source.position_snapshot',
];

const SIGNAL_PATHS = [
  'logs.signalType.buy',
  'logs.signalType.sell',
  'logs.signalType.close',
  'logs.signalType.hold',
  'logs.signalType.modify',
];

const LOCALE_BASES: Record<string, Tree> = {
  'zh-cn': BaseZhCn as unknown as Tree,
  'zh-tw': BaseZhTw as unknown as Tree,
  'ja': BaseJa as unknown as Tree,
  'vi': BaseVi as unknown as Tree,
  'en': BaseEn as unknown as Tree,
};

const LOCALE_LOGS: Record<string, Tree> = {
  'zh-cn': LogsZhCn as unknown as Tree,
  'zh-tw': LogsZhTw as unknown as Tree,
  'ja': LogsJa as unknown as Tree,
  'vi': LogsVi as unknown as Tree,
  'en': LogsEn as unknown as Tree,
};

describe('I18N-MIXED-1 R6: absorbed SSOT keys present in all 5 locales', () => {
  it('resolves all 38 strategy.live.diag.* paths in every locale base tree', () => {
    for (const [loc, tree] of Object.entries(LOCALE_BASES)) {
      for (const path of DIAG_PATHS) {
        const v = at(tree, path);
        expect(v, `[${loc}] ${path}`).toBeDefined();
        expect(v, `[${loc}] ${path}`).not.toBe('');
      }
    }
  });

  it('resolves all 5 logs.signalType.* paths in every locale logs tree', () => {
    for (const [loc, tree] of Object.entries(LOCALE_LOGS)) {
      for (const path of SIGNAL_PATHS) {
        const v = at(tree, path);
        expect(v, `[${loc}] ${path}`).toBeDefined();
        expect(v, `[${loc}] ${path}`).not.toBe('');
      }
    }
  });

  it('keeps zh-cn diag translations distinct from en (regenerated from SSOT, not placeholders)', () => {
    // 跨 locale 回退守卫：zh-cn 必须是真译文（如 订单真相），非 en 原文。
    for (const path of ['strategy.live.diag.orderTruth', 'strategy.live.diag.lifecycle.signal_generated']) {
      const zh = at(LOCALE_BASES['zh-cn'], path);
      const en = at(LOCALE_BASES['en'], path);
      expect(zh).toBeDefined();
      expect(zh).not.toEqual(en);
    }
  });

  it('keeps strategy_gen.execFeedbackPlaceholder present in zh-cn (S3 fix)', () => {
    const v = at(GenZhCn as unknown as Tree, 'strategy.gen.execFeedbackPlaceholder');
    expect(v).toBeDefined();
    expect(v).toContain('把止损收紧到 1%');
  });

  it('keeps strategy_gen.execFeedbackPlaceholder present in zh-tw (I18N-MIXED-2 S3)', () => {
    const v = at(GenZhTw as unknown as Tree, 'strategy.gen.execFeedbackPlaceholder');
    expect(v).toBeDefined();
    expect(v).toContain('把止損收緊到 1%');
  });

  it('keeps zh-tw diag translations distinct from en (absorbed real translations)', () => {
    for (const path of ['strategy.live.diag.orderTruth', 'strategy.live.diag.lifecycle.signal_generated']) {
      const tw = at(LOCALE_BASES['zh-tw'], path);
      const en = at(LOCALE_BASES['en'], path);
      expect(tw).toBeDefined();
      expect(tw).not.toEqual(en);
    }
    expect(at(LOCALE_BASES['zh-tw'], 'strategy.live.diag.orderTruth')).toEqual('訂單真相');
  });

  it('applies the 5 R4 user-facing translations in zh-cn', () => {
    expect(at(LOCALE_BASES['zh-cn'], 'strategy.workspace.tour.aiDesc')).toContain('让 AI 生成、优化或调试');
    expect(at(LOCALE_BASES['zh-cn'], 'strategy.workspace.tour.backtestDesc')).toContain('使用可配置参数运行回测');
    expect(at(LOCALE_BASES['zh-cn'], 'strategy.backtest.diagnostic.suggestion.iCustom')).toContain('不支持 iCustom');
    expect(at(LOCALE_BASES['zh-cn'], 'strategy.backtest.diagnostic.suggestion.dll')).toContain('不支持 DLL 导入');
    expect(at(LOCALE_BASES['zh-cn'], 'strategy.backtest.diagnostic.silenceHint')).toContain('确认为有意为之');
  });
});
