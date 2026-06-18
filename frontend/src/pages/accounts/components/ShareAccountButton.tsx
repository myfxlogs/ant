import { Button, message } from 'antd';
import { ShareAltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

function getAccessToken(): string | null {
  try {
    const raw = localStorage.getItem('auth-storage');
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    return parsed?.state?.accessToken ?? null;
  } catch {
    return null;
  }
}

interface Props {
  accountId: string;
}

export default function ShareAccountButton({ accountId }: Props) {
  const { t } = useTranslation();

  const handleShare = async () => {
    const token = getAccessToken();
    if (!token) {
      message.error(t('common.notAuthenticated', { defaultValue: '请先登录' }));
      return;
    }
    try {
      const resp = await fetch('/ant.v1.ShareService/CreateShareToken', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({ account_id: accountId, expire_days: 7 }),
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.message || 'Failed');
      const url = `${window.location.origin}${data.shareUrl}`;
      await navigator.clipboard.writeText(url);
      message.success(t('accounts.messages.shareLinkCopied', { defaultValue: '分享链接已复制到剪贴板' }));
    } catch {
      message.error(t('accounts.messages.shareLinkFailed', { defaultValue: '创建分享链接失败' }));
    }
  };

  return (
    <Button icon={<ShareAltOutlined />} onClick={handleShare}>
      {t('strategy.share', { defaultValue: '分享' })}
    </Button>
  );
}
