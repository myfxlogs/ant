import { Input, Switch, Row, Col, Form, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import { EDIT_TEMPLATE_MODAL_FIELDS_DESCRIPTION_KEY, EDIT_TEMPLATE_MODAL_FIELDS_NAME_KEY, EDIT_TEMPLATE_MODAL_FIELDS_PUBLIC_SHARE_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_DESCRIPTION_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_NAME_KEY, EDIT_TEMPLATE_MODAL_VALIDATION_NAME_REQUIRED_KEY, VISIBILITY_PRIVATE_KEY, VISIBILITY_PUBLIC_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

const { TextArea } = Input;
const { Text } = Typography;

export default function MetadataHeader() {
  const { t } = useTranslation();
  return (
    <div style={{ background: 'var(--color-bg-secondary)', borderRadius: 10, padding: '12px 16px', marginBottom: 14 }}>
      <Row gutter={16} align="bottom">
        <Col span={7}>
          <Form.Item name="name" label={<Text strong>{t(EDIT_TEMPLATE_MODAL_FIELDS_NAME_KEY)}</Text>}
            rules={[{ required: true, message: t(EDIT_TEMPLATE_MODAL_VALIDATION_NAME_REQUIRED_KEY) }]}
            style={{ marginBottom: 0 }}>
            <Input placeholder={t(EDIT_TEMPLATE_MODAL_PLACEHOLDERS_NAME_KEY)} />
          </Form.Item>
        </Col>
        <Col span={11}>
          <Form.Item name="description" label={<Text strong>{t(EDIT_TEMPLATE_MODAL_FIELDS_DESCRIPTION_KEY)}</Text>} style={{ marginBottom: 0 }}>
            <TextArea rows={1} placeholder={t(EDIT_TEMPLATE_MODAL_PLACEHOLDERS_DESCRIPTION_KEY)} autoSize={{ minRows: 1, maxRows: 2 }} />
          </Form.Item>
        </Col>
        <Col span={6}>
          <Form.Item name="isPublic" label={<Text strong>{t(EDIT_TEMPLATE_MODAL_FIELDS_PUBLIC_SHARE_KEY)}</Text>} valuePropName="checked" style={{ marginBottom: 0 }}>
            <Switch checkedChildren={t(VISIBILITY_PUBLIC_KEY)} unCheckedChildren={t(VISIBILITY_PRIVATE_KEY)} />
          </Form.Item>
        </Col>
      </Row>
    </div>
  );
}
