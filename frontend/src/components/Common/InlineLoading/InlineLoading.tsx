import './InlineLoading.scss';

interface InlineLoadingProps {
  className?: string;
  label: string;
}

export const InlineLoading = ({ className, label }: InlineLoadingProps) => {
  const inlineLoadingClassName =
    className === undefined ? 'inline-loading' : `inline-loading ${className}`;

  return (
    <div className={inlineLoadingClassName} role="status" aria-live="polite">
      <span className="inline-loading-spinner" aria-hidden="true" />
      <span>{label}</span>
    </div>
  );
};
