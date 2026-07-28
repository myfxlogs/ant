import React from 'react';
import { Modal, Typography, Space } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';

const { Text: _Text } = Typography;

interface TradeConfirmModalProps {
  open: boolean;
  title: string;
  content: React.ReactNode;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const TradeConfirmModal: React.FC<TradeConfirmModalProps> = ({
  open, title, content, confirmText, cancelText, danger, loading, onConfirm, onCancel,
}) => {
  return (
    <Modal
      open={open}
      title={
        <Space>
          <ExclamationCircleOutlined style={{ color: danger ? '#ff4d4f' : '#faad14' }} />
          <span>{title}</span>
        </Space>
      }
      onOk={onConfirm}
      onCancel={onCancel}
      okText={confirmText ?? 'Confirm'}
      cancelText={cancelText ?? 'Cancel'}
      okButtonProps={{ danger, loading }}
      width={480}
      destroyOnClose
    >
      {content}
    </Modal>
  );
};
