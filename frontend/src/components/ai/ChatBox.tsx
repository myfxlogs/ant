import { useEffect, useMemo, useRef, useState } from 'react';
import { Spin, Empty } from 'antd';
import { LoadingOutlined, RobotOutlined, UserOutlined } from '@ant-design/icons';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import type { Message } from '@/types/ai';
import { getDeviceLocale, getDeviceTimeZone } from '@/utils/date';
import { useTranslation } from 'react-i18next';

interface ChatBoxProps {
  messages: Message[];
  loading?: boolean;
}

function renderMarkdown(content: string): React.ReactNode {
  // Escape ALL HTML first to prevent XSS. Markdown syntax characters
  // (*, _, #, `, -, etc.) are not HTML special characters, so escaping
  // first does not interfere with the regex transformations below.
  let result = escapeHtml(content);

  // Extract code blocks into placeholders so regex replacements below
  // never mutate content inside triple-backtick fences.
  const codeBlocks: string[] = [];
  result = result.replace(/```(\w+)?\n([\s\S]*?)```/g, (_, lang, code) => {
    const idx = codeBlocks.length;
    codeBlocks.push(`<pre class="overflow-x-auto my-2"><code class="language-${lang || ''}">${code.trim()}</code></pre>`);
    return `\x00CODEBLOCK${idx}\x00`;
  });

  // Handle inline code (content already HTML-escaped)
  result = result.replace(/`([^`]+)`/g, '<code class="text-sm">$1</code>');

  // Handle bold (captured text is already escaped)
  result = result.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');

  // Handle italic (captured text is already escaped)
  result = result.replace(/\*([^*]+)\*/g, '<em>$1</em>');

  // Handle headings (captured text is already escaped)
  result = result.replace(/^### (.+)$/gm, '<h3 class="text-base font-semibold mt-3 mb-2">$1</h3>');
  result = result.replace(/^## (.+)$/gm, '<h2 class="text-lg font-semibold mt-3 mb-2">$1</h2>');
  result = result.replace(/^# (.+)$/gm, '<h1 class="text-xl font-bold mt-3 mb-2">$1</h1>');

  // Handle lists (captured text is already escaped)
  result = result.replace(/^- (.+)$/gm, '<li class="ml-4">$1</li>');
  result = result.replace(/^\d+\. (.+)$/gm, '<li class="ml-4 list-decimal">$1</li>');

  // Handle line breaks
  result = result.replace(/\n\n/g, '</p><p class="my-2">');
  result = result.replace(/\n/g, '<br />');

  // Re-insert code block HTML that was protected above.
  result = result.replace(/\x00CODEBLOCK(\d+)\x00/g, (_, idx: string) => codeBlocks[Number(idx)] || '');

  return <div dangerouslySetInnerHTML={{ __html: result }} />;
}

function escapeHtml(text: string): string {
  const map: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;',
  };
  return text.replace(/[&<>"']/g, (m) => map[m]);
}

export default function ChatBox({ messages, loading }: ChatBoxProps) {
  const { t } = useTranslation();
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  const maxCollapsedChars = 1200;
  const maxCollapsedHeightPx = 280;

  const assistantTooLong = useMemo(() => {
    const map: Record<string, boolean> = {};
    for (const m of messages) {
      if (m.role === 'assistant' && (m.content || '').length > maxCollapsedChars) {
        map[m.id] = true;
      }
    }
    return map;
  }, [messages]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const formatTime = (date: Date) => {
    const d = new Date(date);
    const locale = getDeviceLocale();
    const timeZone = getDeviceTimeZone();
    return d.toLocaleTimeString(locale, { timeZone, hour: '2-digit', minute: '2-digit' });
  };

  const ASSISTANT_AVATAR_STYLE = {
    background: PRIMARY_GRADIENT,
  } as const;

  if (messages.length === 0 && !loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <span style={{ color: '#8A9AA5' }}>
              {t('ai.chatBox.emptyDescription')}
            </span>
          }
        />
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto px-4 py-3 space-y-3">
      {messages.map((msg) => (
        <div
          key={msg.id}
          className={`flex gap-3 ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
        >
          {msg.role === 'assistant' && (
            <div
              className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
              style={ASSISTANT_AVATAR_STYLE}
            >
              <RobotOutlined size={18} stroke={1.5} color="#FFFFFF" />
            </div>
          )}

          <div
            className={`${msg.role === 'assistant' ? 'max-w-[60%]' : 'max-w-[70%]'} ${
              msg.role === 'user'
                ? 'bg-gradient-to-r from-amber-500 to-amber-600 text-white'
                : 'bg-white'
            } rounded-2xl px-4 py-3 shadow-sm`}
            style={{
              border: msg.role === 'assistant' ? '1px solid rgba(0, 0, 0, 0.08)' : 'none',
            }}
          >
            {msg.role === 'user' ? (
              <div className="whitespace-pre-wrap break-words">{msg.content}</div>
            ) : (
              <div className="prose prose-sm max-w-none" style={{ color: '#141D22' }}>
                {msg.isLoading && !msg.content ? (
                  <div className="flex items-center gap-2">
                    <LoadingOutlined size={16} stroke={1.5} className="animate-spin" />
                    <span>{t('ai.chatBox.thinking')}</span>
                  </div>
                ) : (
                  <div>
                    <div
                      style={
                        assistantTooLong[msg.id] && !expanded[msg.id]
                          ? {
                              maxHeight: maxCollapsedHeightPx,
                              overflowY: 'auto',
                            }
                          : undefined
                      }
                    >
                      {renderMarkdown(
                        assistantTooLong[msg.id] && !expanded[msg.id]
                          ? msg.content.slice(0, maxCollapsedChars) + `\n\n...(${t('ai.chatBox.truncated')})`
                          : msg.content
                      )}
                    </div>

                    {assistantTooLong[msg.id] && (
                      <div className="mt-2">
                        <button
                          className="text-xs text-gray-500 hover:text-gray-700"
                          onClick={() =>
                            setExpanded((prev) => ({
                              ...prev,
                              [msg.id]: !prev[msg.id],
                            }))
                          }
                        >
                          {expanded[msg.id]
                            ? t('ai.chatBox.collapse')
                            : t('ai.chatBox.expandAll')}
                        </button>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
            <div
              className={`text-xs mt-2 ${
                msg.role === 'user' ? 'text-white/70' : ''
              }`}
              style={msg.role === 'assistant' ? { color: '#8A9AA5' } : {}}
            >
              {formatTime(msg.timestamp)}
            </div>
          </div>

          {msg.role === 'user' && (
            <div
              className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
              style={{ background: '#5A6B75' }}
            >
              <UserOutlined size={18} stroke={1.5} color="#FFFFFF" />
            </div>
          )}
        </div>
      ))}

      {loading && (
        <div className="flex gap-3 justify-start">
          <div
            className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
            style={ASSISTANT_AVATAR_STYLE}
          >
            <RobotOutlined size={18} stroke={1.5} color="#FFFFFF" />
          </div>
          <div
            className="bg-white rounded-2xl px-4 py-3 shadow-sm"
            style={{ border: '1px solid rgba(0, 0, 0, 0.08)' }}
          >
            <Spin size="small" />
          </div>
        </div>
      )}

      <div ref={messagesEndRef} />
    </div>
  );
}
