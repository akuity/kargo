import React from 'react';

interface ErrorProps {
  hasError: boolean;
  children: React.ReactNode;
}
export class ErrorBoundary extends React.Component<ErrorProps> {
  state: { hasError: boolean };

  constructor(props: ErrorProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  render() {
    if (this.state.hasError) {
      return <p>Loading failed! Please reload.</p>;
    }

    return this.props.children;
  }
}
