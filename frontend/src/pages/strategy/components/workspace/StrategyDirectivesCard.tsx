import { Tooltip } from 'antd';
import { LockOutlined, EditOutlined } from '@ant-design/icons';
import type { StrategyDirective } from '../../hooks/useBacktestParams';

interface Props {
  directives: StrategyDirective[];
  onJumpToLine?: (key: string) => void;
}

const rowStyle: React.CSSProperties = {
  display: 'flex', alignItems: 'center', justifyContent: 'space-between',
  padding: '4px 8px', fontSize: 11, borderBottom: '1px solid #f0f0f0',
};

const labelStyle: React.CSSProperties = {
  color: '#595959', fontWeight: 500,
};

const valueStyle = (isSet: boolean): React.CSSProperties => ({
  fontFamily: 'monospace', fontSize: 11,
  color: isSet ? '#1677ff' : '#bfbfbf',
  fontWeight: isSet ? 600 : 400,
});

export function StrategyDirectivesCard({ directives, onJumpToLine }: Props) {
  if (!directives.length) return null;

  return (
    <div style={{
      marginTop: 12, padding: '8px 12px',
      border: '1px solid #e6f4ff', borderRadius: 8,
      background: 'linear-gradient(180deg, #f8fbff 0%, #f4f9ff 100%)',
    }}>
      <div style={{
        display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6,
        fontSize: 10, fontWeight: 600, color: '#1677ff',
      }}>
        <LockOutlined />
        <span>Risk Controls from Code</span>
        {onJumpToLine && (
          <Tooltip title="Jump to code">
            <EditOutlined
              style={{ marginLeft: 'auto', cursor: 'pointer', fontSize: 11 }}
              onClick={() => onJumpToLine('first')}
            />
          </Tooltip>
        )}
      </div>
      <div style={{ borderRadius: 4, overflow: 'hidden' }}>
        {directives.map((d) => (
          <div key={d.key} style={rowStyle}
            onClick={() => onJumpToLine?.(d.key)}
            title={onJumpToLine ? `Click to jump to @strategy ${d.key}` : undefined}>
            <span style={labelStyle}>{d.label}</span>
            <span style={valueStyle(d.isSet)}>{d.display}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
