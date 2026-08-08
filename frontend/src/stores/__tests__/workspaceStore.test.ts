import { describe, it, expect, beforeEach } from 'vitest'
import { useWorkspaceStore } from '@/stores/workspaceStore'

describe('workspaceStore', () => {
  beforeEach(() => {
    useWorkspaceStore.setState({
      accountId: '',
      symbol: '',
      timeframe: '1h',
      centerTab: 'code',
      leftSidebarCollapsed: false,
      bottomPanelCollapsed: false,
      bottomPanelHeight: 160,
      bottomPanelUserResized: false,
      quickTradeCollapsed: false,
      aiPanelWidth: 380,
      currentCode: '',
      currentCodeName: '',
      _hasHydrated: false,
    })
  })

  it('has correct defaults', () => {
    const s = useWorkspaceStore.getState()
    expect(s.accountId).toBe('')
    expect(s.symbol).toBe('')
    expect(s.timeframe).toBe('1h')
    expect(s.centerTab).toBe('code')
    expect(s.leftSidebarCollapsed).toBe(false)
    expect(s.bottomPanelCollapsed).toBe(false)
    expect(s.bottomPanelHeight).toBe(160)
    expect(s.quickTradeCollapsed).toBe(false)
    expect(s.aiPanelWidth).toBe(380)
  })

  it('setAccountId updates accountId', () => {
    useWorkspaceStore.getState().setAccountId('acc-1')
    expect(useWorkspaceStore.getState().accountId).toBe('acc-1')
  })

  it('setSymbol updates symbol', () => {
    useWorkspaceStore.getState().setSymbol('EURUSD')
    expect(useWorkspaceStore.getState().symbol).toBe('EURUSD')
  })

  it('setTimeframe updates timeframe', () => {
    useWorkspaceStore.getState().setTimeframe('4h')
    expect(useWorkspaceStore.getState().timeframe).toBe('4h')
  })

  it('setCenterTab toggles between chat and code', () => {
    useWorkspaceStore.getState().setCenterTab('chat')
    expect(useWorkspaceStore.getState().centerTab).toBe('chat')
    useWorkspaceStore.getState().setCenterTab('code')
    expect(useWorkspaceStore.getState().centerTab).toBe('code')
  })

  it('setLeftSidebarCollapsed toggles sidebar', () => {
    useWorkspaceStore.getState().setLeftSidebarCollapsed(true)
    expect(useWorkspaceStore.getState().leftSidebarCollapsed).toBe(true)
  })

  it('setBottomPanelHeight updates height', () => {
    useWorkspaceStore.getState().setBottomPanelHeight(300)
    expect(useWorkspaceStore.getState().bottomPanelHeight).toBe(300)
  })

  it('setBottomPanelUserResized marks user resize', () => {
    useWorkspaceStore.getState().setBottomPanelUserResized(true)
    expect(useWorkspaceStore.getState().bottomPanelUserResized).toBe(true)
  })

  it('setCurrentCode updates code', () => {
    useWorkspaceStore.getState().setCurrentCode('void OnTick() {}')
    expect(useWorkspaceStore.getState().currentCode).toBe('void OnTick() {}')
  })

  it('setCurrentCodeName updates code name', () => {
    useWorkspaceStore.getState().setCurrentCodeName('MyEA.mq4')
    expect(useWorkspaceStore.getState().currentCodeName).toBe('MyEA.mq4')
  })

  it('setAiPanelWidth updates width', () => {
    useWorkspaceStore.getState().setAiPanelWidth(500)
    expect(useWorkspaceStore.getState().aiPanelWidth).toBe(500)
  })
})
