import type { ArchiveEntry } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { formatDeploymentBytes } from '@utils';

import {
  DeploymentCompareColumn,
  DeploymentCompareColumnPlaceholder,
  DeploymentCompareGrid,
} from '../DeploymentCompareColumn/DeploymentCompareColumn';

import './DeploymentArchiveListing.scss';

interface DeploymentArchiveListingProps {
  leftEntries: ArchiveEntry[];
  leftLabel: string;
  rightEntries: ArchiveEntry[];
  rightLabel: string;
}

const EntryList = ({ entries }: { entries: ArchiveEntry[] }) => {
  if (entries.length === 0) {
    return <DeploymentCompareColumnPlaceholder />;
  }

  return (
    <ul className="deployment-archive-listing-entries">
      {entries.map((entry) => (
        <li className="deployment-archive-listing-entry" key={entry.Path}>
          <span className="deployment-archive-listing-path">{entry.Path}</span>
          <span className="deployment-archive-listing-meta">
            {entry.IsDirectory ? 'directory' : formatDeploymentBytes(entry.SizeBytes)}
          </span>
        </li>
      ))}
    </ul>
  );
};

export const DeploymentArchiveListing = ({
  leftEntries,
  leftLabel,
  rightEntries,
  rightLabel,
}: DeploymentArchiveListingProps) => (
  <DeploymentCompareGrid aria-label="Archive listing comparison">
    <DeploymentCompareColumn isEmpty={leftEntries.length === 0} label={leftLabel}>
      <EntryList entries={leftEntries} />
    </DeploymentCompareColumn>
    <DeploymentCompareColumn isDesired isEmpty={rightEntries.length === 0} label={rightLabel}>
      <EntryList entries={rightEntries} />
    </DeploymentCompareColumn>
  </DeploymentCompareGrid>
);
