interface OptiScalerWizardSafetyStepProps {
  executableRelativePath: string;
  onTargetConfirmedChange: (value: boolean) => void;
  onWarningAcknowledgedChange: (value: boolean) => void;
  proxyFilename: string;
  targetConfirmed: boolean;
  warningAcknowledged: boolean;
}

export const OptiScalerWizardSafetyStep = ({
  executableRelativePath,
  onTargetConfirmedChange,
  onWarningAcknowledgedChange,
  proxyFilename,
  targetConfirmed,
  warningAcknowledged,
}: OptiScalerWizardSafetyStepProps) => (
  <div className="optiscaler-wizard-content">
    <label className="optiscaler-wizard-checkbox">
      <input
        checked={targetConfirmed}
        onChange={(event) => onTargetConfirmedChange(event.target.checked)}
        type="checkbox"
      />
      I confirm that {executableRelativePath} and {proxyFilename} are correct.
    </label>
    <div className="optiscaler-wizard-warning">
      <p>
        OptiScaler can be incompatible with online games and anti-cheat systems. Candidate ranking and
        upstream compatibility reports do not guarantee that this game is safe or supported.
      </p>
      <a
        href="https://github.com/optiscaler/OptiScaler/wiki/Compatibility-List"
        rel="noreferrer"
        target="_blank"
      >
        Review upstream compatibility guidance
      </a>
    </div>
    <label className="optiscaler-wizard-checkbox">
      <input
        checked={warningAcknowledged}
        onChange={(event) => onWarningAcknowledgedChange(event.target.checked)}
        type="checkbox"
      />
      I understand the online-game and anti-cheat risk.
    </label>
  </div>
);
