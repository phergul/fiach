import { X } from 'lucide-react';

import './WizardError.scss';

interface WizardErrorProps {
  details: string;
  onClose: () => void;
  summary: string;
}

export const WizardError = ({ details, onClose, summary }: WizardErrorProps) => (
  <div className="wizard-error" role="alert">
    <div className="wizard-content">
      <div className="wizard-error-header">
        <p className="wizard-error-summary">{summary}</p>
      </div>
      <div className="wizard-error-body">
        <details className="wizard-error-details">
          <summary>Technical details</summary>
          <p>{details}</p>
        </details>
      </div>
    </div>
    <button
      aria-label="Dismiss error"
      className="wizard-error-close"
      onClick={onClose}
      type="button"
    >
      <X aria-hidden="true" />
    </button>
  </div>
);
