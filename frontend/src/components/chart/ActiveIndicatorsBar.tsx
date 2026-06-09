import { Tag, Tooltip, Button, Space } from 'antd';
import { EyeOutlined, EyeInvisibleOutlined, SettingOutlined, CloseOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useChartIndicatorsStore } from '@/stores/chartIndicatorsStore';
import { useState } from 'react';
import IndicatorSettingsModal from './IndicatorSettingsModal';

interface Props {
  className?: string;
  style?: React.CSSProperties;
}

export default function ActiveIndicatorsBar({ className, style }: Props) {
  const { t } = useTranslation();
  const { active, removeIndicator, toggleVisibility, getDef } = useChartIndicatorsStore();
  const [editingId, setEditingId] = useState<string | null>(null);

  if (active.length === 0) return null;

  const editing = editingId ? active.find((a) => a.instanceId === editingId) : null;
  const editingDef = editing ? getDef(editing.defId) : undefined;

  return (
    <>
      <div className={className} style={{ display: 'flex', flexWrap: 'wrap', gap: 4, alignItems: 'center', ...style }}>
        {active.map((ind) => {
          const def = getDef(ind.defId);
          return (
            <Tag
              key={ind.instanceId}
              color={ind.visible ? 'processing' : 'default'}
              style={{ margin: 0, cursor: 'pointer', opacity: ind.visible ? 1 : 0.5 }}
            >
              <Space size={2}>
                <span
                  style={{ fontSize: 12 }}
                  onClick={() => setEditingId(ind.instanceId)}
                >
                  {def?.name || ind.defId}
                  {ind.defId === 'SMA' || ind.defId === 'EMA' ? `(${ind.params.length})` : ''}
                  {ind.defId === 'BOLL' ? `(${ind.params.length},${ind.params.mult})` : ''}
                  {ind.defId === 'RSI' || ind.defId === 'ATR' || ind.defId === 'CCI' ? `(${ind.params.length})` : ''}
                  {ind.defId === 'MACD' ? `(${ind.params.fast},${ind.params.slow},${ind.params.signal})` : ''}
                </span>
                <Tooltip title={ind.visible ? t('strategy.chartTools.hide') : t('strategy.chartTools.show')} mouseEnterDelay={0.5}>
                  <Button
                    type="text" size="small"
                    icon={ind.visible ? <EyeOutlined /> : <EyeInvisibleOutlined />}
                    onClick={(e) => { e.stopPropagation(); toggleVisibility(ind.instanceId); }}
                    style={{ padding: 0, minWidth: 14, height: 14, lineHeight: 1 }}
                  />
                </Tooltip>
                <Tooltip title={t('strategy.chartTools.settings')} mouseEnterDelay={0.5}>
                  <Button
                    type="text" size="small"
                    icon={<SettingOutlined />}
                    onClick={(e) => { e.stopPropagation(); setEditingId(ind.instanceId); }}
                    style={{ padding: 0, minWidth: 14, height: 14, lineHeight: 1 }}
                  />
                </Tooltip>
                <Tooltip title={t('strategy.chartTools.remove')} mouseEnterDelay={0.5}>
                  <Button
                    type="text" size="small" danger
                    icon={<CloseOutlined />}
                    onClick={(e) => { e.stopPropagation(); removeIndicator(ind.instanceId); }}
                    style={{ padding: 0, minWidth: 14, height: 14, lineHeight: 1 }}
                  />
                </Tooltip>
              </Space>
            </Tag>
          );
        })}
      </div>
      {editing && editingDef && (
        <IndicatorSettingsModal
          visible={true}
          indicator={editing}
          def={editingDef}
          onClose={() => setEditingId(null)}
        />
      )}
    </>
  );
}
