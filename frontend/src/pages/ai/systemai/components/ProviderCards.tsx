import { useTranslation } from 'react-i18next';
import type { AIConfig } from '../model';
import { Section, SoftTag } from './SharedComponents';
import type { ProviderMeta } from '../types';

export function ProviderCardsSection(props: {
  providerCards: AIConfig[];
  selectedProviderId: string;
  providerLabel: (id: string, dbName?: string) => string;
  providerTagline: (id: string) => string;
  metaOf: (providerId: string, fallbackName: string) => ProviderMeta;
  onSelectProvider: (id: string) => void;
  onNewCustomProvider: () => void;
}) {
  const { t } = useTranslation();
  const {
    providerCards,
    selectedProviderId,
    providerLabel,
    providerTagline,
    metaOf,
    onSelectProvider,
    onNewCustomProvider,
  } = props;

  return (
    <Section
      step={1}
      title={t('ai.systemAI.section1.title', { defaultValue: '选择模型厂商' })}
      subtitle={t('ai.systemAI.section1.subtitle', { defaultValue: '卡片直接展示每个厂商的配置与就绪状态，点击选择' })}
    >
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {providerCards.map((cfg) => {
          const isNewCustomCard = cfg.provider_id === '__new_openai_compatible__';
          const m = isNewCustomCard ? metaOf('openai_compatible', '') : metaOf(cfg.provider_id, cfg.name);
          const Icon = m.icon;
          const cfgModelCount = (cfg.models || []).length;
          const ready = cfg.has_secret && cfgModelCount > 0;
          const isSelected = cfg.provider_id === selectedProviderId;
          const stateLabel = !cfg.has_secret
            ? t('ai.systemAI.cardState.noKey', { defaultValue: '未配置' })
            : cfgModelCount === 0
              ? t('ai.systemAI.cardState.noModel', { defaultValue: '待选模型' })
              : cfg.enabled
                ? t('ai.systemAI.cardState.enabled', { defaultValue: '已启用' })
                : t('ai.systemAI.cardState.readyDisabled', { defaultValue: '已就绪 · 未启用' });
          return (
            <button
              key={cfg.provider_id}
              type="button"
              onClick={() => {
                if (isNewCustomCard) {
                  onNewCustomProvider();
                  return;
                }
                if (cfg.provider_id === selectedProviderId) return;
                onSelectProvider(cfg.provider_id);
              }}
              className="text-left rounded-lg border p-3 transition-all hover:shadow-sm"
              style={{
                backgroundColor: isSelected ? 'rgba(212, 175, 55, 0.08)' : '#FFFFFF',
                borderColor: isSelected ? '#D4AF37' : '#E5E7EB',
                borderWidth: isSelected ? 2 : 1,
              }}
            >
              <div className="flex items-center gap-3">
                <div
                  className="w-9 h-9 rounded-md flex items-center justify-center border shrink-0"
                  style={{
                    backgroundColor: 'rgba(212, 175, 55, 0.08)',
                    borderColor: 'rgba(212, 175, 55, 0.35)',
                    color: '#B8960B',
                  }}
                >
                  <Icon className="w-4 h-4 text-[#B8960B]" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-900 truncate">{isNewCustomCard ? providerLabel('openai_compatible') : providerLabel(cfg.provider_id, cfg.name)}</span>
                    {isSelected && !isNewCustomCard && <SoftTag>{t('ai.systemAI.cardTags.current', { defaultValue: '当前' })}</SoftTag>}
                  </div>
                  <div className="text-xs text-gray-500 truncate">{providerTagline(isNewCustomCard ? 'openai_compatible' : cfg.provider_id)}</div>
                </div>
                <SoftTag>{isNewCustomCard ? t('ai.systemAI.cardState.noKey', { defaultValue: '未配置' }) : stateLabel}</SoftTag>
              </div>
              <div className="mt-2 flex items-center gap-1.5 flex-wrap text-xs">
                <SoftTag>
                  {cfg.has_secret
                    ? t('ai.systemAI.cardTags.hasKey', { defaultValue: '已配密钥' })
                    : t('ai.systemAI.cardTags.noKey', { defaultValue: '未配密钥' })}
                </SoftTag>
                <SoftTag>
                  {cfgModelCount > 0
                    ? `${t('ai.settings.fields.availableModels', { defaultValue: '可用模型' })}: ${cfgModelCount}`
                    : t('ai.systemAI.cardTags.noModels', { defaultValue: '未配置可用模型' })}
                </SoftTag>
                {!ready && cfg.enabled && (
                  <SoftTag>{t('ai.systemAI.cardTags.enabledButUnavailable', { defaultValue: '启用但不可用' })}</SoftTag>
                )}
              </div>
            </button>
          );
        })}
      </div>
    </Section>
  );
}
