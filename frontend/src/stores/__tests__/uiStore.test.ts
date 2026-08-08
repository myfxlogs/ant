import { describe, it, expect, beforeEach } from 'vitest'
import { useUIStore } from '@/stores/uiStore'

describe('uiStore', () => {
  beforeEach(() => {
    useUIStore.setState({ theme: 'light', sidebarCollapsed: false })
  })

  it('starts with light theme and expanded sidebar', () => {
    const s = useUIStore.getState()
    expect(s.theme).toBe('light')
    expect(s.sidebarCollapsed).toBe(false)
  })

  it('setTheme updates theme', () => {
    useUIStore.getState().setTheme('dark')
    expect(useUIStore.getState().theme).toBe('dark')
  })

  it('toggleSidebar flips collapsed state', () => {
    expect(useUIStore.getState().sidebarCollapsed).toBe(false)
    useUIStore.getState().toggleSidebar()
    expect(useUIStore.getState().sidebarCollapsed).toBe(true)
    useUIStore.getState().toggleSidebar()
    expect(useUIStore.getState().sidebarCollapsed).toBe(false)
  })

  it('setSidebarCollapsed sets explicitly', () => {
    useUIStore.getState().setSidebarCollapsed(true)
    expect(useUIStore.getState().sidebarCollapsed).toBe(true)
  })
})
