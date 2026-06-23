import './WizardError.scss';

interface WizardErrorProps {
  details: string;
  summary: string;
}

export const WizardError = ({ details, summary }: WizardErrorProps) => (
  <div className="wizard-error" role="alert">
    <p className="wizard-error-summary">{summary}</p>
    <details className="wizard-error-details">
      <summary>Technical details</summary>
      <p>{details}</p>
    </details>
  </div>
);
