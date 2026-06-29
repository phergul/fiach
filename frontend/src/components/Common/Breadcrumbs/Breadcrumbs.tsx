import { ChevronRight } from 'lucide-react';

import './Breadcrumbs.scss';

export interface BreadcrumbItem {
  label: string;
}

interface BreadcrumbsProps {
  items: BreadcrumbItem[];
}

export const Breadcrumbs = ({ items }: BreadcrumbsProps) => {
  if (items.length === 0) {
    return null;
  }

  const lastItemIndex = items.length - 1;

  return (
    <nav className="breadcrumbs" aria-label="Breadcrumb">
      <ol className="breadcrumbs-list">
        {items.map((item, index) => {
          const isCurrentItem = index === lastItemIndex;
          const itemKey = `${item.label}-${index}`;

          return (
            <li className="breadcrumbs-item" key={itemKey}>
              <span
                className={
                  isCurrentItem ? 'breadcrumbs-label breadcrumbs-current' : 'breadcrumbs-label'
                }
                aria-current={isCurrentItem ? 'page' : undefined}
              >
                {item.label}
              </span>

              {!isCurrentItem && (
                <ChevronRight className="breadcrumbs-separator" aria-hidden="true" />
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
};
