import { message } from 'antd';
import { pythonStrategyApi, type ValidateStrategyResult } from '@/client/pythonStrategy';
import i18n from '@/i18n';

export type StrictValidateOptions = {
  showMessage?: boolean;
};

export type StrictValidateResult = {
  ok: boolean;
  result: ValidateStrategyResult;
};

export async function validateStrategyCodeStrict(code: string, opts?: StrictValidateOptions): Promise<StrictValidateResult> {
  const res = await pythonStrategyApi.validate(code);
  const errors = Array.isArray(res?.errors) ? res.errors : [];
  const warnings = Array.isArray(res?.warnings) ? res.warnings : [];
  const valid = Boolean(res?.valid);
  const ok = valid && errors.length === 0 && warnings.length === 0;

  if (opts?.showMessage) {
    if (ok) {
      message.success(i18n.t('strategy.validation.passed'));
    } else {
      message.error(errors[0] || warnings[0] || i18n.t('strategy.validation.notPassed'));
    }
  }

  return {
    ok,
    result: {
      valid,
      errors,
      warnings,
    },
  };
}
