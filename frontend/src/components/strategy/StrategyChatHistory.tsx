import { Button, Input, Typography, Popconfirm } from 'antd';
import { HistoryOutlined, EditOutlined, CheckOutlined, CloseOutlined, DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  HISTORY_TITLE_KEY, NEW_CONVERSATION_KEY, NO_HISTORY_KEY,
  RENAME_KEY, DELETE_CONFIRM_KEY, DELETE_KEY, CANCEL_KEY,
} from '@/gen/ant/v1/i18n/strategy_ai_chat_keys';
import type { Conversation } from './strategyChatUtils';

interface Props {
  conversations: Conversation[];
  activeConvId: string;
  editingConvId: string | null;
  editTitle: string;
  onNewConv: () => void;
  onLoadConv: (id: string) => void;
  onStartRename: (convId: string, currentTitle: string) => void;
  onConfirmRename: (convId: string) => void;
  onCancelRename: () => void;
  onDeleteConv: (convId: string) => void;
  onEditTitleChange: (value: string) => void;
}

export default function StrategyChatHistory({
  conversations, activeConvId, editingConvId, editTitle,
  onNewConv, onLoadConv, onStartRename, onConfirmRename, onCancelRename, onDeleteConv,
  onEditTitleChange,
}: Props) {
  const { t } = useTranslation();

  return (
    <div style={{ flex: 1, overflowY: 'auto', padding: '12px 14px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
        <Typography.Text style={{ fontSize: 13, fontWeight: 600 }}><HistoryOutlined style={{ marginRight: 6 }} />{t(HISTORY_TITLE_KEY)}</Typography.Text>
        <Button size="small" type="primary" onClick={onNewConv}>+ {t(NEW_CONVERSATION_KEY)}</Button>
      </div>
      {conversations.length > 0 ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {conversations.map(conv => (
            <div key={conv.id}
              style={{ padding: '8px 10px', cursor: editingConvId === conv.id ? 'default' : 'pointer', borderRadius: 8, fontSize: 12,
                background: conv.id === activeConvId ? 'var(--color-bg-elevated)' : 'var(--color-bg-secondary)',
                border: conv.id === activeConvId ? '1px solid var(--color-info)' : '1px solid var(--color-border)',
                display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8, transition: 'all 0.15s' }}>
              {editingConvId === conv.id ? (
                <div style={{ display: 'flex', gap: 4, flex: 1, alignItems: 'center' }}>
                  <Input size="small" value={editTitle}
                    onChange={e => onEditTitleChange(e.target.value)}
                    onPressEnter={() => onConfirmRename(conv.id)}
                    style={{ flex: 1, fontSize: 12 }} autoFocus />
                  <Button size="small" type="text" icon={<CheckOutlined />} onClick={() => onConfirmRename(conv.id)}
                    style={{ color: 'var(--color-success)', padding: '0 4px' }} />
                  <Button size="small" type="text" icon={<CloseOutlined />} onClick={onCancelRename}
                    style={{ color: 'var(--color-danger)', padding: '0 4px' }} />
                </div>
              ) : (
                <>
                  <span onClick={() => onLoadConv(conv.id)}
                    style={{ color: 'var(--color-text)', fontWeight: conv.id === activeConvId ? 600 : 400, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {conv.id === activeConvId && <span style={{ color: 'var(--color-info)', marginRight: 4 }}>●</span>}{conv.title}
                  </span>
                  <span style={{ color: 'var(--color-text-muted)', fontSize: 10, flexShrink: 0 }}>{conv.created_at?.slice(0, 10)}</span>
                  <Button size="small" type="text" icon={<EditOutlined style={{ fontSize: 11 }} />}
                    onClick={(e) => { e.stopPropagation(); onStartRename(conv.id, conv.title); }}
                    style={{ color: 'var(--color-text-muted)', padding: '0 2px', flexShrink: 0 }}
                    title={t(RENAME_KEY)} />
                  <Popconfirm title={t(DELETE_CONFIRM_KEY)} okText={t(DELETE_KEY)} cancelText={t(CANCEL_KEY)}
                    okButtonProps={{ danger: true }}
                    onConfirm={() => onDeleteConv(conv.id)}>
                    <Button size="small" type="text" icon={<DeleteOutlined style={{ fontSize: 11 }} />}
                      onClick={(e) => e.stopPropagation()}
                      style={{ color: 'var(--color-danger)', padding: '0 2px', flexShrink: 0 }}
                      title={t(DELETE_KEY)} />
                  </Popconfirm>
                </>
              )}
            </div>
          ))}
        </div>
      ) : <div style={{ fontSize: 13, color: 'var(--color-text-muted)', textAlign: 'center', padding: '40px 0' }}>{t(NO_HISTORY_KEY)}</div>}
    </div>
  );
}
