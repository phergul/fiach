import type { WriterEntryDTO } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './DeploymentWriterStack.scss';

interface DeploymentWriterStackProps {
  writers: WriterEntryDTO[];
}

export const DeploymentWriterStack = ({ writers }: DeploymentWriterStackProps) => {
  if (writers.length === 0) {
    return <p className="deployment-writer-stack-empty">No writers recorded for this path.</p>;
  }

  return (
    <ol className="deployment-writer-stack" aria-label="Writer stack">
      {writers.map((writer) => (
        <li
          className={
            writer.IsWinner
              ? 'deployment-writer-stack-item deployment-writer-stack-item-winner'
              : 'deployment-writer-stack-item'
          }
          key={`${writer.Order}-${writer.SourceID}-${writer.ModID ?? 'none'}`}
        >
          <div className="deployment-writer-stack-item-header">
            <span className="deployment-writer-stack-order">{writer.Order}.</span>
            <span className="deployment-writer-stack-name">
              {writer.ModName.trim() !== '' ? writer.ModName : writer.SourceKind}
            </span>
            {writer.IsWinner && (
              <span className="deployment-writer-stack-badge">Final winner</span>
            )}
          </div>
          <p className="deployment-writer-stack-meta">
            {writer.SourceKind === 'mod' && (
              <>
                Load order {writer.DisplayLoadOrder}
                {writer.WouldWrite ? ' · Would write' : ' · Overwritten'}
              </>
            )}
            {writer.SourceKind !== 'mod' && (
              <>{writer.WouldWrite ? 'Would write' : 'Overwritten'}</>
            )}
          </p>
        </li>
      ))}
    </ol>
  );
};
