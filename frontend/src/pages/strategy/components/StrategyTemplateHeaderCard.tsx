import { CodeOutlined, PlusOutlined, SyncOutlined } from "@ant-design/icons";
import { Button, Card, Col, Row, Space, Typography } from "antd";
import { useTranslation } from "react-i18next";

const { Title } = Typography;

type Props = {
  onRefresh: () => void;
  onCreate: () => void;
};

export default function StrategyTemplateHeaderCard({
  onRefresh,
  onCreate,
}: Props) {
  const { t } = useTranslation();
  return (
    <Card>
      <Row justify="space-between" align="middle">
        <Col>
          <Space>
            <CodeOutlined style={{ fontSize: 24, color: "#1890ff" }} />
            <Title level={4} style={{ margin: 0 }}>
              {t("strategy.templates.title")}
            </Title>
          </Space>
        </Col>
        <Col>
          <Space>
            <Button icon={<SyncOutlined />} onClick={onRefresh}>
              {t("common.refresh")}
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={onCreate}>
              {t("strategy.templates.actions.createTemplate")}
            </Button>
          </Space>
        </Col>
      </Row>
    </Card>
  );
}
