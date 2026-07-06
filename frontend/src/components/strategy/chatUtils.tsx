import { useState, memo } from 'react';
import { Button } from 'antd';
import { CopyOutlined, CodeOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomOneDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';

interface CodeSegment { type: 'text' | 'code'; content: string; lang?: string }

export function parseCodeBlocks(text: string): CodeSegment[] {
  const segments: CodeSegment[] = [];
  const re = /```(\w+)?\n([\s\S]*?)```/g;
  let lastIdx = 0;
  let match: RegExpExecArray | null;
  while ((match = re.exec(text)) !== null) {
    if (match.index > lastIdx) {
      segments.push({ type: 'text', content: text.slice(lastIdx, match.index).trim() });
    }
    segments.push({ type: 'code', lang: match[1] || 'python', content: match[2] });
    lastIdx = re.lastIndex;
  }
  if (lastIdx < text.length) {
    const tail = text.slice(lastIdx).trim();
    if (tail) segments.push({ type: 'text', content: tail });
  }
  return segments;
}

export const CodeBlock = memo(({ code, lang, onApply }: { code: string; lang: string; onApply?: (code: string) => void }) => {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const copy = () => { navigator.clipboard.writeText(code); setCopied(true); setTimeout(() => setCopied(false), 2000); };

  return (
    <div style={{ marginTop: 8, borderRadius: 6, overflow: 'hidden', border: '1px solid var(--ant-color-border)' }}>
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        padding: '4px 10px', background: '#282c34',
      }}>
        <span style={{ fontSize: 11, color: '#abb2bf' }}><CodeOutlined /> {lang}</span>
        <div style={{ display: 'flex', gap: 4 }}>
          <Button size="small" type="text" icon={<CopyOutlined />}
            onClick={copy} style={{ color: '#abb2bf', fontSize: 11 }}>{copied ? '✓' : ''}</Button>
          {onApply && (
            <Button size="small" type="primary" onClick={() => onApply(code)}
              style={{ fontSize: 11 }}>{t('strategy.gen.execApplyCode', 'Apply')}</Button>
          )}
        </div>
      </div>
      <SyntaxHighlighter
        language={lang} style={atomOneDark} showLineNumbers wrapLines
        customStyle={{ margin: 0, fontSize: 12, maxHeight: 300, overflow: 'auto' }}
        lineNumberStyle={{ fontSize: 10, minWidth: '2em', color: '#636d83' }}
      >{code}</SyntaxHighlighter>
    </div>
  );
});

export const StreamContent = memo(({ text, onApply }: { text: string; onApply?: (code: string) => void }) => {
  const segments = parseCodeBlocks(text);
  if (segments.length === 1 && segments[0].type === 'text') {
    return <div style={{ fontSize: 13, lineHeight: '20px', whiteSpace: 'pre-wrap' }}>{segments[0].content}</div>;
  }
  return (
    <div>
      {segments.map((seg, i) => seg.type === 'code'
        ? <CodeBlock key={i} code={seg.content} lang={seg.lang || 'python'} onApply={onApply} />
        : <div key={i} style={{ fontSize: 13, lineHeight: '20px', whiteSpace: 'pre-wrap', marginBottom: 4 }}>{seg.content}</div>
      )}
    </div>
  );
});

export const phaseLabels: Record<string, string> = {
  planning: '📋 Planning strategy...',
  chatting: '💬 Discussing strategy...',
  generating: '⚙️ Generating code...',
  compiling: '🔨 Compiling strategy...',
  backtesting: '🧪 Running backtest...',
  analyzing: '📊 Analyzing results...',
};

export function isNoMarketData(s: string) {
  return /no\s+market\s+data|insufficient\s+data|no\s+available\s+data|symbol\s+not\s+found/i.test(s);
}
