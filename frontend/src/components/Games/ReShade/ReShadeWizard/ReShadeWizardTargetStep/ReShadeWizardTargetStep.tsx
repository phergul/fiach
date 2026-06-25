import type { ReShadeOperationSelection } from '@components/Games/ReShade/ReShadeTargetTable/ReShadeTargetTable';

import './ReShadeWizardTargetStep.scss';

interface ReShadeWizardTargetStepProps {
  selection: ReShadeOperationSelection;
}

const filename = (path: string) => path.split(/[\\/]/).pop() ?? path;

export const ReShadeWizardTargetStep = ({ selection }: ReShadeWizardTargetStepProps) => {
  const executable =
    selection.candidate?.executableRelativePath ?? selection.target?.ExecutableRelativePath ?? '';
  const targetPath =
    selection.candidate?.targetRelativePath ?? selection.target?.TargetRelativePath ?? '';
  const evidence = selection.candidate?.proxyEvidence ?? [];

  return (
    <div className="reshade-wizard-content">
      <div className="reshade-wizard-target-step">
        <dl>
          <div>
            <dt>Executable</dt>
            <dd>{filename(executable)}</dd>
          </div>
          <div>
            <dt>Target folder</dt>
            <dd>{targetPath === '.' ? 'Game Root' : targetPath}</dd>
          </div>
        </dl>
        {evidence.length > 0 && (
          <section>
            <h3>Proxy evidence</h3>
            <ul>
              {evidence.map((item) => (
                <li key={item.filename}>
                  {item.filename}: {item.isReShade ? 'ReShade' : (item.conflict ?? 'present')}
                </li>
              ))}
            </ul>
          </section>
        )}
      </div>
    </div>
  );
};
