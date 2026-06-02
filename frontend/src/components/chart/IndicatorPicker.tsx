import { Dropdown, Button, Checkbox, Space } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useChartIndicatorsStore } from '@/stores/chartIndicatorsStore';
import type { MenuProps } from 'antd';

interface Props {
  style?: React.CSSProperties;
}

export default function IndicatorPicker({ style }: Props) {
  const { registry, active, addIndicator } = useChartIndicatorsStore();

  const menuItems: MenuProps['items'] = [
    { key: 'overlay-header', label: 'Overlay (main chart)', type: 'group' as const, className: 'indicator-menu-group' },
    ...registry.filter((d) => d.kind === 'overlay').map((d) => ({
      key: d.id,
      label: (
        <Space>
          <span>{d.name}</span>
        </Space>
      ),
      onClick: () => addIndicator(d.id),
    })),
    { key: 'sub-header', label: 'Sub-pane indicators', type: 'group' as const, className: 'indicator-menu-group' },
    ...registry.filter((d) => d.kind === 'sub').map((d) => ({
      key: d.id,
      label: <span>{d.name}</span>,
      onClick: () => addIndicator(d.id),
    })),
  ];

  return (
    <Dropdown menu={{ items: menuItems }} trigger={['click']} placement="bottomRight">
      <Button size="small" icon={<PlusOutlined />} style={style}>
        Indicators
      </Button>
    </Dropdown>
  );
}
