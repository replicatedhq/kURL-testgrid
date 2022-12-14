import * as React from "react";
import * as Sentry from "@sentry/browser";

export default class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      error: null,
    };
  }

  static getDerivedStateFromError(error) {
    // Update state so the next render will show the fallback UI.
    return { error };
  }

  componentDidCatch(error, errorInfo) {
    this.setState({ error });
    Sentry.withScope(scope => {
      Object.keys(errorInfo).forEach(key => {
        scope.setExtra(key, errorInfo[key]);
      });
      Sentry.captureException(error);
    });
  }

  render() {
    if (this.state.error) {
      return (
        <div className="center">
          <h3>Something unfortunate just happened.</h3>
          <div style={{height: "40px"}}></div>
        </div>
      );
    } else {
      return this.props.children;
    }
  }
}
