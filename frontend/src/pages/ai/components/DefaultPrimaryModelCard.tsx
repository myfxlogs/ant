import { useEffect, useMemo, useState } from 'react';
import { Card, Select, Button, Space, Typography, message } from 'antd';
import { useTranslation } from 'react-i18next';

import { aiApi } from '@/client/ai';
import type { AIConfig as SystemAIConfig } from '@/pages/ai/systemai/model';

const { Text } = Typography;

// Default Primary Model 卡片：独立组件，自给自足。
// 之所以单独抽出，是为了让 SystemAI / AISettings 都能复用，且不会把
// 主模型的 state 和 Agent 列表 state 搅在一起。
//
// 数据流：
//   挂载 → aiApi.getPrimary() 拉一次
//   保存 → aiApi.setPrimary({providerId, model}) → 回填
// modelOptions 来自父级传入的 systemConfigs（已启用 + has_secret 的行）。
//
// 值的编码与 Agent 选择器对齐："providerId|model"，方便共享 Select options。

export interface DefaultPrimaryModelCardProps {
	systemConfigs: SystemAIConfig[];
	/** provider_id → 本地化显示名（en/zh-cn/...）；调用方注入。 */
	labelOf: (providerId: string, fallbackName?: string) => string;
}

function buildOptions(
	systemConfigs: SystemAIConfig[],
	labelOf: (id: string, name?: string) => string,
) {
	return systemConfigs
		.filter((c) => c && c.provider_id && c.has_secret && c.enabled)
		.flatMap((c) => {
			const models = Array.from(
				new Set((c.models || []).map((m) => (m || '').trim()).filter(Boolean)),
			);
			const list = models.length > 0 ? models : c.default_model ? [c.default_model] : [];
			return list.map((m) => ({
				value: `${c.provider_id}|${m}`,
				label: `${labelOf(c.provider_id, c.name)} · ${m}`,
			}));
		});
}

function decode(value: string): { providerId: string; model: string } {
	if (!value) return { providerId: '', model: '' };
	const idx = value.indexOf('|');
	if (idx < 0) return { providerId: value, model: '' };
	return { providerId: value.slice(0, idx), model: value.slice(idx + 1) };
}

export default function DefaultPrimaryModelCard({
	systemConfigs,
	labelOf,
}: DefaultPrimaryModelCardProps) {
	const { t } = useTranslation();
	const [value, setValue] = useState<string>('');
	const [saving, setSaving] = useState(false);
	const [loaded, setLoaded] = useState(false);

	const options = useMemo(
		() => buildOptions(systemConfigs, labelOf),
		// labelOf 的 i18n 切换只影响显示，value 不变；依赖列表只跟 configs。
		// eslint-disable-next-line react-hooks/exhaustive-deps
		[systemConfigs],
	);

	useEffect(() => {
		let mounted = true;
		(async () => {
			try {
				const r = await aiApi.getPrimary();
				if (!mounted) return;
				setValue(r.providerId ? `${r.providerId}|${r.model || ''}` : '');
			} catch {
				/* 拿不到就当未设置，UI 仍可让用户选择并保存 */
			} finally {
				if (mounted) setLoaded(true);
			}
		})();
		return () => {
			mounted = false;
		};
	}, []);

	const save = async (next: string) => {
		setSaving(true);
		try {
			const dec = decode(next);
			const saved = await aiApi.setPrimary({ providerId: dec.providerId, model: dec.model });
			setValue(saved.providerId ? `${saved.providerId}|${saved.model || ''}` : '');
			message.success(t('common.saveSuccess', { defaultValue: 'Saved' }));
		} catch (e: any) {
			message.error(e?.message || t('common.saveFailed', { defaultValue: 'Save failed' }));
		} finally {
			setSaving(false);
		}
	};

	const empty = options.length === 0;

	return (
		<Card
			className="mb-4"
			title={t('ai.settings.primary.title', { defaultValue: 'Default Primary Model' })}
			loading={!loaded}
		>
			<div style={{ marginBottom: 8 }}>
				<Text type="secondary">
					{t('ai.settings.primary.hint', {
						defaultValue:
							'Used by Clarify Intent, code generation, the strategy template "AI Assistant — modify code" panel, and any Agent that has not picked its own model.',
					})}
				</Text>
			</div>
			<Space wrap>
				<Select
					style={{ minWidth: 320 }}
					allowClear
					showSearch
					optionFilterProp="label"
					value={value || undefined}
					placeholder={
						empty
							? t('ai.settings.agent.fields.modelProfileEmpty', {
									defaultValue: 'No usable model — please configure providers above first.',
								})
							: t('ai.settings.primary.placeholder', {
									defaultValue: 'Pick a provider · model as the default brain',
								})
					}
					options={options}
					notFoundContent={t('ai.settings.agent.fields.modelProfileEmpty', {
						defaultValue: 'No usable model',
					})}
					disabled={saving || empty}
					onChange={(v) => setValue(v || '')}
				/>
				<Button
					type="primary"
					size="small"
					loading={saving}
					disabled={empty}
					onClick={() => save(value)}
				>
					{t('common.save', { defaultValue: 'Save' })}
				</Button>
				{value ? (
					<Button
						size="small"
						disabled={saving}
						onClick={() => {
							setValue('');
							void save('');
						}}
						style={{ color: '#B8960B', borderColor: 'rgba(212, 175, 55, 0.45)' }}
					>
						{t('common.clear', { defaultValue: 'Clear' })}
					</Button>
				) : null}
			</Space>
		</Card>
	);
}
