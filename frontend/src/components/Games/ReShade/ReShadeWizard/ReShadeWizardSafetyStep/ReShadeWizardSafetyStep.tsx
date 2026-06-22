import './ReShadeWizardSafetyStep.scss';

interface ReShadeWizardSafetyStepProps {
  antiCheatRiskAcknowledged: boolean;
  onAntiCheatRiskAcknowledgedChange: (value: boolean) => void;
  onSinglePlayerAcknowledgedChange: (value: boolean) => void;
  singlePlayerAcknowledged: boolean;
}

export const ReShadeWizardSafetyStep = ({
  antiCheatRiskAcknowledged,
  onAntiCheatRiskAcknowledgedChange,
  onSinglePlayerAcknowledgedChange,
  singlePlayerAcknowledged,
}: ReShadeWizardSafetyStepProps) => (
  <div className="reshade-wizard-content">
    <div className="reshade-wizard-safety-step">
      <p>The full add-on build is unsigned and should only be used where add-ons are appropriate.</p>
      <label>
        <input
          checked={singlePlayerAcknowledged}
          onChange={(event) => onSinglePlayerAcknowledgedChange(event.target.checked)}
          type="checkbox"
        />
        <span>I will use this ReShade build for single-player or explicitly allowed scenarios.</span>
      </label>
      <label>
        <input
          checked={antiCheatRiskAcknowledged}
          onChange={(event) => onAntiCheatRiskAcknowledgedChange(event.target.checked)}
          type="checkbox"
        />
        <span>I understand anti-cheat protected games may block or ban add-on usage.</span>
      </label>
    </div>
  </div>
);
