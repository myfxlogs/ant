import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Button, Table, Modal, Input, message, Space, Popconfirm, Typography } from 'antd';
import { KeyOutlined, PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { webauthnApi } from '@/client/webauthn';
import { base64urlToBuffer, bufferToBase64url } from '@/utils/webauthn';

const { Text } = Typography;

export function PasskeyManagement() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [registerName, setRegisterName] = useState('');
  const [registerOpen, setRegisterOpen] = useState(false);
  const [registering, setRegistering] = useState(false);

  const { data: credentials, isLoading } = useQuery({
    queryKey: ['webauthn', 'credentials'],
    queryFn: () => webauthnApi.listCredentials(),
  });

  const removeMutation = useMutation({
    mutationFn: (credentialId: string) => webauthnApi.removeCredential(credentialId),
    onSuccess: () => {
      message.success(t('wallet.passkey.removed', { defaultValue: 'Passkey removed' }));
      queryClient.invalidateQueries({ queryKey: ['webauthn', 'credentials'] });
    },
    onError: (err: Error) => message.error(err.message),
  });

  const handleRegister = async () => {
    setRegistering(true);
    try {
      const rawBytes = await webauthnApi.beginRegistration(registerName);
      const rawStr = new TextDecoder().decode(rawBytes);
      const sepIdx = rawStr.indexOf('|');
      if (sepIdx < 0) throw new Error('Invalid registration options format');
      const sessionHeader = rawStr.slice(0, sepIdx);
      const optionsJson = rawStr.slice(sepIdx + 1);
      const options = JSON.parse(optionsJson);

      const publicKey = {
        ...options.publicKey,
        challenge: base64urlToBuffer(options.publicKey.challenge),
        user: {
          ...options.publicKey.user,
          id: base64urlToBuffer(options.publicKey.user.id),
        },
        excludeCredentials: (options.publicKey.excludeCredentials || []).map((c: any) => ({
          ...c,
          id: base64urlToBuffer(c.id),
        })),
      };

      const credential = await navigator.credentials.create({ publicKey }) as PublicKeyCredential;
      const response = credential.response as AuthenticatorAttestationResponse;

      const attestationObj = new Uint8Array(response.attestationObject);
      const clientDataJSON = new Uint8Array(response.clientDataJSON);

      const responseJson = JSON.stringify({
        id: credential.id,
        rawId: bufferToBase64url(credential.rawId),
        type: credential.type,
        response: {
          attestationObject: bufferToBase64url(attestationObj),
          clientDataJSON: bufferToBase64url(clientDataJSON),
        },
      });

      const finishPayload = new TextEncoder().encode(sessionHeader + '|' + responseJson);
      await webauthnApi.finishRegistration(finishPayload, registerName);
      message.success(t('wallet.passkey.registered', { defaultValue: 'Passkey registered successfully' }));
      setRegisterOpen(false);
      setRegisterName('');
      queryClient.invalidateQueries({ queryKey: ['webauthn', 'credentials'] });
    } catch (err: any) {
      message.error(err.message || t('wallet.passkey.registerFailed', { defaultValue: 'Registration failed' }));
    }
    setRegistering(false);
  };

  const columns = [
    {
      title: t('wallet.passkey.name', { defaultValue: 'Name' }),
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: t('wallet.passkey.credentialId', { defaultValue: 'Credential ID' }),
      dataIndex: 'credentialId',
      key: 'credentialId',
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v?.slice(0, 24)}...</span>,
    },
    {
      title: t('wallet.passkey.signCount', { defaultValue: 'Sign Count' }),
      dataIndex: 'signCount',
      key: 'signCount',
      width: 100,
    },
    {
      title: t('wallet.passkey.createdAt', { defaultValue: 'Created' }),
      dataIndex: 'createdAtTsMs',
      key: 'createdAtTsMs',
      width: 180,
      render: (v: string) => v ? new Date(Number(v)).toLocaleString() : '-',
    },
    {
      title: '',
      key: 'action',
      width: 80,
      render: (_: any, record: any) => (
        <Popconfirm
          title={t('wallet.passkey.confirmRemove', { defaultValue: 'Remove this passkey?' })}
          onConfirm={() => removeMutation.mutate(record.credentialId)}
        >
          <Button type="text" danger icon={<DeleteOutlined />} size="small" />
        </Popconfirm>
      ),
    },
  ];

  return (
    <Card
      size="small"
      style={{ marginBottom: 24 }}
      title={<span><KeyOutlined style={{ marginRight: 8, color: '#D4AF37' }} />{t('wallet.passkey.title', { defaultValue: 'Passkeys (Withdrawal Authorization)' })}</span>}
      extra={<Button type="primary" size="small" icon={<PlusOutlined />} onClick={() => setRegisterOpen(true)}>{t('wallet.passkey.add', { defaultValue: 'Add Passkey' })}</Button>}
    >
      <Table
        columns={columns}
        dataSource={credentials || []}
        rowKey="credentialId"
        loading={isLoading}
        size="small"
        pagination={false}
      />
      <Modal
        title={t('wallet.passkey.register', { defaultValue: 'Register New Passkey' })}
        open={registerOpen}
        onCancel={() => setRegisterOpen(false)}
        onOk={handleRegister}
        confirmLoading={registering}
        okText={t('wallet.passkey.register', { defaultValue: 'Register' })}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Text type="secondary">{t('wallet.passkey.registerHint', { defaultValue: 'Name your passkey for easy identification, then authenticate with your device.' })}</Text>
          <Input
            placeholder={t('wallet.passkey.namePlaceholder', { defaultValue: 'e.g. iPhone Face ID' })}
            value={registerName}
            onChange={(e) => setRegisterName(e.target.value)}
            onPressEnter={handleRegister}
          />
        </Space>
      </Modal>
    </Card>
  );
}
