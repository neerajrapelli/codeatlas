import { Component, type ErrorInfo, type ReactNode } from 'react';

type Props = { children: ReactNode; onReset?: () => void };
type State = { error: Error | null };

export class GraphErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error('Graph render failed', error, info.componentStack);
  }

  private reset = () => {
    this.setState({ error: null });
    this.props.onReset?.();
  };

  render() {
    if (this.state.error) {
      return (
        <div className="graph-error-fallback" role="alert">
          <i className="codicon codicon-warning graph-error-fallback__icon" aria-hidden />
          <h3 className="graph-error-fallback__title">Architecture map unavailable</h3>
          <p className="graph-error-fallback__message">{this.state.error.message}</p>
          <button type="button" className="btn-primary" onClick={this.reset}>
            Retry
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
