import { GraphicsAPI } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';

interface OptiScalerWizardConfigurationStepProps {
  dxgiSpoofing: boolean | null;
  enableReShadeCoexistence: boolean;
  graphicsAPI: GraphicsAPI | '';
  hasDetectedReShade: boolean;
  onChooseGraphicsAPI: (value: GraphicsAPI | '') => void;
  onDXGISpoofingChange: (value: boolean | null) => void;
  onProcessFilterChange: (value: string) => void;
  onProxyFilenameChange: (value: string) => void;
  processFilter: string;
  proxyFilename: string;
  supportedProxyFilenames: string[];
}

export const OptiScalerWizardConfigurationStep = ({
  dxgiSpoofing,
  enableReShadeCoexistence,
  graphicsAPI,
  hasDetectedReShade,
  onChooseGraphicsAPI,
  onDXGISpoofingChange,
  onProcessFilterChange,
  onProxyFilenameChange,
  processFilter,
  proxyFilename,
  supportedProxyFilenames,
}: OptiScalerWizardConfigurationStepProps) => (
  <div className="optiscaler-wizard-content">
    <div className="optiscaler-wizard-fields">
      <label>
        Graphics API
        <select
          onChange={(event) => onChooseGraphicsAPI(event.target.value as GraphicsAPI | '')}
          value={graphicsAPI}
        >
          <option value="">Choose an API</option>
          <option value={GraphicsAPI.GraphicsAPIDirectX}>DirectX</option>
          <option value={GraphicsAPI.GraphicsAPIVulkan}>Vulkan</option>
        </select>
      </label>
      <label>
        Proxy filename
        <select
          disabled={graphicsAPI === ''}
          onChange={(event) => onProxyFilenameChange(event.target.value)}
          value={proxyFilename}
        >
          {supportedProxyFilenames.map((filename) => (
            <option key={filename} value={filename}>{filename}</option>
          ))}
        </select>
        {graphicsAPI !== '' && (
          <span>Recommended: {graphicsAPI === GraphicsAPI.GraphicsAPIDirectX ? 'dxgi.dll' : 'winmm.dll'}</span>
        )}
      </label>
      <label>
        DXGI spoofing
        <select
          onChange={(event) => onDXGISpoofingChange(
            event.target.value === '' ? null : event.target.value === 'true',
          )}
          value={dxgiSpoofing === null ? '' : String(dxgiSpoofing)}
        >
          <option value="">Choose a setting</option>
          <option value="false">Disabled</option>
          <option value="true">Enabled</option>
        </select>
      </label>
      <label>
        Process filter
        <input
          onChange={(event) => onProcessFilterChange(event.target.value)}
          placeholder="Leave empty for a shared executable directory"
          type="text"
          value={processFilter}
        />
        <span>Defaults to the selected executable and may be cleared.</span>
      </label>
      {hasDetectedReShade && (
        <p>
          {enableReShadeCoexistence
            ? 'Detected ReShade will be chained through OptiScaler.'
            : 'Detected ReShade cannot be chained for this configuration.'}
        </p>
      )}
      {graphicsAPI === GraphicsAPI.GraphicsAPIVulkan && hasDetectedReShade && (
        <p className="optiscaler-wizard-error-inline">
          Automated Vulkan and ReShade coexistence is not supported.
        </p>
      )}
    </div>
  </div>
);
