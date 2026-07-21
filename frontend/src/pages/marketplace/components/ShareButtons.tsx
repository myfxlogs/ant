import { useCallback } from 'react';
import { Button, Space, Tooltip, message } from 'antd';
import { ShareAltOutlined, CopyOutlined, TwitterOutlined, MessageOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

interface Props {
  strategyId: string;
  title: string;
}

const BASE_URL = 'https://alfq.org';

export default function ShareButtons({ strategyId, title }: Props) {
  const { t } = useTranslation();

  const shareUrl = `${BASE_URL}/strategy/${strategyId}`;
  const shareText = encodeURIComponent(`${title} — AlphaForge Strategy Marketplace`);

  const copyLink = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(shareUrl);
      message.success(t('marketplace.share.linkCopied', { defaultValue: 'Link copied to clipboard' }));
    } catch {
      message.error(t('marketplace.share.copyFailed', { defaultValue: 'Failed to copy link' }));
    }
  }, [shareUrl, t]);

  const twitterUrl = `https://twitter.com/intent/tweet?text=${shareText}&url=${encodeURIComponent(shareUrl)}`;
  const telegramUrl = `https://t.me/share/url?url=${encodeURIComponent(shareUrl)}&text=${shareText}`;

  return (
    <Space size="small">
      <Tooltip title={t('marketplace.share.copyLink', { defaultValue: 'Copy Link' })}>
        <Button type="text" size="small" icon={<CopyOutlined />} onClick={copyLink} />
      </Tooltip>
      <Tooltip title="Twitter">
        <Button type="text" size="small" icon={<TwitterOutlined />} href={twitterUrl} target="_blank" rel="noopener noreferrer" />
      </Tooltip>
      <Tooltip title="Telegram">
        <Button type="text" size="small" icon={<MessageOutlined />} href={telegramUrl} target="_blank" rel="noopener noreferrer" />
      </Tooltip>
      <Tooltip title={t('marketplace.share.wechat', { defaultValue: 'WeChat (copy link)' })}>
        <Button type="text" size="small" icon={<ShareAltOutlined />} onClick={copyLink} />
      </Tooltip>
    </Space>
  );
}
