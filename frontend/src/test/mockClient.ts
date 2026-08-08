/**
 * ConnectRPC client mock helper — lets component tests inject mock responses
 * per service/method without making real network calls.
 *
 * Usage:
 *   const mock = createMockClient();
 *   mock.mockMethod(AuthService, 'login', { user: {...}, accessToken: 'xxx' });
 *   // or mock a method to throw:
 *   mock.mockMethod(AuthService, 'login', new Error('auth failed'));
 *
 * Then vi.mock the client module to return mock proxies.
 */
import { vi } from 'vitest'

type MockResponse = Record<string, unknown> | Error

interface MockEntry {
  response: MockResponse
}

/**
 * Create a proxy client that returns mock responses for any method call.
 * Methods are matched by name; if no mock is set, returns an empty object.
 */
export function createMockClient() {
  const mocks = new Map<string, MockEntry>()

  function key(serviceName: string, methodName: string): string {
    return `${serviceName}.${methodName}`
  }

  return {
    /**
     * Register a mock response for a service.method call.
     * Pass an Error to make the method reject.
     */
    mockMethod(serviceName: string, methodName: string, response: MockResponse) {
      mocks.set(key(serviceName, methodName), { response })
    },

    /** Clear all mocks. */
    clear() {
      mocks.clear()
    },

    /** Create a proxy object whose methods return the mock responses. */
    createProxy(serviceName: string): Record<string, ReturnType<typeof vi.fn>> {
      return new Proxy({} as Record<string, ReturnType<typeof vi.fn>>, {
        get(_target, prop: string) {
          if (prop === '$mocks' || prop === 'then' || typeof prop === 'symbol') {
            return undefined
          }
          const k = key(serviceName, prop)
          const entry = mocks.get(k)
          return vi.fn().mockImplementation((..._args: unknown[]) => {
            if (entry?.response instanceof Error) {
              throw entry.response
            }
            return entry?.response ?? {}
          })
        },
      })
    },
  }
}
