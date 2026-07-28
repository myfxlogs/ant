import { useCallback } from 'react';
import { useWorkspaceStore } from '@/stores/workspaceStore';

export function useWorkspaceResize() {
  const rightPanelWidth = useWorkspaceStore(s => s.rightPanelWidth);
  const setRightPanelWidth = useWorkspaceStore(s => s.setRightPanelWidth);

  const startResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    const startX = e.clientX;
    const startW = rightPanelWidth;
    const onMove = (ev: MouseEvent) => { setRightPanelWidth(startW + (startX - ev.clientX)); };
    const onUp = () => {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  }, [rightPanelWidth, setRightPanelWidth]);

  return { rightPanelWidth, startResize };
}
