import React from 'react';

interface ErrorDetailsProps {
  error?: Error | null;
  detail?: string;
}

const ErrorDetails: React.FC<ErrorDetailsProps> = ({ error, detail }) => {
  if (!error && !detail) return null;
  return (
    <details style={{ marginTop: 8, padding: 8, background: '#fff2f0', borderRadius: 4 }}>
      <summary style={{ cursor: 'pointer', color: '#ff4d4f' }}>Error Details</summary>
      <pre style={{ fontSize: 12, whiteSpace: 'pre-wrap', margin: '8px 0 0' }}>
        {error?.message || detail}
      </pre>
    </details>
  );
};

export default ErrorDetails;
