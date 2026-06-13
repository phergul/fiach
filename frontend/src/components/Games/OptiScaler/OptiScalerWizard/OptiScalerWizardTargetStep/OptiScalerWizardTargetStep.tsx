import type { OptiScalerOperationSelection } from '../OptiScalerWizard';

interface OptiScalerWizardTargetStepProps {
  actionLabel: string;
  selection: OptiScalerOperationSelection;
}

export const OptiScalerWizardTargetStep = ({
  actionLabel,
  selection,
}: OptiScalerWizardTargetStepProps) => {
  const executableRelativePath =
    selection.candidate?.executableRelativePath ?? selection.target?.ExecutableRelativePath ?? '';
  const executableName = executableRelativePath.split(/[\\/]/).pop() ?? executableRelativePath;
  const targetRelativePath =
    selection.candidate?.targetRelativePath ?? selection.target?.TargetRelativePath ?? '';

  return (
    <div className="optiscaler-wizard-content">
      <dl className="optiscaler-wizard-summary">
        <div><dt>Executable</dt><dd>{executableName}</dd></div>
        <div><dt>Directory</dt><dd>{targetRelativePath}</dd></div>
        <div><dt>Architecture</dt><dd>{selection.candidate?.architecture ?? 'x64'}</dd></div>
        <div><dt>Action</dt><dd>{actionLabel}</dd></div>
      </dl>
      {selection.candidate?.hasOptiScaler && (
        <p className="optiscaler-wizard-warning">
          OptiScaler files are already present in this directory. Adoption will have these files become managed.
        </p>
      )}
    </div>
  );
};
