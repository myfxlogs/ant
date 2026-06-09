import { Component, type ReactNode } from 'react';
import { Button, Result } from 'antd';
import { withTranslation, type TFunction } from 'react-i18next';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  t: TFunction;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

class ErrorBoundaryClass extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;
      const { t } = this.props;
      return (
        <Result
          status="error"
          title={t('common.pageError')}
          subTitle={this.state.error?.message || t('common.unexpectedError')}
          extra={
            <Button type="primary" onClick={() => this.setState({ hasError: false, error: null })}>
              {t('common.retry')}
            </Button>
          }
        />
      );
    }
    return this.props.children;
  }
}

export const ErrorBoundary = withTranslation()(ErrorBoundaryClass);
