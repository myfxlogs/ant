import { describe, it, expect, beforeEach } from 'vitest'
import { useNotificationStore } from '@/stores/notificationStore'
import type { Notification } from '@/types/notification'

function makeNotificationInput(overrides: Record<string, unknown> = {}): Omit<Notification, 'id' | 'read' | 'created_at'> {
  return {
    type: 'system',
    title: 'Test',
    message: 'Test message',
    ...overrides,
  } as Omit<Notification, 'id' | 'read' | 'created_at'>
}

describe('notificationStore', () => {
  beforeEach(() => {
    useNotificationStore.setState({ notifications: [], unreadCount: 0 })
  })

  it('starts empty', () => {
    const s = useNotificationStore.getState()
    expect(s.notifications).toHaveLength(0)
    expect(s.unreadCount).toBe(0)
  })

  it('addNotification prepends and increments unreadCount', () => {
    useNotificationStore.getState().addNotification(makeNotificationInput({ title: 'A' }))
    const s = useNotificationStore.getState()
    expect(s.notifications).toHaveLength(1)
    expect(s.notifications[0].title).toBe('A')
    expect(s.unreadCount).toBe(1)
  })

  it('addNotification generates unique id and sets read=false', () => {
    useNotificationStore.getState().addNotification(makeNotificationInput())
    useNotificationStore.getState().addNotification(makeNotificationInput())
    const s = useNotificationStore.getState()
    expect(s.notifications).toHaveLength(2)
    expect(s.notifications[0].id).not.toBe(s.notifications[1].id)
    expect(s.notifications.every((n) => !n.read)).toBe(true)
  })

  it('addNotification caps at 100', () => {
    for (let i = 0; i < 105; i++) {
      useNotificationStore.getState().addNotification(makeNotificationInput({ title: `N${i}` }))
    }
    expect(useNotificationStore.getState().notifications).toHaveLength(100)
  })

  it('markAsRead sets read=true and decrements unreadCount', () => {
    useNotificationStore.getState().addNotification(makeNotificationInput())
    const id = useNotificationStore.getState().notifications[0].id
    useNotificationStore.getState().markAsRead(id)
    const s = useNotificationStore.getState()
    expect(s.notifications[0].read).toBe(true)
    expect(s.unreadCount).toBe(0)
  })

  it('markAllAsRead sets all read=true and clears unreadCount', () => {
    useNotificationStore.getState().addNotification(makeNotificationInput())
    useNotificationStore.getState().addNotification(makeNotificationInput())
    useNotificationStore.getState().markAllAsRead()
    const s = useNotificationStore.getState()
    expect(s.unreadCount).toBe(0)
    expect(s.notifications.every((n) => n.read)).toBe(true)
  })

  it('removeNotification removes by id and recalculates unreadCount', () => {
    useNotificationStore.getState().addNotification(makeNotificationInput())
    useNotificationStore.getState().addNotification(makeNotificationInput())
    const id0 = useNotificationStore.getState().notifications[0].id
    useNotificationStore.getState().removeNotification(id0)
    const s = useNotificationStore.getState()
    expect(s.notifications).toHaveLength(1)
    expect(s.unreadCount).toBe(1)
  })

  it('clearAll empties everything', () => {
    useNotificationStore.getState().addNotification(makeNotificationInput())
    useNotificationStore.getState().clearAll()
    const s = useNotificationStore.getState()
    expect(s.notifications).toHaveLength(0)
    expect(s.unreadCount).toBe(0)
  })
})
