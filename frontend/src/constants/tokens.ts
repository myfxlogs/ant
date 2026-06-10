/**
 * Design tokens — single source of truth for colors, shadows, and layout constants.
 * Mirrors the CSS custom properties defined in src/index.css :root.
 *
 * Usage:
 *   import { COLORS, SHADOWS } from '@/constants/tokens';
 *   <div style={{ color: COLORS.text, background: COLORS.bgCard, boxShadow: SHADOWS.card }} />
 */

export const COLORS = {
  primary: '#D4AF37',
  primaryLight: '#E6C65C',
  primaryDark: '#B8960B',
  accent: '#141D22',
  bgMain: '#FFFFFF',
  bgCard: '#FFFFFF',
  bgSecondary: '#F5F7F9',
  bgTertiary: '#E8ECF0',
  border: 'rgba(0, 0, 0, 0.08)',
  text: '#141D22',
  textSecondary: '#5A6B75',
  textMuted: '#8A9AA5',
  success: '#00A651',
  danger: '#E53935',
  info: '#2196F3',
  white: '#FFFFFF',
} as const;

export const SHADOWS = {
  card: '0 2px 8px rgba(0, 0, 0, 0.06)',
  cardHover: '0 4px 16px rgba(0, 0, 0, 0.08)',
  elevated: '0 4px 12px rgba(0, 0, 0, 0.1)',
  glow: '0 0 20px rgba(212, 175, 55, 0.2)',
} as const;

export const RADIUS = {
  sm: '4px',
  md: '8px',
  lg: '10px',
  xl: '12px',
  '2xl': '16px',
} as const;

export const GRADIENTS = {
  primary: 'linear-gradient(135deg, #D4AF37 0%, #B8960B 100%)',
  primaryHover: 'linear-gradient(135deg, #E6C65C 0%, #D4AF37 100%)',
} as const;

export const LAYOUT = {
  sidebarWidth: 240,
  contentMaxWidth: 1360,
  topBarHeight: 64,
  mobileBreakpoint: 992,
} as const;
