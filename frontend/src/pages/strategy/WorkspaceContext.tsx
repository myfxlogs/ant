import { createContext, useContext, type ReactNode } from 'react';
import { useStrategyWorkspaceState } from './hooks/useStrategyWorkspaceState';

type WsState = ReturnType<typeof useStrategyWorkspaceState>;
export type WsAccount = WsState['account'];
export type WsCode = WsState['code'];
export type WsTemplates = WsState['templates'];

const AccountCtx = createContext<WsState['account']>(null!);
const CodeCtx = createContext<WsState['code']>(null!);
const TemplatesCtx = createContext<WsState['templates']>(null!);
const BacktestCtx = createContext<WsState['backtest']>(null!);
const TuningCtx = createContext<WsState['tuning']>(null!);
const GateCtx = createContext<WsState['gate']>(null!);
const QuickTradeCtx = createContext<WsState['quickTrade']>(null!);
const LayoutCtx = createContext<WsState['layout']>(null!);
const HistoryCtx = createContext<WsState['history'] & { autoExpandHistory: boolean }>(null!);
const AICtx = createContext<WsState['ai']>(null!);

export function WorkspaceProvider({ children }: { children: ReactNode }) {
  const ws = useStrategyWorkspaceState();

  return (
    <AccountCtx.Provider value={ws.account}>
      <CodeCtx.Provider value={ws.code}>
        <TemplatesCtx.Provider value={ws.templates}>
          <BacktestCtx.Provider value={ws.backtest}>
            <TuningCtx.Provider value={ws.tuning}>
              <GateCtx.Provider value={ws.gate}>
                <QuickTradeCtx.Provider value={ws.quickTrade}>
                  <LayoutCtx.Provider value={ws.layout}>
                    <HistoryCtx.Provider value={{ ...ws.history, autoExpandHistory: ws.autoExpandHistory }}>
                      <AICtx.Provider value={ws.ai}>
                        {children}
                      </AICtx.Provider>
                    </HistoryCtx.Provider>
                  </LayoutCtx.Provider>
                </QuickTradeCtx.Provider>
              </GateCtx.Provider>
            </TuningCtx.Provider>
          </BacktestCtx.Provider>
        </TemplatesCtx.Provider>
      </CodeCtx.Provider>
    </AccountCtx.Provider>
  );
}

export const useWsAccount = () => useContext(AccountCtx);
export const useWsCode = () => useContext(CodeCtx);
export const useWsTemplates = () => useContext(TemplatesCtx);
export const useWsBacktest = () => useContext(BacktestCtx);
export const useWsTuning = () => useContext(TuningCtx);
export const useWsQuickTrade = () => useContext(QuickTradeCtx);
export const useWsLayout = () => useContext(LayoutCtx);
export const useWsHistory = () => useContext(HistoryCtx);
export const useWsAI = () => useContext(AICtx);

