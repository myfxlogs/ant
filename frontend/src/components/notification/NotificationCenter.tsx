import React from 'react';
import { BellOutlined } from '@ant-design/icons';

const NotificationCenter: React.FC = () => {
  return (
    <span style={{ cursor: 'pointer', color: '#5A6B75' }}>
      <BellOutlined style={{ fontSize: 18 }} />
    </span>
  );
};

export default NotificationCenter;
