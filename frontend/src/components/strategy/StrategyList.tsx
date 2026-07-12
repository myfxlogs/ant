import { useState } from 'react';
import { Button, Typography, Popconfirm, Input, Modal } from 'antd';
import { FileTextOutlined, SendOutlined, DeleteOutlined, EditOutlined, CopyOutlined, CodeOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons';
import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';
import python from 'react-syntax-highlighter/dist/esm/languages/hljs/python';
import { atomOneDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import { useTranslation } from 'react-i18next';

SyntaxHighlighter.registerLanguage('python', python);

interface Tpl { id: string; name: string; code: string }

interface Props {
  templates: Tpl[];
  loadedId: string;
  hasCode: boolean;
  onLoad: (id: string) => void;
  onSave: () => void;
  onRename: (id: string, name: string) => void;
  onDelete: (id: string) => void;
  onSendToAI: (code: string, name: string) => void;
}

export default function StrategyList({ templates, loadedId, hasCode, onLoad, onSave, onRename, onDelete, onSendToAI }: Props) {
  const { t } = useTranslation();
  const [editId, setEditId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');
  const [viewCode, setViewCode] = useState<Tpl | null>(null);
  const [copied, setCopied] = useState(false);

  const copy = (text: string) => { navigator.clipboard.writeText(text); setCopied(true); setTimeout(() => setCopied(false), 2000); };

  return (
    <div style={{ flex: 1, overflowY: 'auto', padding: '12px 14px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
        <Typography.Text style={{ fontSize: 13, fontWeight: 600 }}>
          <FileTextOutlined style={{ marginRight: 6 }} />{t('strategy.templates.title', { defaultValue: 'Strategy Templates' })}
        </Typography.Text>
        {hasCode && <Button size="small" type="primary" onClick={onSave}>{t('strategy.templates.saveCurrent', { defaultValue: 'Save Current Strategy' })}</Button>}
      </div>
      {templates.length > 0 ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {templates.map(tpl => (
            <div key={tpl.id}
              style={{ padding: '8px 10px', borderRadius: 8, fontSize: 12,
                background: tpl.id === loadedId ? '#f6ffed' : '#fafafa',
                border: tpl.id === loadedId ? '1px solid #b7eb8f' : '1px solid #f0f0f0',
                transition: 'all 0.15s',
              }}>
              {/* Row 1: name + line count */}
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                {editId === tpl.id ? (
                  <div style={{ display: 'flex', gap: 4, flex: 1, alignItems: 'center' }}>
                    <Input size="small" value={editName}
                      onChange={e => setEditName(e.target.value)}
                      onPressEnter={() => { onRename(tpl.id, editName.trim()); setEditId(null); }}
                      style={{ flex: 1, fontSize: 12 }} autoFocus />
                    <Button size="small" type="text" icon={<CheckOutlined />}
                      onClick={() => { onRename(tpl.id, editName.trim()); setEditId(null); }}
                      style={{ color: '#52c41a', padding: '0 4px' }} />
                    <Button size="small" type="text" icon={<CloseOutlined />}
                      onClick={() => setEditId(null)} style={{ color: '#ff4d4f', padding: '0 4px' }} />
                  </div>
                ) : (
                  <>
                    <span onClick={() => onLoad(tpl.id)}
                      style={{ color: '#262626', fontWeight: tpl.id === loadedId ? 600 : 400, cursor: 'pointer', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {tpl.id === loadedId && <span style={{ color: '#52c41a', marginRight: 4 }}>●</span>}{tpl.name}
                    </span>
                    <span style={{ color: '#8c8c8c', fontSize: 10, flexShrink: 0, marginLeft: 8 }}>
                      {tpl.code ? `${tpl.code.split('\n').length} ${t('strategy.templates.lines', { defaultValue: 'lines' })}` : ''}
                    </span>
                  </>
                )}
              </div>
              {/* Row 2: action buttons */}
              <div style={{ display: 'flex', gap: 4 }}>
                <Button size="small" type="link" icon={<SendOutlined />}
                  onClick={() => onSendToAI(tpl.code, tpl.name)}
                  style={{ fontSize: 10, padding: '0 4px', height: 20 }}>{t('strategy.templates.chatEdit', { defaultValue: 'Chat Edit' })}</Button>
                <Button size="small" type="link" icon={<CodeOutlined />}
                  onClick={() => setViewCode(tpl)}
                  style={{ fontSize: 10, padding: '0 4px', height: 20 }}>{t('strategy.templates.source', { defaultValue: 'Source' })}</Button>
                <Button size="small" type="link" icon={<EditOutlined />}
                  onClick={() => { setEditId(tpl.id); setEditName(tpl.name); }}
                  style={{ fontSize: 10, padding: '0 4px', height: 20 }}>{t('strategy.templates.rename', { defaultValue: 'Rename' })}</Button>
                <Popconfirm title={t('strategy.templates.confirmDelete', { defaultValue: 'Delete this strategy?' })} okText={t('common.delete', { defaultValue: 'Delete' })} cancelText={t('common.cancel', { defaultValue: 'Cancel' })}
                  okButtonProps={{ danger: true }} onConfirm={() => onDelete(tpl.id)}>
                  <Button size="small" type="link" icon={<DeleteOutlined />}
                    style={{ fontSize: 10, padding: '0 4px', height: 20, color: '#ff4d4f' }}>{t('common.delete', { defaultValue: 'Delete' })}</Button>
                </Popconfirm>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div style={{ fontSize: 13, color: '#8c8c8c', textAlign: 'center', padding: '40px 0' }}>
          {t('strategy.templates.noTemplates', { defaultValue: 'No saved strategy templates' })}
        </div>
      )}

      {/* Code viewer modal */}
      <Modal title={viewCode?.name || t('strategy.templates.sourceCode', { defaultValue: 'Strategy Source' })} open={!!viewCode} onCancel={() => setViewCode(null)}
        footer={<Button onClick={() => { copy(viewCode?.code || ''); }} icon={<CopyOutlined />}>{copied ? t('common.copied', { defaultValue: 'Copied' }) : t('strategy.templates.copyAll', { defaultValue: 'Copy All' })}</Button>}
        width={700} style={{ top: 20 }}>
        {viewCode && (
          <SyntaxHighlighter language="python" style={atomOneDark} showLineNumbers wrapLines
            customStyle={{ margin: 0, borderRadius: 6, fontSize: 12, maxHeight: '60vh' }}
            lineNumberStyle={{ fontSize: 10, minWidth: '2em', color: '#636d83' }}>
            {viewCode.code}
          </SyntaxHighlighter>
        )}
      </Modal>
    </div>
  );
}
