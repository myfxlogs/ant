import { memo } from 'react';

export const StreamContent = memo(({ text }: { text: string }) => {
  // §3.1b: code never enters free text — only via write_strategy tool.
  // StreamContent renders the free-text channel (explanations, questions, plans).
  // No code block parsing needed — the single deliverable is rendered via generatedCode.
  return <div style={{ fontSize: 13, lineHeight: '20px', whiteSpace: 'pre-wrap' }}>{text}</div>;
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
