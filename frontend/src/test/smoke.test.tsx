import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'

// M0.3 smoke tests — verify vitest + @testing-library framework is operational.
// Full page rendering tests require mock setup (router, i18n, auth store, API hooks).
// Those will be built out as the components are refactored in M1-M5.

describe('Vitest + Testing Library smoke', () => {
  it('renders a simple component', () => {
    const { container: _container } = render(<div data-testid="hello">Hello, ant</div>)
    expect(screen.getByTestId('hello')).toBeTruthy()
    expect(screen.getByText('Hello, ant')).toBeTruthy()
  })

  it('supports React Testing Library queries', () => {
    render(
      <ul>
        <li>Login</li>
        <li>Dashboard</li>
        <li>Accounts</li>
      </ul>,
    )
    const items = screen.getAllByRole('listitem')
    expect(items).toHaveLength(3)
    expect(items[0]).toHaveTextContent('Login')
  })
})
