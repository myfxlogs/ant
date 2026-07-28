import { Button, Input, Select, Tag } from 'antd';
import { CloudServerOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import GradientButton from '@/components/common/GradientButton';
import { useTranslation } from 'react-i18next'
import { BIND_ACTIONS_CONFIRM_BIND_KEY, BIND_ACTIONS_SEARCH_KEY, BIND_FIELDS_BROKER_NAME_KEY, BIND_FIELDS_COMPANY_KEY, BIND_FIELDS_PASSWORD_KEY, BIND_FIELDS_PLATFORM_KEY, BIND_FIELDS_SERVER_KEY, BIND_FIELDS_TRADING_ACCOUNT_KEY, BIND_LABELS_SERVER_COUNT_KEY, BIND_MESSAGES_LOGIN_DIGITS_ONLY_KEY, BIND_PASSWORD_HINT_KEY, BIND_PLACEHOLDERS_BROKER_NAME_KEY, BIND_PLACEHOLDERS_COMPANY_KEY, BIND_PLACEHOLDERS_PASSWORD_KEY, BIND_PLACEHOLDERS_SERVER_KEY, BIND_PLACEHOLDERS_TRADING_ACCOUNT_KEY, BIND_STEP1_SUBTITLE_KEY, BIND_STEP1_TITLE_KEY, BIND_STEP2_SUBTITLE_KEY, BIND_STEP2_TITLE_KEY, BIND_STEP3_SUBTITLE_KEY, BIND_STEP3_TITLE_KEY } from '@/gen/ant/v1/i18n/accounts_keys';
import type { BrokerSearchResult, BrokerServer } from './BindAccount';

export function Step1SearchBroker({
  mtType, setMtType, companySearch, setCompanySearch, searching,
  searchResults, setSearchResults, selectedCompany, selectedServer,
  setSelectedCompany, setSelectedServer, alias: _alias, setAlias: _setAlias,
  handleSearch, handleCompanyChange, handleServerChange, onNext,
}: {
  mtType: 'MT4' | 'MT5';
  setMtType: (v: 'MT4' | 'MT5') => void;
  companySearch: string;
  setCompanySearch: (v: string) => void;
  searching: boolean;
  searchResults: BrokerSearchResult[];
  setSearchResults: (v: BrokerSearchResult[]) => void;
  selectedCompany: BrokerSearchResult | null;
  selectedServer: BrokerServer | null;
  setSelectedCompany: (v: BrokerSearchResult | null) => void;
  setSelectedServer: (v: BrokerServer | null) => void;
  alias: string;
  setAlias: (v: string) => void;
  handleSearch: () => Promise<void>;
  handleCompanyChange: (companyName: string) => void;
  handleServerChange: (serverName: string) => void;
  onNext: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-xl font-semibold" style={{ color: 'var(--color-text)' }}>{t(BIND_STEP1_TITLE_KEY)}</h2>
        <p className="mt-2" style={{ color: 'var(--color-text-muted)' }}>{t(BIND_STEP1_SUBTITLE_KEY)}</p>
      </div>
      <div>
        <label className="block mb-3 font-medium" style={{ color: 'var(--color-text)' }}>{t(BIND_FIELDS_PLATFORM_KEY)}</label>
        <div className="flex gap-4">
          {(['MT4', 'MT5'] as const).map((p) => (
            <div key={p} onClick={() => { setMtType(p); setSearchResults([]); setSelectedCompany(null); setSelectedServer(null); }}
              className="flex-1 p-4 rounded-xl cursor-pointer transition-all"
              style={{ background: mtType === p ? 'rgba(212, 175, 55, 0.1)' : 'var(--color-bg-secondary)', border: `2px solid ${mtType === p ? '#D4AF37' : 'transparent'}` }}>
              <div className="text-center">
                <div className="text-2xl font-bold" style={{ color: mtType === p ? '#D4AF37' : 'var(--color-text)' }}>{p}</div>
                <div className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>MetaTrader {p === 'MT4' ? '4' : '5'}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
      <div>
        <label className="block mb-3 font-medium" style={{ color: 'var(--color-text)' }}>{t(BIND_FIELDS_BROKER_NAME_KEY)}</label>
        <div className="flex gap-2">
          <input type="text" value={companySearch} onChange={(e) => setCompanySearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()} placeholder={t(BIND_PLACEHOLDERS_BROKER_NAME_KEY)}
            className="flex-1 outline-none transition-all"
            style={{ background: 'var(--color-bg-card)', border: '1px solid rgba(185, 201, 223, 0.4)', borderRadius: '10px', padding: '14px 16px', fontSize: '16px', color: 'var(--color-text)', height: '48px' }} />
          <GradientButton onClick={handleSearch} loading={searching} style={{ padding: '0 24px', height: '48px' }}>
            {t(BIND_ACTIONS_SEARCH_KEY)}
          </GradientButton>
        </div>
      </div>
      {searchResults.length > 0 && (<>
        <div>
          <label className="block mb-2 font-medium" style={{ color: 'var(--color-text)' }}>{t(BIND_FIELDS_COMPANY_KEY)}</label>
          <Select placeholder={t(BIND_PLACEHOLDERS_COMPANY_KEY)} value={selectedCompany?.companyName}
            onChange={handleCompanyChange} style={{ width: '100%' }} size="large" optionLabelProp="label">
            {searchResults.map((c) => (
              <Select.Option key={c.companyName} value={c.companyName} label={c.companyName}>
                <div className="flex items-center justify-between"><span>{c.companyName}</span><Tag color="blue">{t(BIND_LABELS_SERVER_COUNT_KEY, { count: c.servers.length })}</Tag></div>
              </Select.Option>
            ))}
          </Select>
        </div>
        {selectedCompany && (
          <div>
            <label className="block mb-2 font-medium" style={{ color: 'var(--color-text)' }}>{t(BIND_FIELDS_SERVER_KEY)}</label>
            <Select placeholder={t(BIND_PLACEHOLDERS_SERVER_KEY)} value={selectedServer?.name}
              onChange={handleServerChange} style={{ width: '100%' }} size="large"
              showSearch
              filterOption={(input, option) =>
                (option?.label as string || '').toLowerCase().includes(input.toLowerCase())
              }>
              {[...selectedCompany.servers].sort((a, b) => a.name.localeCompare(b.name)).map((s) => (
                <Select.Option key={s.name} value={s.name} label={s.name}>
                  <div className="flex items-center justify-between"><span>{s.name}</span><Tag color={mtType === 'MT4' ? 'blue' : 'purple'}>{mtType}</Tag></div>
                </Select.Option>
              ))}
            </Select>
          </div>
        )}
      </>)}
      <div className="flex justify-end pt-4">
        <GradientButton disabled={!selectedServer} onClick={onNext} style={{ padding: '0 32px' }}>{t('common.next')}</GradientButton>
      </div>
    </div>
  );
}

export function Step2Credentials({
  mtType, selectedServer, selectedCompany, login, setLogin, password, setPassword, alias, setAlias, onBack, onNext,
}: {
  mtType: 'MT4' | 'MT5';
  selectedServer: BrokerServer | null;
  selectedCompany: BrokerSearchResult | null;
  login: string;
  setLogin: (v: string) => void;
  password: string;
  setPassword: (v: string) => void;
  alias: string;
  setAlias: (v: string) => void;
  onBack: () => void;
  onNext: () => void;
}) {
  const { t } = useTranslation();
  const loginHasNonDigits = login.length > 0 && !/^\d+$/.test(login);
  return (
    <div className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-xl font-semibold" style={{ color: 'var(--color-text)' }}>{t(BIND_STEP2_TITLE_KEY)}</h2>
        <p className="mt-2" style={{ color: 'var(--color-text-muted)' }}>{t(BIND_STEP2_SUBTITLE_KEY)}</p>
      </div>
      <div className="p-4 rounded-xl" style={{ background: 'var(--color-bg-secondary)' }}>
        <div className="flex items-center gap-3">
          <CloudServerOutlined style={{ fontSize: 20, color: '#D4AF37' }} />
          <div><div className="font-medium" style={{ color: 'var(--color-text)' }}>{selectedServer?.name}</div>
            <div className="text-sm" style={{ color: 'var(--color-text-muted)' }}>{selectedCompany?.companyName} · {mtType}</div></div>
        </div>
      </div>
      <div>
        <label className="block mb-2 font-medium" style={{ color: 'var(--color-text)' }}>{t(BIND_FIELDS_TRADING_ACCOUNT_KEY)}</label>
        <input type="text" value={login} onChange={(e) => setLogin(e.target.value)}
          placeholder={t(BIND_PLACEHOLDERS_TRADING_ACCOUNT_KEY)} className="w-full outline-none transition-all"
          style={{ background: 'var(--color-bg-card)', border: `1px solid ${loginHasNonDigits ? '#E53935' : 'rgba(185, 201, 223, 0.4)'}`, borderRadius: '10px', padding: '14px 16px', fontSize: '16px', color: 'var(--color-text)', height: '48px' }} />
        {loginHasNonDigits && (
          <p className="mt-1 text-xs" style={{ color: '#E53935' }}>{t(BIND_MESSAGES_LOGIN_DIGITS_ONLY_KEY)}</p>
        )}
      </div>
      <div>
        <label className="block mb-2 font-medium" style={{ color: 'var(--color-text)' }}>{t(BIND_FIELDS_PASSWORD_KEY)}</label>
        <Input type="text" value={password} onChange={(e) => setPassword(e.target.value)}
          placeholder={t(BIND_PLACEHOLDERS_PASSWORD_KEY)} className="w-full"
          style={{ background: 'var(--color-bg-card)', border: '1px solid rgba(185, 201, 223, 0.4)', borderRadius: '10px', padding: '14px 16px', fontSize: '16px', color: 'var(--color-text)', height: '48px' }} />
        <p className="mt-2 text-sm" style={{ color: 'var(--color-text-muted)' }}>{t(BIND_PASSWORD_HINT_KEY)}</p>
      </div>
      <div>
        <label className="block mb-2 font-medium" style={{ color: 'var(--color-text)' }}>{t('accounts.bind.fields.alias', { defaultValue: 'Account Alias' })}</label>
        <input type="text" value={alias} onChange={(e) => setAlias(e.target.value)}
          placeholder={t('accounts.bind.placeholders.alias', { defaultValue: 'Optional custom name' })} className="w-full outline-none transition-all"
          style={{ background: 'var(--color-bg-card)', border: '1px solid rgba(185, 201, 223, 0.4)', borderRadius: '10px', padding: '14px 16px', fontSize: '16px', color: 'var(--color-text)', height: '48px' }} />
      </div>
      <div className="flex justify-between pt-4">
        <Button onClick={onBack} style={{ borderRadius: '10px' }}>{t('common.previous')}</Button>
        <GradientButton disabled={!login.trim() || !password.trim()} onClick={onNext} style={{ padding: '0 32px' }}>{t('common.next')}</GradientButton>
      </div>
    </div>
  );
}

export function Step3Bind({
  mtType, selectedServer, selectedCompany, login, loading, bindError, handleBind, onBack,
}: {
  mtType: 'MT4' | 'MT5';
  selectedServer: BrokerServer | null;
  selectedCompany: BrokerSearchResult | null;
  login: string;
  loading: boolean;
  bindError: string;
  handleBind: () => Promise<void>;
  onBack: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-xl font-semibold" style={{ color: 'var(--color-text)' }}>{t(BIND_STEP3_TITLE_KEY)}</h2>
        <p className="mt-2" style={{ color: 'var(--color-text-muted)' }}>{t(BIND_STEP3_SUBTITLE_KEY)}</p>
      </div>

      <div className="p-4 rounded-xl space-y-2" style={{ background: 'var(--color-bg-secondary)' }}>
        <div className="flex items-center gap-3">
          <CloudServerOutlined style={{ fontSize: 20, color: '#D4AF37' }} />
          <div><div className="font-medium" style={{ color: 'var(--color-text)' }}>{selectedServer?.name}</div>
            <div className="text-sm" style={{ color: 'var(--color-text-muted)' }}>{selectedCompany?.companyName} · {mtType} · {login}</div></div>
        </div>
        {selectedServer && selectedServer.access.length > 0 && (
          <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
            {selectedServer.access.join(', ')}
          </div>
        )}
      </div>

      {bindError && (
        <div className="p-3 rounded-xl text-center" style={{ background: 'rgba(229, 57, 53, 0.05)', border: '1px solid rgba(229, 57, 53, 0.15)' }}>
          <ExclamationCircleOutlined style={{ fontSize: 16, color: '#E53935' }} />
          <p className="mt-1 text-sm" style={{ color: '#E53935' }}>{bindError}</p>
          <button onClick={onBack} className="mt-2 text-xs underline" style={{ color: '#E53935' }}>
            {t('accounts.bind.messages.changeCredentials', { defaultValue: 'Change credentials' })}
          </button>
        </div>
      )}

      <div className="flex justify-between pt-4">
        <Button onClick={onBack} style={{ borderRadius: '10px' }}>{t('common.previous')}</Button>
        <GradientButton loading={loading} onClick={handleBind} style={{ padding: '0 32px' }}>
          {t(BIND_ACTIONS_CONFIRM_BIND_KEY)}
        </GradientButton>
      </div>
    </div>
  );
}
