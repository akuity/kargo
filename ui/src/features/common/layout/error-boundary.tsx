import React from 'react';

interface ErrorProps {
  children: React.ReactNode;
  errorRender?: React.ReactNode;
  onError?: (err: string) => void;
}
export class ErrorBoundary extends React.Component<ErrorProps> {
  state: { hasError: boolean; err?: string };

  constructor(props: ErrorProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  componentDidCatch(error: Error) {
    // eslint-disable-next-line no-console
    console.error(error);
    this.props.onError?.(error?.message);
  }

  render() {
    if (this.state.hasError) {
      return this.props.errorRender || <p>Loading failed! Please reload.</p>;
    }

    return this.props.children;
  }
}
