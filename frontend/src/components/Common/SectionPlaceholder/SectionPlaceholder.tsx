import './SectionPlaceholder.scss';

interface SectionPlaceholderProps {
  description: string;
  title: string;
}

export const SectionPlaceholder = ({ description, title }: SectionPlaceholderProps) => {
  return (
    <section className="section-placeholder" aria-labelledby={`${title.toLowerCase()}-title`}>
      <h1 className="section-placeholder-title" id={`${title.toLowerCase()}-title`}>
        {title}
      </h1>
      <p className="section-placeholder-description">{description}</p>
    </section>
  );
};
