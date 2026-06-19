import { useTranslation } from 'react-i18next'
import { FIELDS_AVAILABLE_MODELS_KEY } from '@/gen/ant/v1/i18n/ai_settings_keys';
import { SYSTEM_A_I_CARD_STATE_ENABLED_KEY, SYSTEM_A_I_CARD_STATE_NO_KEY_KEY, SYSTEM_A_I_CARD_STATE_NO_MODEL_KEY, SYSTEM_A_I_CARD_STATE_READY_DISABLED_KEY, SYSTEM_A_I_CARD_TAGS_CURRENT_KEY, SYSTEM_A_I_CARD_TAGS_ENABLED_BUT_UNAVAILABLE_KEY, SYSTEM_A_I_CARD_TAGS_HAS_KEY_KEY, SYSTEM_A_I_CARD_TAGS_NO_KEY_KEY, SYSTEM_A_I_CARD_TAGS_NO_MODELS_KEY, SYSTEM_A_I_SECTION1_SUBTITLE_KEY, SYSTEM_A_I_SECTION1_TITLE_KEY } from '@/gen/ant/v1/i18n/ai_core_keys';

;
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
      title={t(SYSTEM_A_I_SECTION1_TITLE_KEY, { defaultValue: '选择模型厂商' })}
      subtitle={t(SYSTEM_A_I_SECTION1_SUBTITLE_KEY, { defaultValue: '卡片直接展示每个厂商的配置与就绪状态，点击选择' })}
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
            ? t(SYSTEM_A_I_CARD_STATE_NO_KEY_KEY, { defaultValue: '未配置' })
            : cfgModelCount === 0
              ? t(SYSTEM_A_I_CARD_STATE_NO_MODEL_KEY, { defaultValue: '待选模型' })
              : cfg.enabled
                ? t(SYSTEM_A_I_CARD_STATE_ENABLED_KEY, { defaultValue: '已启用' })
                : t(SYSTEM_A_I_CARD_STATE_READY_DISABLED_KEY, { defaultValue: '已就绪 · 未启用' });
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
                borderColor: isSelected ? '#D4AF37' : 'var(--color-border)',
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
                    {isSelected && !isNewCustomCard && <SoftTag>{t(SYSTEM_A_I_CARD_TAGS_CURRENT_KEY, { defaultValue: '当前' })}</SoftTag>}
                  </div>
                  <div className="text-xs text-gray-500 truncate">{providerTagline(isNewCustomCard ? 'openai_compatible' : cfg.provider_id)}</div>
                </div>
                <SoftTag>{isNewCustomCard ? t(SYSTEM_A_I_CARD_STATE_NO_KEY_KEY, { defaultValue: '未配置' }) : stateLabel}</SoftTag>
              </div>
              <div className="mt-2 flex items-center gap-1.5 flex-wrap text-xs">
                <SoftTag>
                  {cfg.has_secret
                    ? t(SYSTEM_A_I_CARD_TAGS_HAS_KEY_KEY, { defaultValue: '已配密钥' })
                    : t(SYSTEM_A_I_CARD_TAGS_NO_KEY_KEY, { defaultValue: '未配密钥' })}
                </SoftTag>
                <SoftTag>
                  {cfgModelCount > 0
                    ? `${t(FIELDS_AVAILABLE_MODELS_KEY, { defaultValue: '可用模型' })}: ${cfgModelCount}`
                    : t(SYSTEM_A_I_CARD_TAGS_NO_MODELS_KEY, { defaultValue: '未配置可用模型' })}
                </SoftTag>
                {!ready && cfg.enabled && (
                  <SoftTag>{t(SYSTEM_A_I_CARD_TAGS_ENABLED_BUT_UNAVAILABLE_KEY, { defaultValue: '启用但不可用' })}</SoftTag>
                )}
              </div>
            </button>
          );
        })}
      </div>
    </Section>
  );
}
