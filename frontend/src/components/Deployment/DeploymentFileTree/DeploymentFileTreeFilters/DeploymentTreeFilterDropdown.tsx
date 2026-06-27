import { useState } from 'react';

import { ChevronDown } from 'lucide-react';

import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';

import './DeploymentTreeFilterDropdown.scss';

export interface DeploymentTreeFilterOption {
  label: string;
  value: string;
}

interface DeploymentTreeFilterDropdownProps {
  label: string;
  options: DeploymentTreeFilterOption[];
  selectedValues: string[];
  onChange: (values: string[]) => void;
}

const toggleValue = (values: string[], value: string) => {
  if (values.includes(value)) {
    return values.filter((currentValue) => currentValue !== value);
  }

  return [...values, value];
};

export const DeploymentTreeFilterDropdown = ({
  label,
  options,
  selectedValues,
  onChange,
}: DeploymentTreeFilterDropdownProps) => {
  const [isOpen, setIsOpen] = useState(false);
  const isActive = selectedValues.length > 0;

  return (
    <div className="deployment-tree-filter-dropdown">
      <div className="deployment-tree-filter-dropdown-control">
        <button
          aria-expanded={isOpen}
          className={
            isActive
              ? 'deployment-tree-filter-dropdown-button deployment-tree-filter-dropdown-button-active'
              : 'deployment-tree-filter-dropdown-button'
          }
          onClick={() => setIsOpen((currentValue) => !currentValue)}
          type="button"
        >
          <span>{label}</span>
          {selectedValues.length > 0 && (
            <span className="deployment-tree-filter-dropdown-count">({selectedValues.length})</span>
          )}
          <ChevronDown className="deployment-tree-filter-dropdown-chevron" aria-hidden="true" />
        </button>

        <DropdownMenu
          align="right"
          ariaLabel={`${label} filter`}
          isOpen={isOpen}
          items={options.map((option) => ({
            checked: selectedValues.includes(option.value),
            label: option.label,
            onSelect: () => onChange(toggleValue(selectedValues, option.value)),
            type: 'checkbox',
          }))}
        />
      </div>
    </div>
  );
};
