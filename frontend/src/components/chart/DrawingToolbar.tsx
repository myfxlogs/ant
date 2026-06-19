import { useState } from 'react';
import { Tooltip } from 'antd';
import { useTranslation } from 'react-i18next'
import { CLEAR_DRAWINGS_KEY } from '@/gen/ant/v1/i18n/strategy_chart_tools_keys';

;
import {
  LineChartOutlined, MinusOutlined, ColumnWidthOutlined,
  VerticalAlignMiddleOutlined, ArrowUpOutlined, DashOutlined,
  BorderOuterOutlined, DollarOutlined, ExpandOutlined,
  ItalicOutlined, DeleteOutlined,
} from '@ant-design/icons';
import type { Chart } from 'klinecharts';

interface ToolDef {
  key: string;
  overlayName: string;
  icon: React.ReactNode;
  label: string;
}

const DRAWING_TOOLS: ToolDef[] = [
  { key: 'measure', overlayName: 'priceRangeMeasure', icon: <ExpandOutlined />, label: 'Measure (Shift+drag)' },
  { key: 'segment', overlayName: 'segment', icon: <ItalicOutlined />, label: 'Trend Line' },
  { key: 'horizontalLine', overlayName: 'horizontalStraightLine', icon: <MinusOutlined />, label: 'Horizontal Line' },
  { key: 'verticalLine', overlayName: 'verticalStraightLine', icon: <ColumnWidthOutlined />, label: 'Vertical Line' },
  { key: 'ray', overlayName: 'rayLine', icon: <ArrowUpOutlined />, label: 'Ray' },
  { key: 'straightLine', overlayName: 'straightLine', icon: <DashOutlined />, label: 'Extended Line' },
  { key: 'parallelLine', overlayName: 'parallelStraightLine', icon: <BorderOuterOutlined />, label: 'Parallel Channel' },
  { key: 'priceLine', overlayName: 'priceLine', icon: <DollarOutlined />, label: 'Price Line' },
  { key: 'priceChannel', overlayName: 'priceChannelLine', icon: <VerticalAlignMiddleOutlined />, label: 'Price Channel' },
  { key: 'fibonacci', overlayName: 'fibonacciLine', icon: <LineChartOutlined />, label: 'Fibonacci Retracement' },
];

interface Props {
  chart: Chart | null;
}

export default function DrawingToolbar({ chart }: Props) {
  const { t } = useTranslation();
  const [activeTool, setActiveTool] = useState<string | null>(null);

  const handleToolClick = (tool: ToolDef) => {
    if (!chart) return;

    // Toggle: clicking the active tool deactivates drawing mode
    if (activeTool === tool.key) {
      setActiveTool(null);
      return;
    }

    // Activate drawing tool
    setActiveTool(tool.key);
    try {
      chart.createOverlay(tool.overlayName as any);
    } catch {
      // klinecharts handles the drawing interaction internally
    }
  };

  const handleClearAll = () => {
    if (!chart) return;
    try {
      (chart as any).removeAllOverlays?.();
    } catch {
      // Fallback: remove one by one
    }
    setActiveTool(null);
  };

  if (!chart) return null;

  return (
    <div style={{
      display: 'flex', flexDirection: 'column', gap: 2,
      padding: '4px 2px', background: 'rgba(24,28,38,0.95)',
      borderRadius: 6, border: '1px solid rgba(255,255,255,0.08)',
      position: 'absolute', top: 8, left: 8, zIndex: 20,
    }}>
      {DRAWING_TOOLS.map((tool) => (
        <Tooltip key={tool.key} title={tool.label} placement="right">
          <div
            onClick={() => handleToolClick(tool)}
            style={{
              width: 28, height: 28, display: 'flex', alignItems: 'center', justifyContent: 'center',
              borderRadius: 4, cursor: 'pointer', fontSize: 14,
              color: activeTool === tool.key ? '#26a69a' : '#8b8f99',
              background: activeTool === tool.key ? 'rgba(38,166,154,0.15)' : 'transparent',
              transition: 'all 0.15s',
            }}
            onMouseEnter={(e) => {
              if (activeTool !== tool.key) {
                e.currentTarget.style.color = '#d1d5db';
                e.currentTarget.style.background = 'rgba(255,255,255,0.06)';
              }
            }}
            onMouseLeave={(e) => {
              if (activeTool !== tool.key) {
                e.currentTarget.style.color = '#8b8f99';
                e.currentTarget.style.background = 'transparent';
              }
            }}
          >
            {tool.icon}
          </div>
        </Tooltip>
      ))}

      {/* Separator */}
      <div style={{ height: 1, margin: '4px 4px', background: 'rgba(255,255,255,0.1)' }} />

      <Tooltip title={t(CLEAR_DRAWINGS_KEY)} placement="right">
        <div
          onClick={handleClearAll}
          style={{
            width: 28, height: 28, display: 'flex', alignItems: 'center', justifyContent: 'center',
            borderRadius: 4, cursor: 'pointer', fontSize: 14,
            color: '#ef5350', background: 'transparent', transition: 'all 0.15s',
          }}
          onMouseEnter={(e) => { e.currentTarget.style.background = 'rgba(239,83,80,0.15)'; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; }}
        >
          <DeleteOutlined />
        </div>
      </Tooltip>
    </div>
  );
}
