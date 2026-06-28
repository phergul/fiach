import type { TextDiffLine } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './DeploymentTextDiff.scss';

interface DeploymentTextDiffProps {
  lines: TextDiffLine[];
}

export const DeploymentTextDiff = ({ lines }: DeploymentTextDiffProps) => {
  const visibleLines = lines.filter((line) => line.Kind !== 'equal');

  if (visibleLines.length === 0) {
    return <p className="deployment-text-diff-empty">No textual differences.</p>;
  }

  return (
    <div className="deployment-text-diff" aria-label="Text diff">
      {visibleLines.map((line, index) => (
        <div
          className={[
            'deployment-text-diff-line',
            line.Kind === 'insert' ? 'deployment-text-diff-line-insert' : '',
            line.Kind === 'delete' ? 'deployment-text-diff-line-delete' : '',
          ]
            .filter(Boolean)
            .join(' ')}
          key={`${line.Kind}-${line.LineNo}-${index}`}
        >
          <span className="deployment-text-diff-prefix">{line.Kind === 'insert' ? '+' : '-'}</span>
          <code className="deployment-text-diff-content">{line.Line}</code>
        </div>
      ))}
    </div>
  );
};
