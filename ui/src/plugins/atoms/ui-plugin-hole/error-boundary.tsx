import { Badge, Tooltip } from 'antd';
import React, { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children?: ReactNode;
}

interface State {
  hasError: boolean;
}

export class PluginErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false
  };

  public static getDerivedStateFromError(): State {
    // Update state so the next render will show the fallback UI.
    return { hasError: true };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // eslint-disable-next-line no-console
    console.error('Plugin error:', error, errorInfo);
  }

  public render() {
    if (this.state.hasError) {
      return (
        <Tooltip
          title='There is some error with UI Plugin. Please check console and open an issue.'
          className='ml-2'
        >
          <Badge color='red' />
        </Tooltip>
      );
    }

    return this.props.children;
  }
}
