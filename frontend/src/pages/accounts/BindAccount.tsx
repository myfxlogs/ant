import { useState, useContext } from 'react';
import { Button } from 'antd';
import { showSuccess, showError, showWarning, showInfo } from '@/utils/message';
import { ArrowLeftOutlined, CheckOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import GradientButton, { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import { useAccount } from '@/hooks/useAccount';
import { accountApi } from '@/client/account';
import { getErrorMessage } from '@/utils/error';
import type { BindAccountRequest } from '@/types/account';
import type { Account } from '@/types/account';
import { useTranslation } from 'react-i18next'
import { BIND_ERRORS_BROKER_UNAVAILABLE_KEY, BIND_ERRORS_CONNECTION_FAILED_KEY, BIND_ERRORS_INVALID_CREDENTIALS_KEY, BIND_ERRORS_TIMEOUT_KEY, BIND_MESSAGES_BIND_FAILED_KEY, BIND_MESSAGES_BIND_SUCCESS_KEY, BIND_MESSAGES_ENTER_BROKER_NAME_KEY, BIND_MESSAGES_ENTER_PASSWORD_KEY, BIND_MESSAGES_ENTER_TRADING_ACCOUNT_KEY, BIND_MESSAGES_FOUND_BROKERS_KEY, BIND_MESSAGES_NO_ACCESS_HOSTS_KEY, BIND_MESSAGES_NO_BROKERS_FOUND_KEY, BIND_MESSAGES_SEARCH_FAILED_KEY, BIND_MESSAGES_SELECT_SERVER_KEY, BIND_MESSAGES_VERIFY_FAILED_KEY, BIND_TITLE_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;
import i18n from '@/i18n';
import { ConnectContext } from '@/providers/connectContext';
import { Step1SearchBroker, Step2Credentials, Step3Verify } from './BindAccountSteps';

export interface BrokerServer {
  name: string;
  access: string[];
}

export interface BrokerSearchResult {
  companyName: string;
  servers: BrokerServer[];
}

export interface VerifyResult {
  verified: boolean;
  balance?: number;
  equity?: number;
  margin?: number;
  freeMargin?: number;
  leverage?: number;
  currency?: string;
  accountType?: string;
}

/** Translate raw MT broker errors into user-friendly messages via i18n. */
function friendlyError(msg: string | undefined): string {
  if (!msg) return '';
  if (msg.includes('SERVICE_NOT_AVAILABLE') || msg.includes('code=11')) {
    return i18n.t(BIND_ERRORS_BROKER_UNAVAILABLE_KEY);
  }
  if (msg.includes('INVALID_ACCOUNT') || msg.includes('code=1001')) {
    return i18n.t(BIND_ERRORS_INVALID_CREDENTIALS_KEY);
  }
  if (msg.includes('connection failed') || msg.includes('connect')) {
    return i18n.t(BIND_ERRORS_CONNECTION_FAILED_KEY);
  }
  if (msg.includes('timeout') || msg.includes('Timed out')) {
    return i18n.t(BIND_ERRORS_TIMEOUT_KEY);
  }
  return msg;
}

export default function BindAccount() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null);
  const [verifyError, setVerifyError] = useState('');
  const [step, setStep] = useState(1);
  const navigate = useNavigate();
  const { createAccount } = useAccount();
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
    if (!companySearch.trim()) { showWarning(t(BIND_MESSAGES_ENTER_BROKER_NAME_KEY)); return; }
    setSearching(true); setSearchResults([]); setSelectedCompany(null); setSelectedServer(null);
    try {
      const companies = await accountApi.searchBroker(companySearch.trim(), mtType);
      if (companies && companies.length > 0) {
        const results: BrokerSearchResult[] = (companies as Array<Record<string, unknown>>).map((c) => ({
          companyName: String(c.companyName || c.company_name || ''),
          servers: ((c.servers as Array<Record<string, unknown>>) || []).map((s) => ({
            name: String(s.name || ''), access: (s.access as string[]) || [],
          })),
        }));
        setSearchResults(results);
        showSuccess(t(BIND_MESSAGES_FOUND_BROKERS_KEY, { count: results.length }));
      } else {
        showInfo(t(BIND_MESSAGES_NO_BROKERS_FOUND_KEY));
      }
    } catch { showError(t(BIND_MESSAGES_SEARCH_FAILED_KEY)); }
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
    if (!selectedCompany || !selectedServer) { showWarning(t(BIND_MESSAGES_SELECT_SERVER_KEY)); return; }
    if (!login.trim()) { showWarning(t(BIND_MESSAGES_ENTER_TRADING_ACCOUNT_KEY)); return; }
    if (!password.trim()) { showWarning(t(BIND_MESSAGES_ENTER_PASSWORD_KEY)); return; }
    if (!selectedServer.access || selectedServer.access.length === 0) { showError(t(BIND_MESSAGES_NO_ACCESS_HOSTS_KEY)); return; }
    setVerifying(true); setVerifyError(''); setVerifyResult(null);
    try {
      const host = selectedServer.access[0];
      const result = await accountApi.verifyAccount({ login: login.trim(), password, mtType, brokerHost: host });
      setVerifyResult(result);
      if (!result.verified) setVerifyError(friendlyError(result.message) || t(BIND_MESSAGES_VERIFY_FAILED_KEY));
    } catch (error) {
      setVerifyError(friendlyError(getErrorMessage(error, '')) || t(BIND_MESSAGES_VERIFY_FAILED_KEY));
    } finally { setVerifying(false); }
  };

  const handleBind = async () => {
    if (!selectedCompany || !selectedServer) return;
    setLoading(true);
    try {
      if (!selectedServer.access || selectedServer.access.length === 0) { showError(t(BIND_MESSAGES_NO_ACCESS_HOSTS_KEY)); return; }
      const host = selectedServer.access[0];
      const request: BindAccountRequest = {
        alias: alias || selectedServer.name, mtType,
        login: login.trim(), password,
        brokerCompany: selectedCompany.companyName,
        brokerServer: selectedServer.name, brokerHost: host,
      };
      const account = await createAccount(request);
      setPassword('');
      await accountApi.connect(account.id);
      await connectCtx?.reconnect();
      showSuccess(t(BIND_MESSAGES_BIND_SUCCESS_KEY));
      navigate(`/accounts/${account.id}`);
    } catch (error) {
      setPassword('');
      showError(friendlyError(getErrorMessage(error, '')) || t(BIND_MESSAGES_BIND_FAILED_KEY));
    } finally { setLoading(false); }
  };

  const renderStepIndicator = () => (
    <div className="flex items-center justify-center gap-4 mb-8">
      {[1, 2, 3].map((s) => (
        <div key={s} className="flex items-center">
          <div className="w-8 h-8 rounded-full flex items-center justify-center font-medium"
            style={{ background: step >= s ? PRIMARY_GRADIENT : 'var(--color-bg-tertiary)', color: step >= s ? '#FFFFFF' : 'var(--color-text-muted)' }}>
            {step > s ? <CheckOutlined style={{ fontSize: 16 }} /> : s}
          </div>
          {s < 3 && <div className="w-16 h-0.5 mx-2" style={{ background: step > s ? '#D4AF37' : '#E8ECF0' }} />}
        </div>
      ))}
    </div>
  );

  return (
    <div className="min-h-screen" style={{ background: 'var(--color-bg-secondary)' }}>
      <div className="max-w-xl mx-auto p-4">
        <div className="flex items-center gap-4 mb-8">
          <Button type="text" icon={<ArrowLeftOutlined style={{ fontSize: 20 }} />} onClick={() => navigate('/')} style={{ color: 'var(--color-text-muted)' }} />
          <h1 className="text-2xl font-bold" style={{ fontFamily: 'Poppins, sans-serif', color: 'var(--color-text)' }}>{t(BIND_TITLE_KEY)}</h1>
        </div>
        <div className="rounded-2xl p-6" style={{ background: 'var(--color-bg-card)', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
          {renderStepIndicator()}
          {step === 1 && <Step1SearchBroker mtType={mtType} setMtType={setMtType} companySearch={companySearch} setCompanySearch={setCompanySearch} searching={searching} searchResults={searchResults} setSearchResults={setSearchResults} selectedCompany={selectedCompany} selectedServer={selectedServer} setSelectedCompany={setSelectedCompany} setSelectedServer={setSelectedServer} alias={alias} setAlias={setAlias} handleSearch={handleSearch} handleCompanyChange={handleCompanyChange} handleServerChange={handleServerChange} onNext={() => setStep(2)} />}
          {step === 2 && <Step2Credentials mtType={mtType} selectedServer={selectedServer} selectedCompany={selectedCompany} login={login} setLogin={setLogin} password={password} setPassword={setPassword} onBack={() => setStep(1)} onNext={() => setStep(3)} />}
          {step === 3 && <Step3Verify mtType={mtType} selectedServer={selectedServer} selectedCompany={selectedCompany} login={login} verifying={verifying} verifyResult={verifyResult} verifyError={verifyError} loading={loading} setVerifyResult={setVerifyResult} setVerifyError={setVerifyError} handleVerify={handleVerify} handleBind={handleBind} onBack={() => setStep(2)} />}
        </div>
      </div>
    </div>
  );
}
