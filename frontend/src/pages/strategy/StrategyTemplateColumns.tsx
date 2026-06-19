import { TABLE_ACTIONS_KEY, TABLE_NAME_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';
import { ACTIONS_BACKTEST_KEY, ACTIONS_COPY_KEY, ACTIONS_LAUNCH_SCHEDULE_KEY, ACTIONS_VIEW_CODE_KEY, BADGES_PRESET_KEY, DELETE_CONFIRM_KEY, TABLE_CREATED_AT_KEY, TABLE_DESCRIPTION_KEY, TABLE_TAGS_KEY, TABLE_USE_COUNT_KEY, TABLE_VISIBILITY_KEY, VISIBILITY_PRIVATE_KEY, VISIBILITY_PUBLIC_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

import { Button, Space, Tag, Tooltip, Typography, Popconfirm } from 'antd';
import {
	CodeOutlined,
	CopyOutlined,
	DeleteOutlined,
	EditOutlined,
	GlobalOutlined,
	HistoryOutlined,
	ThunderboltOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { TFunction } from 'i18next';
import type { StrategyTemplate } from '@/client/strategy';
import { formatDateTime } from '@/utils/date';

const { Text } = Typography;

export type BuildColumnsParams = {
	t: TFunction;
	onBacktest: (tpl: StrategyTemplate) => void;
	onViewCode: (tpl: StrategyTemplate) => void;
	onCopyToCreate: (tpl: StrategyTemplate) => void;
	onEdit: (tpl: StrategyTemplate) => void;
	onDelete: (id: string) => void;
	onLaunchSchedule: (tpl: StrategyTemplate) => void;
};

export const buildStrategyTemplateColumns = (params: BuildColumnsParams): ColumnsType<StrategyTemplate> => {
	const { t, onBacktest, onViewCode, onCopyToCreate, onEdit, onDelete, onLaunchSchedule } = params;
	return [
		{
			title: t(TABLE_NAME_KEY),
			dataIndex: 'name',
			key: 'name',
			width: 220,
			render: (name: string, record) => {
				// 优先读 isSystem（权威字段，来自 proto）。
				// 兼容后端旧版本：若字段缺失，回落到 tags.preset 判断。
				const tags = Array.isArray(record?.tags) ? record.tags : [];
				const isSystem = Boolean(record?.isSystem) || tags.includes('preset');
				return (
					<Space size={4} wrap>
						<Text strong>{name}</Text>
						{isSystem && (
							<Tag color="gold" style={{ marginInlineEnd: 0 }}>
								{t(BADGES_PRESET_KEY, '预设')}
							</Tag>
						)}
					</Space>
				);
			},
		},
		{
			title: t(TABLE_DESCRIPTION_KEY),
			dataIndex: 'description',
			key: 'description',
			width: 250,
			ellipsis: true,
			render: (desc: string) => (
				<Tooltip title={desc}>
					<Text type="secondary">{desc || '-'}</Text>
				</Tooltip>
			),
		},
		{
			title: t(TABLE_TAGS_KEY, '标签'),
			dataIndex: 'tags',
			key: 'tags',
			width: 220,
			render: (tags: string[] | undefined) => {
				const list = (tags || []).filter((tag) => tag && tag !== 'preset');
				if (list.length === 0) return <Text type="secondary">-</Text>;
				return (
					<Space size={4} wrap>
						{list.slice(0, 4).map((tag) => (
							<Tag key={tag} color="blue" style={{ marginInlineEnd: 0 }}>
								{tag}
							</Tag>
						))}
						{list.length > 4 && <Text type="secondary">+{list.length - 4}</Text>}
					</Space>
				);
			},
		},
		{
			title: t(TABLE_VISIBILITY_KEY),
			dataIndex: 'isPublic',
			key: 'isPublic',
			width: 80,
			render: (isPublic: boolean) =>
				isPublic ? (
					<Tag icon={<GlobalOutlined />} color="blue">{t(VISIBILITY_PUBLIC_KEY)}</Tag>
				) : (
					<Tag>{t(VISIBILITY_PRIVATE_KEY)}</Tag>
				),
		},
		{
			title: t(TABLE_USE_COUNT_KEY),
			dataIndex: 'useCount',
			key: 'useCount',
			width: 100,
			render: (count: number) => <Tag color="green">{count || 0}</Tag>,
		},
		{
			title: t(TABLE_CREATED_AT_KEY),
			dataIndex: 'createdAt',
			key: 'createdAt',
			width: 180,
			render: (date: string) => formatDateTime(date),
		},
		{
			title: t(TABLE_ACTIONS_KEY),
			key: 'action',
			width: 380,
			fixed: 'right',
			render: (_, record) => {
				// 系统模板（含本地的 default-* 占位）不能编辑/删除，仅可回测/查看/复制。
				const tags = Array.isArray(record?.tags) ? record.tags : [];
				const isSystem =
					Boolean(record?.isSystem) ||
					tags.includes('preset') ||
					record.id.startsWith('default-');
				// 本地 default-* 占位模板没有真实后端 id，不能直接用于 createSchedule。
				const isLocalPlaceholder = record.id.startsWith('default-');
				const launchBtn = !isLocalPlaceholder ? (
					<Button
						type="link"
						size="small"
						icon={<ThunderboltOutlined />}
						onClick={() => onLaunchSchedule(record)}
					>
						{t(ACTIONS_LAUNCH_SCHEDULE_KEY, '上线到调度')}
					</Button>
				) : null;
				if (isSystem) {
					return (
						<Space size="small">
							<Button type="link" size="small" icon={<HistoryOutlined />} onClick={() => onBacktest(record)}>
								{t(ACTIONS_BACKTEST_KEY)}
							</Button>
							{launchBtn}
							<Button type="link" size="small" icon={<CodeOutlined />} onClick={() => onViewCode(record)}>
								{t(ACTIONS_VIEW_CODE_KEY)}
							</Button>
							<Button type="link" size="small" icon={<CopyOutlined />} onClick={() => onCopyToCreate(record)}>
								{t(ACTIONS_COPY_KEY)}
							</Button>
						</Space>
					);
				}
				return (
					<Space size="small">
						<Button type="link" size="small" icon={<HistoryOutlined />} onClick={() => onBacktest(record)}>
							{t(ACTIONS_BACKTEST_KEY)}
						</Button>
						{launchBtn}
						<Button type="link" size="small" icon={<CodeOutlined />} onClick={() => onViewCode(record)}>
							{t(ACTIONS_VIEW_CODE_KEY)}
						</Button>
						<Button type="link" size="small" icon={<EditOutlined />} onClick={() => onEdit(record)}>
							{t('common.edit')}
						</Button>
						<Popconfirm title={t(DELETE_CONFIRM_KEY)} onConfirm={() => onDelete(record.id)}>
							<Button type="link" size="small" danger icon={<DeleteOutlined />}>
								{t('common.delete')}
							</Button>
						</Popconfirm>
					</Space>
				);
			},
		},
	];
};
