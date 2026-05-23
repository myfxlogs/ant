import React from 'react';
import { message, Modal } from 'antd';
import { translateMaybeI18nKey } from '@/utils/error';

// 全局配置 message 组件
message.config({
  top: '20%',
  duration: 2,
  maxCount: 1,
});

/**
 * 显示居中的加载提示
 * @param content 加载提示内容
 * @param duration 持续时间（秒），0 表示持久显示
 * @returns 关闭提示的函数
 */
export const showLoading = (content: string, duration: number = 0) => {
  return message.loading({
    content: translateMaybeI18nKey(content, content),
    duration,
    style: {
      textAlign: 'center',
    },
    className: 'centered-message',
  });
};

/**
 * 显示居中的成功提示
 * @param content 成功提示内容
 * @param duration 持续时间（秒）
 */
export const showSuccess = (content: string, duration: number = 2) => {
  return message.success({
    content: translateMaybeI18nKey(content, content),
    duration,
    style: {
      textAlign: 'center',
    },
    className: 'centered-message',
  });
};

/**
 * 显示居中的错误提示
 * @param content 错误提示内容
 * @param duration 持续时间（秒）
 */
export const showError = (content: string, duration: number = 3) => {
  return message.error({
    content: translateMaybeI18nKey(content, content),
    duration,
    style: {
      textAlign: 'center',
    },
    className: 'centered-message',
  });
};

/**
 * 显示居中的警告提示
 * @param content 警告提示内容
 * @param duration 持续时间（秒）
 */
export const showWarning = (content: string, duration: number = 3) => {
  return message.warning({
    content: translateMaybeI18nKey(content, content),
    duration,
    style: {
      textAlign: 'center',
    },
    className: 'centered-message',
  });
};

/**
 * 显示居中的成功提示（Modal 样式）
 * @param title 标题
 * @param content 内容
 */
export const showSuccessModal = (title: string, content?: string) => {
  const modal = Modal.success({
    title: translateMaybeI18nKey(title, title),
    content: content ? translateMaybeI18nKey(content, content) : content,
    centered: true,
    footer: null,
    width: 400,
    className: 'centered-modal',
    icon: (
      <span style={{ color: '#52c41a', fontSize: '24px' }}>
        ✓
      </span>
    ),
  });
  
  // 2 秒后自动关闭
  setTimeout(() => {
    modal.destroy();
  }, 2000);
  
  return modal;
};

/**
 * 显示居中的错误提示（Modal 样式）
 * @param title 标题
 * @param content 内容
 */
export const showErrorModal = (title: string, content?: string) => {
  const modal = Modal.error({
    title: translateMaybeI18nKey(title, title),
    content: content ? translateMaybeI18nKey(content, content) : content,
    centered: true,
    footer: null,
    width: 400,
    className: 'centered-modal',
    icon: (
      <span style={{ color: '#ff4d4f', fontSize: '24px' }}>
        ✕
      </span>
    ),
  });
  
  // 5 秒后自动关闭
  setTimeout(() => {
    modal.destroy();
  }, 5000);
  
  return modal;
};

/**
 * 显示居中的加载提示（Modal 样式）
 * @param title 标题
 * @param content 内容
 */
export const showLoadingModal = (title: string, content?: string) => {
  return Modal.info({
    title: translateMaybeI18nKey(title, title),
    content: content ? translateMaybeI18nKey(content, content) : content,
    centered: true,
    footer: null,
    width: 400,
    className: 'centered-modal',
    icon: (
      <span style={{ fontSize: '24px' }}>
        ⏳
      </span>
    ),
  });
};
