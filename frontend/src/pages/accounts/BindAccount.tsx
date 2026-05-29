import { useState } from 'react';
import { Button, Select, Tag } from 'antd';
import { showSuccess, showError, showWarning, showInfo } from '@/utils/message';
import { ArrowLeftOutlined, CloudServerOutlined, CheckOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useContext } from 'react';
import GradientButton, { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import { useAccount } from '@/hooks/useAccount';
import { accountApi } from '@/client/account';
import { getErrorMessage } from '@/utils/error';
import type { BindAccountRequest } from '@/types/account';
import type { Account } from '@/types/account';
import { useTranslation } from 'react-i18next';
import { ConnectContext } from '@/providers/connectContext';

interface BrokerServer {
  name: string;
  access: string[];
}

interface BrokerSearchResult {
  companyName: string;
  servers: BrokerServer[];
}

export default function BindAccount() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [verifyResult, setVerifyResult] = useState<any>(null);
  const [verifyError, setVerifyError] = useState('');
  const [step, setStep] = useState(1);
  const navigate = useNavigate();
  const { bindAccount } = useAccount();
  const connectCtx = useContext(ConnectContext);

  const [mtType, setMtType] = useState<'MT4' | 'MT5'>('MT4');
  const [companySearch, setCompanySearch] = useState('');
  const [searchResults, setSearchResults] = useState<BrokerSearchResult[]>([]);
  const [selectedCompany, setSelectedCompany] = useState<BrokerSearchResult | null>(null);
  const [selectedServer, setSelectedServer] = useState<BrokerServer | null>(null);
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [alias, setAlias] = useState('');

  const handleSearch = async () => {
    if (!companySearch.trim()) { showWarning(t('accounts.bind.messages.enterBrokerName')); return; }
    setSearching(true); setSearchResults([]); setSelectedCompany(null); setSelectedServer(null);
    try {
      const companies = await accountApi.searchBroker(companySearch.trim(), mtType);
      if (companies && companies.length > 0) {
        const results = companies.map((c: any) => ({
          companyName: c.companyName || c.company_name,
          servers: (c.servers || []).map((s: any) => ({ name: s.name, access: s.access })),
        }));
        setSearchResults(results);
        showSuccess(t('accounts.bind.messages.foundBrokers', { count: results.length }));
      } else {
        showInfo(t('accounts.bind.messages.noBrokersFound'));
      }
    } catch { showError(t('accounts.bind.messages.searchFailed')); }
    finally { setSearching(false); }
  };

  const handleCompanyChange = (companyName: string) => {
    setSelectedCompany(searchResults.find(c => c.companyName === companyName) || null);
    setSelectedServer(null);
  };

  const handleServerChange = (serverName: string) => {
    const server = selectedCompany?.servers.find(s => s.name === serverName);
    if (server) { setSelectedServer(server); if (!alias) setAlias(server.name); }
  };

  const handleVerify = async () => {
    if (!selectedCompany || !selectedServer) { showWarning(t('accounts.bind.messages.selectServer')); return; }
    if (!login.trim()) { showWarning(t('accounts.bind.messages.enterTradingAccount')); return; }
    if (!password.trim()) { showWarning(t('accounts.bind.messages.enterPassword')); return; }
    setVerifying(true); setVerifyError(''); setVerifyResult(null);
    try {
      const host = selectedServer.access[0] || '';
      const result = await accountApi.verifyAccount({ login: login.trim(), password, mtType, brokerHost: host });
      setVerifyResult(result);
      if (!result.verified) setVerifyError(result.message || t('accounts.bind.messages.verifyFailed'));
    } catch (error) {
      setVerifyError(getErrorMessage(error, t('accounts.bind.messages.verifyFailed')));
    } finally { setVerifying(false); }
  };

  const handleBind = async () => {
    if (!selectedCompany || !selectedServer) return;
    setLoading(true);
    try {
      const host = selectedServer.access[0] || '';
      const request: BindAccountRequest = {
        alias: alias || selectedServer.name, mtType,
        login: login.trim(), password,
        brokerCompany: selectedCompany.companyName,
        brokerServer: selectedServer.name, brokerHost: host,
      };
      const account = await bindAccount(request) as Account;
      setPassword('');
      // Start the live stream connection for the newly bound account.
      await accountApi.connect(account.id);
      connectCtx?.reconnect();
      showSuccess(t('accounts.bind.messages.bindSuccess'));
      navigate(`/accounts/${account.id}`);
    } catch (error) {
      showError(getErrorMessage(error, t('accounts.bind.messages.bindFailed')));
    } finally { setLoading(false); }
  };

  const renderStepIndicator = () => (
    <div className="flex items-center justify-center gap-4 mb-8">
      {[1, 2, 3].map((s) => (
        <div key={s} className="flex items-center">
          <div className="w-8 h-8 rounded-full flex items-center justify-center font-medium"
            style={{ background: step >= s ? PRIMARY_GRADIENT : '#E8ECF0', color: step >= s ? '#FFFFFF' : '#8A9AA5' }}>
            {step > s ? <CheckOutlined style={{ fontSize: 16 }} /> : s}
          </div>
          {s < 3 && <div className="w-16 h-0.5 mx-2" style={{ background: step > s ? '#D4AF37' : '#E8ECF0' }} />}
        </div>
      ))}
    </div>
  );

  const renderStep1 = () => (
    <div className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-xl font-semibold" style={{ color: '#141D22' }}>{t('accounts.bind.step1.title')}</h2>
        <p className="mt-2" style={{ color: '#8A9AA5' }}>{t('accounts.bind.step1.subtitle')}</p>
      </div>
      <div>
        <label className="block mb-3 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.platform')}</label>
        <div className="flex gap-4">
          {(['MT4', 'MT5'] as const).map((p) => (
            <div key={p} onClick={() => { setMtType(p); setSearchResults([]); setSelectedCompany(null); setSelectedServer(null); }}
              className="flex-1 p-4 rounded-xl cursor-pointer transition-all"
              style={{ background: mtType === p ? 'rgba(212, 175, 55, 0.1)' : '#F5F7F9', border: `2px solid ${mtType === p ? '#D4AF37' : 'transparent'}` }}>
              <div className="text-center">
                <div className="text-2xl font-bold" style={{ color: mtType === p ? '#D4AF37' : '#141D22' }}>{p}</div>
                <div className="text-sm mt-1" style={{ color: '#8A9AA5' }}>MetaTrader {p === 'MT4' ? '4' : '5'}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
      <div>
        <label className="block mb-3 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.brokerName')}</label>
        <div className="flex gap-2">
          <input type="text" value={companySearch} onChange={(e) => setCompanySearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()} placeholder={t('accounts.bind.placeholders.brokerName')}
            className="flex-1 outline-none transition-all"
            style={{ background: '#FFFFFF', border: '1px solid rgba(185, 201, 223, 0.4)', borderRadius: '10px', padding: '14px 16px', fontSize: '16px', color: '#141D22', height: '48px' }} />
          <GradientButton onClick={handleSearch} loading={searching} style={{ padding: '0 24px', height: '48px' }}>
            {t('accounts.bind.actions.search')}
          </GradientButton>
        </div>
      </div>
      {searchResults.length > 0 && (<>
        <div>
          <label className="block mb-2 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.company')}</label>
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
            <label className="block mb-2 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.server')}</label>
            <Select placeholder={t('accounts.bind.placeholders.server')} value={selectedServer?.name}
              onChange={handleServerChange} style={{ width: '100%' }} size="large" optionLabelProp="label">
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
        <GradientButton disabled={!selectedServer} onClick={() => setStep(2)} style={{ padding: '0 32px' }}>{t('common.next')}</GradientButton>
      </div>
    </div>
  );

  const renderStep2 = () => (
    <div className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-xl font-semibold" style={{ color: '#141D22' }}>{t('accounts.bind.step2.title')}</h2>
        <p className="mt-2" style={{ color: '#8A9AA5' }}>{t('accounts.bind.step2.subtitle')}</p>
      </div>
      <div className="p-4 rounded-xl" style={{ background: '#F5F7F9' }}>
        <div className="flex items-center gap-3">
          <CloudServerOutlined style={{ fontSize: 20, color: '#D4AF37' }} />
          <div><div className="font-medium" style={{ color: '#141D22' }}>{selectedServer?.name}</div>
            <div className="text-sm" style={{ color: '#8A9AA5' }}>{selectedCompany?.companyName} · {mtType}</div></div>
        </div>
      </div>
      <div>
        <label className="block mb-2 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.tradingAccount')}</label>
        <input type="text" value={login} onChange={(e) => setLogin(e.target.value)}
          placeholder={t('accounts.bind.placeholders.tradingAccount')} className="w-full outline-none transition-all"
          style={{ background: '#FFFFFF', border: '1px solid rgba(185, 201, 223, 0.4)', borderRadius: '10px', padding: '14px 16px', fontSize: '16px', color: '#141D22', height: '48px' }} />
      </div>
      <div>
        <label className="block mb-2 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.password')}</label>
        <input type="text" value={password} onChange={(e) => setPassword(e.target.value)}
          placeholder={t('accounts.bind.placeholders.password')} className="w-full outline-none transition-all"
          style={{ background: '#FFFFFF', border: '1px solid rgba(185, 201, 223, 0.4)', borderRadius: '10px', padding: '14px 16px', fontSize: '16px', color: '#141D22', height: '48px' }} />
        <p className="mt-2 text-sm" style={{ color: '#8A9AA5' }}>{t('accounts.bind.passwordHint')}</p>
      </div>
      <div className="flex justify-between pt-4">
        <Button onClick={() => setStep(1)} style={{ borderRadius: '10px' }}>{t('common.previous')}</Button>
        <GradientButton disabled={!login.trim() || !password.trim()} onClick={() => setStep(3)} style={{ padding: '0 32px' }}>{t('common.next')}</GradientButton>
      </div>
    </div>
  );

  const renderStep3 = () => (
    <div className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-xl font-semibold" style={{ color: '#141D22' }}>{t('accounts.bind.step3.title')}</h2>
        <p className="mt-2" style={{ color: '#8A9AA5' }}>{t('accounts.bind.step3.subtitle')}</p>
      </div>

      <div className="p-4 rounded-xl" style={{ background: '#F5F7F9' }}>
        <div className="flex items-center gap-3">
          <CloudServerOutlined style={{ fontSize: 20, color: '#D4AF37' }} />
          <div><div className="font-medium" style={{ color: '#141D22' }}>{selectedServer?.name}</div>
            <div className="text-sm" style={{ color: '#8A9AA5' }}>{selectedCompany?.companyName} · {mtType} · {login}</div></div>
        </div>
      </div>

      {!verifyResult && (
        <GradientButton loading={verifying} onClick={handleVerify} block style={{ padding: '12px 0' }}>
          {t('accounts.bind.actions.verifyAccount')}
        </GradientButton>
      )}

      {verifyError && (
        <div className="p-3 rounded-xl text-center" style={{ background: 'rgba(229, 57, 53, 0.05)', border: '1px solid rgba(229, 57, 53, 0.15)' }}>
          <ExclamationCircleOutlined style={{ fontSize: 16, color: '#E53935' }} />
          <p className="mt-1 text-sm" style={{ color: '#E53935' }}>{verifyError}</p>
          <Button size="small" onClick={() => { setVerifyError(''); setVerifyResult(null); }} style={{ marginTop: 8 }}>
            {t('accounts.bind.actions.retryVerify')}
          </Button>
        </div>
      )}

      {verifyResult?.verified && (<>
        <div className="p-4 rounded-xl" style={{ background: 'rgba(0, 166, 81, 0.05)', border: '1px solid rgba(0, 166, 81, 0.15)' }}>
          <div className="flex items-center gap-2 mb-3">
            <CheckOutlined style={{ color: '#00A651' }} />
            <span className="font-medium" style={{ color: '#00A651' }}>{t('accounts.bind.summary.verified')}</span>
          </div>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.balance')}</span><span className="font-medium" style={{ color: '#141D22' }}>{Number(verifyResult.balance || 0).toFixed(2)} {verifyResult.currency || ''}</span></div>
            <div className="flex justify-between"><span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.equity')}</span><span className="font-medium" style={{ color: '#141D22' }}>{Number(verifyResult.equity || 0).toFixed(2)} {verifyResult.currency || ''}</span></div>
            <div className="flex justify-between"><span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.margin')}</span><span className="font-medium" style={{ color: '#141D22' }}>{Number(verifyResult.margin || 0).toFixed(2)} {verifyResult.currency || ''}</span></div>
            <div className="flex justify-between"><span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.freeMargin')}</span><span className="font-medium" style={{ color: '#141D22' }}>{Number(verifyResult.freeMargin || 0).toFixed(2)} {verifyResult.currency || ''}</span></div>
            {verifyResult.leverage > 0 && <div className="flex justify-between"><span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.leverage')}</span><span className="font-medium" style={{ color: '#141D22' }}>1:{verifyResult.leverage}</span></div>}
            <div className="flex justify-between"><span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.currency')}</span><span className="font-medium" style={{ color: '#141D22' }}>{verifyResult.currency || '-'}</span></div>
          </div>
        </div>
        <div className="flex justify-between pt-4">
          <Button onClick={() => { setVerifyResult(null); setVerifyError(''); setStep(2); }} style={{ borderRadius: '10px' }}>{t('common.previous')}</Button>
          <GradientButton loading={loading} onClick={handleBind} style={{ padding: '0 32px' }}>
            {t('accounts.bind.actions.confirmBind')}
          </GradientButton>
        </div>
      </>)}
    </div>
  );

  return (
    <div className="min-h-screen" style={{ background: '#F5F7F9' }}>
      <div className="max-w-xl mx-auto p-4">
        <div className="flex items-center gap-4 mb-8">
          <Button type="text" icon={<ArrowLeftOutlined style={{ fontSize: 20 }} />} onClick={() => navigate('/')} style={{ color: '#8A9AA5' }} />
          <h1 className="text-2xl font-bold" style={{ fontFamily: 'Poppins, sans-serif', color: '#141D22' }}>{t('accounts.bind.title')}</h1>
        </div>
        <div className="rounded-2xl p-6" style={{ background: '#FFFFFF', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
          {renderStepIndicator()}
          {step === 1 && renderStep1()}
          {step === 2 && renderStep2()}
          {step === 3 && renderStep3()}
        </div>
      </div>
    </div>
  );
}
