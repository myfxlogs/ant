import { create } from 'zustand';

type Theme = 'light' | 'dark';

interface UIState {
  theme: Theme;
  sidebarCollapsed: boolean;
  setTheme: (theme: Theme) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
}

export const useUIStore = create<UIState>((set) => ({
  theme: (localStorage.getItem('ant-theme') as Theme) || 'dark',
  sidebarCollapsed: false,

  setTheme: (theme) => {
    localStorage.setItem('ant-theme', theme);
    set({ theme });
  },

  toggleSidebar: () =>
    set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),

  setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
}));
