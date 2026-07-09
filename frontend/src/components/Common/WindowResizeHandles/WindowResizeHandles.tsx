import './WindowResizeHandles.scss';

interface WindowResizeHandlesProps {
  disabled?: boolean;
}

const resizeDirections = [
  'top',
  'right',
  'bottom',
  'left',
  'top-left',
  'top-right',
  'bottom-right',
  'bottom-left',
];

export const WindowResizeHandles = ({ disabled = false }: WindowResizeHandlesProps) => {
  if (disabled) {
    return null;
  }

  return (
    <div className="window-resize-handles" aria-hidden="true">
      {resizeDirections.map((direction) => (
        <div
          className={`window-resize-handle window-resize-handle-${direction}`}
          key={direction}
        />
      ))}
    </div>
  );
};
