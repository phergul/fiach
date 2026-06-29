import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { Breadcrumbs } from './Breadcrumbs';

const renderBreadcrumbs = () =>
  render(
    <Breadcrumbs
      items={[
        {
          label: 'Counter-Strike 2',
        },
        {
          label: 'Deployment',
        },
        {
          label: 'profile-1',
        },
      ]}
    />,
  );

describe('Breadcrumbs', () => {
  it('renders all crumbs as display text', () => {
    renderBreadcrumbs();

    expect(screen.getByText('Counter-Strike 2')).toBeInTheDocument();
    expect(screen.getByText('Deployment')).toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('renders the current crumb as page text', () => {
    renderBreadcrumbs();

    const currentCrumb = screen.getByText('profile-1');
    expect(currentCrumb).toHaveAttribute('aria-current', 'page');
    expect(screen.queryByRole('link', { name: 'profile-1' })).not.toBeInTheDocument();
  });

  it('hides separators from assistive tech', () => {
    const { container } = renderBreadcrumbs();

    const separators = container.querySelectorAll('.breadcrumbs-separator');
    expect(separators).toHaveLength(2);
    separators.forEach((separator) => {
      expect(separator).toHaveAttribute('aria-hidden', 'true');
    });
  });
});
