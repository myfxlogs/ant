import { Button, Select, Tag } from 'antd';
import { CloudServerOutlined, CheckOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import GradientButton, { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import { useTranslation } from 'react-i18next';
import type { BrokerSearchResult, BrokerServer, VerifyResult } from './BindAccount';

export function Step1SearchBroker({
  mtType, setMtType, companySearch, setCompanySearch, searching,
  searchResults, setSearchResults, selectedCompany, selectedServer,
  setSelectedCompany, setSelectedServer, alias, setAlias,
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
        <h2 className="text-xl font-semibold" style={{ color: 'var(--color-text)' }}>{t('accounts.bind.step1.title')}</h2>
        <p className="mt-2" style={{ color: 'var(--color-text-muted)' }}>{t('accounts.bind.step1.subtitle')}</p>
      </div>
      <div>
        <label className="block mb-3 font-medium" style={{ color: 'var(--color-text)' }}>{t('accounts.bind.fields.platform')}</label>
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
        <label className="block mb-3 font-medium" style={{ color: 'var(--color-text)' }}>{t('accounts.bind.fields.brokerName')}</label>
        <div className="flex gap-2">
          <input type="text" value={companySearch} onChange={(e) => setCompanySearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()} placeholder={t('accounts.bind.placeholders.brokerName')}
            className="flex-1 outline-none transition-all"
            style={{ background: 'var(--color-bg-card)', border: '1px solid rgba(185, 201, 223, 0.4)', borderRadius: '10px', padding: '14px 16px', fontSize: '16px', color: 'var(--color-text)', height: '48px' }} />
          <GradientButton onClick={handleSearch} loading={searching} style={{ padding: '0 24px', height: '48px' }}>
            {t('accounts.bind.actions.search')}
          </GradientButton>
        </div>
      </div>
      {searchResults.length > 0 && (<>
        <div>
          <label className="block mb-2 font-medium" style={{ color: 'var(--color-text)' }}>{t('accounts.bind.fields.company')}</label>
          <Select placeholder={t('accounts.bind.placeholders.company')} value={selectedCompany?.companyName}
            onChange={handleCompanyChange} style={{ width: '100%' }} size="large" optionLabelProp="label">
            {searchResults.map((c) => (
              <Select.Option key={c.companyName} value={c.companyName} label={c.companyName}>
                <div className="flex items-center justify-between"><span>{c.companyName}</span><Tag color="blue">{t('accounts.bind.labels.serverCount', { count: c.servers.length })}</Tag></div>
              </Select.Option>
            ))}
          </Select>
        </div>
        {selectedCompany && (
          <div>
            <label className="block mb-2 font-medium" style={{ color: 'var(--color-text)' }}>{t('accounts.bind.fields.server')}</label>
            <Select placeholder={t('accounts.bind.placeholders.server')} value={selectedServer?.name}
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
  mtType, selectedServer, selectedCompany, login, setLogin, password, setPassword, onBack, onNext,
}: {
  mtType: 'MT4' | 'MT5';
  selectedServer: BrokerServer | null;
  selectedCompany: BrokerSearchResult | null;
  login: string;
  setLogin: (v: string) => void;
  password: string;
  setPassword: (v: string) => void;
  onBack: () => void;
  onNext: () => void;
}) {
  const { t } = useTranslation();
  const loginHasNonDigits = login.length > 0 && !/^\d+$/.test(login);
  return (
    <div className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-xl font-semibold" style={{ color: 'var(--color-text)' }}>{t('accounts.bind.step2.title')}</h2>
        <p className="mt-2" style={{ color: 'var(--color-text-muted)' }}>{t('accounts.bind.step2.subtitle')}</p>
      </div>
      <div className="p-4 rounded-xl" style={{ background: 'var(--color-bg-secondary)' }}>
        <div className="flex items-center gap-3">
          <CloudServerOutlined style={{ fontSize: 20, color: '#D4AF37' }} />
          <div><div className="font-medium" style={{ color: 'var(--color-text)' }}>{selectedServer?.name}</div>
            <div className="text-sm" style={{ color: 'var(--color-text-muted)' }}>{selectedCompany?.companyName} · {mtType}</div></div>
        </div>
      </div>
      <div>
        <label className="block mb-2 font-medium" style={{ color: 'var(--color-text)' }}>{t('accounts.bind.fields.tradingAccount')}</label>
        <input type="text" value={login} onChange={(e) => setLogin(e.target.value)}
          placeholder={t('accounts.bind.placeholders.tradingAccount')} className="w-full outline-none transition-all"
          style={{ background: 'var(--color-bg-card)', border: `1px solid ${loginHasNonDigits ? '#E53935' : 'rgba(185, 201, 223, 0.4)'}`, borderRadius: '10px', padding: '14px 16px', fontSize: '16px', color: 'var(--color-text)', height: '48px' }} />
        {loginHasNonDigits && (
          <p className="mt-1 text-xs" style={{ color: '#E53935' }}>{t('accounts.bind.messages.loginDigitsOnly')}</p>
        )}
      </div>
      <div>
        <label className="block mb-2 font-medium" style={{ color: 'var(--color-text)' }}>{t('accounts.bind.fields.password')}</label>
        <input type="password" value={password} onChange={(e) => setPassword(e.target.value)}
          placeholder={t('accounts.bind.placeholders.password')} className="w-full outline-none transition-all"
          style={{ background: 'var(--color-bg-card)', border: '1px solid rgba(185, 201, 223, 0.4)', borderRadius: '10px', padding: '14px 16px', fontSize: '16px', color: 'var(--color-text)', height: '48px' }} />
        <p className="mt-2 text-sm" style={{ color: 'var(--color-text-muted)' }}>{t('accounts.bind.passwordHint')}</p>
      </div>
      <div className="flex justify-between pt-4">
        <Button onClick={onBack} style={{ borderRadius: '10px' }}>{t('common.previous')}</Button>
        <GradientButton disabled={!login.trim() || !password.trim()} onClick={onNext} style={{ padding: '0 32px' }}>{t('common.next')}</GradientButton>
      </div>
    </div>
  );
}

export function Step3Verify({
  mtType, selectedServer, selectedCompany, login, verifying, verifyResult,
  verifyError, loading, setVerifyResult, setVerifyError, handleVerify, handleBind, onBack,
}: {
  mtType: 'MT4' | 'MT5';
  selectedServer: BrokerServer | null;
  selectedCompany: BrokerSearchResult | null;
  login: string;
  verifying: boolean;
  verifyResult: VerifyResult | null;
  verifyError: string;
  loading: boolean;
  setVerifyResult: (v: VerifyResult | null) => void;
  setVerifyError: (v: string) => void;
  handleVerify: () => Promise<void>;
  handleBind: () => Promise<void>;
  onBack: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-xl font-semibold" style={{ color: 'var(--color-text)' }}>{t('accounts.bind.step3.title')}</h2>
        <p className="mt-2" style={{ color: 'var(--color-text-muted)' }}>{t('accounts.bind.step3.subtitle')}</p>
      </div>

      <div className="p-4 rounded-xl" style={{ background: 'var(--color-bg-secondary)' }}>
        <div className="flex items-center gap-3">
          <CloudServerOutlined style={{ fontSize: 20, color: '#D4AF37' }} />
          <div><div className="font-medium" style={{ color: 'var(--color-text)' }}>{selectedServer?.name}</div>
            <div className="text-sm" style={{ color: 'var(--color-text-muted)' }}>{selectedCompany?.companyName} · {mtType} · {login}</div></div>
        </div>
      </div>

      {!verifyResult && (<>
        <div className="flex justify-between pt-4">
          <Button onClick={onBack} style={{ borderRadius: '10px' }}>{t('common.previous')}</Button>
          <GradientButton loading={verifying} onClick={handleVerify} style={{ padding: '0 32px' }}>
            {t('accounts.bind.actions.verifyAccount')}
          </GradientButton>
        </div>
      </>)}

      {verifyError && (
        <div className="p-3 rounded-xl text-center" style={{ background: 'rgba(229, 57, 53, 0.05)', border: '1px solid rgba(229, 57, 53, 0.15)' }}>
          <ExclamationCircleOutlined style={{ fontSize: 16, color: '#E53935' }} />
          <p className="mt-1 text-sm" style={{ color: '#E53935' }}>{verifyError}</p>
          <div className="flex justify-center gap-2 mt-3">
            <Button size="small" onClick={onBack}>{t('common.previous')}</Button>
            <Button size="small" onClick={() => { setVerifyError(''); setVerifyResult(null); }}>{t('accounts.bind.actions.retryVerify')}</Button>
          </div>
        </div>
      )}

      {verifyResult?.verified && (<>
        <div className="p-4 rounded-xl" style={{ background: 'rgba(0, 166, 81, 0.05)', border: '1px solid rgba(0, 166, 81, 0.15)' }}>
          <div className="flex items-center gap-2 mb-3">
            <CheckOutlined style={{ color: '#00A651' }} />
            <span className="font-medium" style={{ color: '#00A651' }}>{t('accounts.bind.summary.verified')}</span>
          </div>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span style={{ color: 'var(--color-text-muted)' }}>{t('accounts.bind.summary.balance')}</span><span className="font-medium" style={{ color: 'var(--color-text)' }}>{Number(verifyResult.balance || 0).toFixed(2)} {verifyResult.currency || ''}</span></div>
            <div className="flex justify-between"><span style={{ color: 'var(--color-text-muted)' }}>{t('accounts.bind.summary.equity')}</span><span className="font-medium" style={{ color: 'var(--color-text)' }}>{Number(verifyResult.equity || 0).toFixed(2)} {verifyResult.currency || ''}</span></div>
            <div className="flex justify-between"><span style={{ color: 'var(--color-text-muted)' }}>{t('accounts.bind.summary.margin')}</span><span className="font-medium" style={{ color: 'var(--color-text)' }}>{Number(verifyResult.margin || 0).toFixed(2)} {verifyResult.currency || ''}</span></div>
            <div className="flex justify-between"><span style={{ color: 'var(--color-text-muted)' }}>{t('accounts.bind.summary.freeMargin')}</span><span className="font-medium" style={{ color: 'var(--color-text)' }}>{Number(verifyResult.freeMargin || 0).toFixed(2)} {verifyResult.currency || ''}</span></div>
            {verifyResult.leverage > 0 && <div className="flex justify-between"><span style={{ color: 'var(--color-text-muted)' }}>{t('accounts.bind.summary.leverage')}</span><span className="font-medium" style={{ color: 'var(--color-text)' }}>1:{verifyResult.leverage}</span></div>}
            <div className="flex justify-between"><span style={{ color: 'var(--color-text-muted)' }}>{t('accounts.bind.summary.currency')}</span><span className="font-medium" style={{ color: 'var(--color-text)' }}>{verifyResult.currency || '-'}</span></div>
          </div>
        </div>
        <div className="flex justify-between pt-4">
          <Button onClick={() => { setVerifyResult(null); setVerifyError(''); onBack(); }} style={{ borderRadius: '10px' }}>{t('common.previous')}</Button>
          <GradientButton loading={loading} onClick={handleBind} style={{ padding: '0 32px' }}>
            {t('accounts.bind.actions.confirmBind')}
          </GradientButton>
        </div>
      </>)}
    </div>
  );
}
