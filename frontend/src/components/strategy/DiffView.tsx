import { useState } from 'react';
import { Button, Typography } from 'antd';
import { DiffOutlined, CloseOutlined } from '@ant-design/icons';

interface Props {
  oldCode: string;
  newCode: string;
  onApply?: (code: string) => void;
  onDismiss?: () => void;
  language?: string;
}

interface DiffLine {
  type: 'same' | 'add' | 'del';
  numA: number;
  numB: number;
  text: string;
}

/** Simple line-by-line diff (Myers-like greedy algorithm). */
function computeDiff(oldLines: string[], newLines: string[]): DiffLine[] {
  const result: DiffLine[] = [];
  let oi = 0, ni = 0;

  while (oi < oldLines.length || ni < newLines.length) {
    // Skip matching lines
    while (oi < oldLines.length && ni < newLines.length && oldLines[oi] === newLines[ni]) {
      result.push({ type: 'same', numA: oi + 1, numB: ni + 1, text: oldLines[oi] });
      oi++; ni++;
    }

    if (oi >= oldLines.length) {
      // Remaining new lines are additions
      while (ni < newLines.length) {
        result.push({ type: 'add', numA: 0, numB: ni + 1, text: newLines[ni] });
        ni++;
      }
      break;
    }
    if (ni >= newLines.length) {
      while (oi < oldLines.length) {
        result.push({ type: 'del', numA: oi + 1, numB: 0, text: oldLines[oi] });
        oi++;
      }
      break;
    }

    // Look ahead for the next match
    let matchFound = false;
    for (let lookAhead = 1; lookAhead < 6 && oi + lookAhead < oldLines.length; lookAhead++) {
      const idx = newLines.indexOf(oldLines[oi + lookAhead], ni);
      if (idx >= 0 && idx - ni < 6) {
        // Delete old lines up to lookAhead, add new lines up to idx
        for (let d = 0; d < lookAhead; d++) {
          result.push({ type: 'del', numA: oi + 1 + d, numB: 0, text: oldLines[oi + d] });
        }
        for (let a = ni; a < idx; a++) {
          result.push({ type: 'add', numA: 0, numB: a + 1, text: newLines[a] });
        }
        oi += lookAhead;
        ni = idx;
        matchFound = true;
        break;
      }
    }

    if (!matchFound) {
      result.push({ type: 'del', numA: oi + 1, numB: 0, text: oldLines[oi] });
      result.push({ type: 'add', numA: 0, numB: ni + 1, text: newLines[ni] });
      oi++; ni++;
    }
  }
  return result;
}

export default function DiffView({ oldCode, newCode, onApply, onDismiss }: Props) {
  const [collapsed, setCollapsed] = useState(false);

  if (!oldCode || oldCode === newCode) return null;

  const oldLines = oldCode.split('\n');
  const newLines = newCode.split('\n');
  const diff = computeDiff(oldLines, newLines);
  const added = diff.filter(d => d.type === 'add').length;
  const deleted = diff.filter(d => d.type === 'del').length;

  return (
    <div style={{ margin: '6px 0', borderRadius: 6, border: '1px solid #e8e8e8', overflow: 'hidden' }}>
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '4px 10px', background: '#fafafa', borderBottom: collapsed ? 'none' : '1px solid #f0f0f0',
      }}>
        <Typography.Text style={{ fontSize: 12 }}>
          <DiffOutlined style={{ marginRight: 4, color: '#1677ff' }} />
          Changes: <span style={{ color: '#52c41a' }}>+{added}</span>
          {' / '}
          <span style={{ color: '#ff4d4f' }}>-{deleted}</span>
          {' lines'}
        </Typography.Text>
        <div>
          <Button size="small" type="link" onClick={() => setCollapsed(!collapsed)} style={{ fontSize: 11 }}>
            {collapsed ? 'Show' : 'Hide'}
          </Button>
          {onApply && (
            <Button size="small" type="primary" onClick={() => onApply(newCode)} style={{ marginLeft: 4 }}>
              Apply
            </Button>
          )}
          {onDismiss && (
            <Button size="small" type="text" icon={<CloseOutlined />} onClick={onDismiss} />
          )}
        </div>
      </div>

      {!collapsed && (
        <div style={{
          maxHeight: 300, overflow: 'auto', fontSize: 11, fontFamily: 'monospace',
          lineHeight: '18px', background: '#fff',
        }}>
          {diff.map((line, i) => (
            <div key={i} style={{
              padding: '0 8px',
              background: line.type === 'add' ? '#e6ffec' : line.type === 'del' ? '#ffebe9' : 'transparent',
              borderLeft: line.type === 'add' ? '3px solid #52c41a' : line.type === 'del' ? '3px solid #ff4d4f' : '3px solid transparent',
            }}>
              <span style={{ color: '#bbb', width: 36, display: 'inline-block', textAlign: 'right', marginRight: 8, userSelect: 'none' }}>
                {line.numA || ' '}
              </span>
              <span style={{ color: '#bbb', width: 36, display: 'inline-block', textAlign: 'right', marginRight: 8, userSelect: 'none' }}>
                {line.numB || ' '}
              </span>
              <span style={{
                color: line.type === 'add' ? '#1a7f37' : line.type === 'del' ? '#cf222e' : '#24292f',
              }}>
                {line.type === 'add' ? '+' : line.type === 'del' ? '-' : ' '}{line.text}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
