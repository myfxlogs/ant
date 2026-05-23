import { useState } from 'react';
import { Button, Modal, Select, Tag } from 'antd';
import { showSuccess, showError, showWarning } from '@/utils/message';
import {
  IconArrowLeft,
  IconServer,
  IconCheck,
  IconAlertCircle,
} from '@tabler/icons-react';
import { useNavigate } from 'react-router-dom';
import GradientButton, { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import { useAccount } from '@/hooks/useAccount';
import { accountApi } from '@/client/account';
import { getErrorMessage } from '@/utils/error';
import { extractErrorDetail } from '@/utils/llmTranslate';
import ErrorDetails from '@/components/common/ErrorDetails';
import type { BindAccountRequest } from '@/types/account';
import { useTranslation } from 'react-i18next';

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
  const [step, setStep] = useState(1);
  const [errorModalVisible, setErrorModalVisible] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [errorDetail, setErrorDetail] = useState('');
  const navigate = useNavigate();
  const { bindAccount } = useAccount();

  const [mtType, setMtType] = useState<'MT4' | 'MT5'>('MT4');
  const [companySearch, setCompanySearch] = useState('');
  const [searchResults, setSearchResults] = useState<BrokerSearchResult[]>([]);
  const [selectedCompany, setSelectedCompany] = useState<BrokerSearchResult | null>(null);
  const [selectedServer, setSelectedServer] = useState<BrokerServer | null>(null);
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [alias, setAlias] = useState('');

  const handleSearch = async () => {
    if (!companySearch.trim()) {
      showWarning(t('accounts.bind.messages.enterBrokerName'));
      return;
    }

    setSearching(true);
    setSearchResults([]);
    setSelectedCompany(null);
    setSelectedServer(null);

    try {
      const companies = await accountApi.searchBroker(companySearch.trim(), mtType);
      
      if (companies && companies.length > 0) {
        const results = companies.map((c: any) => ({
          companyName: c.companyName || c.company_name,
          servers: (c.servers || []).map((s: any) => ({
            name: s.name,
            access: s.access,
          })),
        }));
        setSearchResults(results);
        showSuccess(t('accounts.bind.messages.foundBrokers', { count: results.length }));
      } else {
        message.info(t('accounts.bind.messages.noBrokersFound'));
      }
    } catch (_error) {
      showError(t('accounts.bind.messages.searchFailed'));
    } finally {
      setSearching(false);
    }
  };

  const handleCompanyChange = (companyName: string) => {
    const company = searchResults.find(c => c.companyName === companyName);
    setSelectedCompany(company || null);
    setSelectedServer(null);
  };

  const handleServerChange = (serverName: string) => {
    const server = selectedCompany?.servers.find(s => s.name === serverName);
    if (server) {
      setSelectedServer(server);
      if (!alias) {
        setAlias(server.name);
      }
    }
  };

  const handleBind = async () => {
    if (!selectedCompany || !selectedServer) {
      showWarning(t('accounts.bind.messages.selectServer'));
      return;
    }
    if (!login.trim()) {
      showWarning(t('accounts.bind.messages.enterTradingAccount'));
      return;
    }
    if (!password.trim()) {
      showWarning(t('accounts.bind.messages.enterPassword'));
      return;
    }

    setLoading(true);
    try {
      const host = selectedServer.access[0] || '';
      const request: BindAccountRequest = {
        alias: alias || selectedServer.name,
        mtType: mtType,
        login: login.trim(),
        password: password,
        brokerCompany: selectedCompany.companyName,
        brokerServer: selectedServer.name,
        brokerHost: host,
      };

      await bindAccount(request as any);
      showSuccess(t('accounts.bind.messages.bindSuccess'));
      navigate('/');
    } catch (error) {
      setErrorMessage(getErrorMessage(error, t('accounts.bind.messages.bindFailed')));
      setErrorDetail(extractErrorDetail(error));
      setErrorModalVisible(true);
    } finally {
      setLoading(false);
    }
  };

  const renderStepIndicator = () => (
    <div className="flex items-center justify-center gap-4 mb-8">
      {[1, 2, 3].map((s) => (
        <div key={s} className="flex items-center">
          <div
            className="w-8 h-8 rounded-full flex items-center justify-center font-medium"
            style={{
              background: step >= s ? PRIMARY_GRADIENT : '#E8ECF0',
              color: step >= s ? '#FFFFFF' : '#8A9AA5',
            }}
          >
            {step > s ? <IconCheck size={16} stroke={2} /> : s}
          </div>
          {s < 3 && (
            <div
              className="w-16 h-0.5 mx-2"
              style={{ background: step > s ? '#D4AF37' : '#E8ECF0' }}
            />
          )}
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
          <div
            onClick={() => setMtType('MT4')}
            className="flex-1 p-4 rounded-xl cursor-pointer transition-all"
            style={{
              background: mtType === 'MT4' ? 'rgba(212, 175, 55, 0.1)' : '#F5F7F9',
              border: `2px solid ${mtType === 'MT4' ? '#D4AF37' : 'transparent'}`,
            }}
          >
            <div className="text-center">
              <div className="text-2xl font-bold" style={{ color: mtType === 'MT4' ? '#D4AF37' : '#141D22' }}>
                MT4
              </div>
              <div className="text-sm mt-1" style={{ color: '#8A9AA5' }}>MetaTrader 4</div>
            </div>
          </div>
          <div
            onClick={() => setMtType('MT5')}
            className="flex-1 p-4 rounded-xl cursor-pointer transition-all"
            style={{
              background: mtType === 'MT5' ? 'rgba(212, 175, 55, 0.1)' : '#F5F7F9',
              border: `2px solid ${mtType === 'MT5' ? '#D4AF37' : 'transparent'}`,
            }}
          >
            <div className="text-center">
              <div className="text-2xl font-bold" style={{ color: mtType === 'MT5' ? '#D4AF37' : '#141D22' }}>
                MT5
              </div>
              <div className="text-sm mt-1" style={{ color: '#8A9AA5' }}>MetaTrader 5</div>
            </div>
          </div>
        </div>
      </div>

      <div>
        <label className="block mb-3 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.brokerName')}</label>
        <div className="flex gap-2">
          <input
            type="text"
            value={companySearch}
            onChange={(e) => setCompanySearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            placeholder={t('accounts.bind.placeholders.brokerName')}
            className="flex-1 outline-none transition-all"
            style={{
              background: '#FFFFFF',
              border: '1px solid rgba(185, 201, 223, 0.4)',
              borderRadius: '10px',
              padding: '14px 16px',
              fontSize: '16px',
              color: '#141D22',
              height: '48px',
            }}
          />
          <GradientButton
            onClick={handleSearch}
            loading={searching}
            style={{ 
              padding: '0 24px', 
              height: '48px',
            }}
          >
            {t('accounts.bind.actions.search')}
          </GradientButton>
        </div>
      </div>

      {searchResults.length > 0 && (
        <>
          <div>
            <label className="block mb-2 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.company')}</label>
            <Select
              placeholder={t('accounts.bind.placeholders.company')}
              value={selectedCompany?.companyName}
              onChange={handleCompanyChange}
              style={{ width: '100%' }}
              size="large"
              optionLabelProp="label"
            >
              {searchResults.map((company) => (
                <Select.Option 
                  key={company.companyName} 
                  value={company.companyName}
                  label={company.companyName}
                >
                  <div className="flex items-center justify-between">
                    <span>{company.companyName}</span>
                    <Tag color="blue">{t('accounts.bind.labels.serverCount', { count: company.servers.length })}</Tag>
                  </div>
                </Select.Option>
              ))}
            </Select>
          </div>

          {selectedCompany && (
            <div>
              <label className="block mb-2 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.server')}</label>
              <Select
                placeholder={t('accounts.bind.placeholders.server')}
                value={selectedServer?.name}
                onChange={handleServerChange}
                style={{ width: '100%' }}
                size="large"
                optionLabelProp="label"
              >
                {selectedCompany.servers.map((server) => (
                  <Select.Option 
                    key={server.name} 
                    value={server.name}
                    label={server.name}
                  >
                    <div className="flex items-center justify-between">
                      <span>{server.name}</span>
                      <Tag color={mtType === 'MT4' ? 'blue' : 'purple'}>{mtType}</Tag>
                    </div>
                  </Select.Option>
                ))}
              </Select>
            </div>
          )}
        </>
      )}

      <div className="flex justify-end pt-4">
        <GradientButton
          disabled={!selectedServer}
          onClick={() => setStep(2)}
          style={{ padding: '0 32px' }}
        >
          {t('common.next')}
        </GradientButton>
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
          <IconServer size={20} stroke={1.5} color="#D4AF37" />
          <div>
            <div className="font-medium" style={{ color: '#141D22' }}>{selectedServer?.name}</div>
            <div className="text-sm" style={{ color: '#8A9AA5' }}>{selectedCompany?.companyName} · {mtType}</div>
          </div>
        </div>
      </div>

      <div>
        <label className="block mb-2 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.tradingAccount')}</label>
        <input
          type="text"
          value={login}
          onChange={(e) => setLogin(e.target.value)}
          placeholder={t('accounts.bind.placeholders.tradingAccount')}
          className="w-full outline-none transition-all"
          style={{
            background: '#FFFFFF',
            border: '1px solid rgba(185, 201, 223, 0.4)',
            borderRadius: '10px',
            padding: '14px 16px',
            fontSize: '16px',
            color: '#141D22',
            height: '48px',
          }}
        />
      </div>

      <div>
        <label className="block mb-2 font-medium" style={{ color: '#141D22' }}>{t('accounts.bind.fields.password')}</label>
        <input
          type="text"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={t('accounts.bind.placeholders.password')}
          className="w-full outline-none transition-all"
          style={{
            background: '#FFFFFF',
            border: '1px solid rgba(185, 201, 223, 0.4)',
            borderRadius: '10px',
            padding: '14px 16px',
            fontSize: '16px',
            color: '#141D22',
            height: '48px',
          }}
        />
        <p className="mt-2 text-sm" style={{ color: '#8A9AA5' }}>
          {t('accounts.bind.passwordHint')}
        </p>
      </div>

      <div className="flex justify-between pt-4">
        <Button
          onClick={() => setStep(1)}
          style={{ borderRadius: '10px' }}
        >
          {t('common.previous')}
        </Button>
        <GradientButton
          disabled={!login.trim() || !password.trim()}
          onClick={() => setStep(3)}
          style={{ padding: '0 32px' }}
        >
          {t('common.next')}
        </GradientButton>
      </div>
    </div>
  );

  const renderStep3 = () => (
    <div className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-xl font-semibold" style={{ color: '#141D22' }}>{t('accounts.bind.step3.title')}</h2>
        <p className="mt-2" style={{ color: '#8A9AA5' }}>{t('accounts.bind.step3.subtitle')}</p>
      </div>

      <div className="space-y-4">
        <div className="p-4 rounded-xl" style={{ background: '#F5F7F9' }}>
          <div className="flex justify-between mb-3">
            <span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.broker')}</span>
            <span className="font-medium" style={{ color: '#141D22' }}>{selectedCompany?.companyName}</span>
          </div>
          <div className="flex justify-between mb-3">
            <span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.server')}</span>
            <span className="font-medium" style={{ color: '#141D22' }}>{selectedServer?.name}</span>
          </div>
          <div className="flex justify-between mb-3">
            <span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.platform')}</span>
            <Tag color={mtType === 'MT4' ? 'blue' : 'purple'}>{mtType}</Tag>
          </div>
          <div className="flex justify-between mb-3">
            <span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.tradingAccount')}</span>
            <span className="font-medium" style={{ color: '#141D22' }}>{login}</span>
          </div>
          <div className="flex justify-between">
            <span style={{ color: '#8A9AA5' }}>{t('accounts.bind.summary.password')}</span>
            <span className="font-medium" style={{ color: '#141D22' }}>{password}</span>
          </div>
        </div>
      </div>

      <div className="flex justify-between pt-4">
        <Button
          onClick={() => setStep(2)}
          style={{ borderRadius: '10px' }}
        >
          {t('common.previous')}
        </Button>
        <GradientButton
          loading={loading}
          onClick={handleBind}
          style={{ padding: '0 32px' }}
        >
          {t('accounts.bind.actions.confirmBind')}
        </GradientButton>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen" style={{ background: '#F5F7F9' }}>
      <div className="max-w-xl mx-auto p-4">
        <div className="flex items-center gap-4 mb-8">
          <Button
            type="text"
            icon={<IconArrowLeft size={20} stroke={1.5} />}
            onClick={() => navigate('/')}
            style={{ color: '#8A9AA5' }}
          />
          <h1 className="text-2xl font-bold" style={{ fontFamily: 'Poppins, sans-serif', color: '#141D22' }}>
            {t('accounts.bind.title')}
          </h1>
        </div>

        <div
          className="rounded-2xl p-6"
          style={{
            background: '#FFFFFF',
            boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)',
          }}
        >
          {renderStepIndicator()}

          {step === 1 && renderStep1()}
          {step === 2 && renderStep2()}
          {step === 3 && renderStep3()}
        </div>
      </div>

      <Modal
        open={errorModalVisible}
        onCancel={() => setErrorModalVisible(false)}
        footer={null}
        centered
        width={400}
      >
        <div className="text-center py-4">
          <div 
            className="mx-auto mb-4 w-16 h-16 rounded-full flex items-center justify-center"
            style={{ background: 'rgba(255, 77, 79, 0.1)' }}
          >
            <IconAlertCircle size={32} stroke={1.5} color="#FF4D4F" />
          </div>
          <h3 className="text-lg font-semibold mb-2" style={{ color: '#141D22' }}>
            {t('accounts.bind.errorModal.title')}
          </h3>
          <p className="text-base mb-6" style={{ color: '#8A9AA5' }}>
            {errorMessage}
          </p>

          <ErrorDetails detail={errorDetail} />

          <GradientButton
            onClick={() => setErrorModalVisible(false)}
            style={{ padding: '0 32px' }}
          >
            {t('common.gotIt')}
          </GradientButton>
        </div>
      </Modal>
    </div>
  );
}
