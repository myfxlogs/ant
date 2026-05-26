import { Component, type ReactNode } from 'react';
import { Button, Result } from 'antd';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;
      return (
        <Result
          status="error"
          title="Page Error"
          subTitle={this.state.error?.message || 'An unexpected error occurred'}
          extra={
            <Button type="primary" onClick={() => this.setState({ hasError: false, error: null })}>
              Retry
            </Button>
          }
        />
      );
    }
    return this.props.children;
  }
}
