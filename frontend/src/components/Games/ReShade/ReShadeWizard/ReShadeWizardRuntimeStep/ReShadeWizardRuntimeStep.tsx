import { BuildVariant, RenderingAPI } from '@bindings/github.com/phergul/fiach/internal/reshade/models';

import './ReShadeWizardRuntimeStep.scss';

interface APIOption {
  renderingApi: RenderingAPI;
  proxies: string[];
}

interface ReShadeWizardRuntimeStepProps {
  apiOptions: APIOption[];
  buildVariant: BuildVariant;
  onBuildVariantChange: (value: BuildVariant) => void;
  onProxyFilenameChange: (value: string) => void;
  onRenderingAPIChange: (value: RenderingAPI) => void;
  proxyFilename: string;
  renderingAPI: RenderingAPI | '';
}

export const ReShadeWizardRuntimeStep = ({
  apiOptions,
  buildVariant,
  onBuildVariantChange,
  onProxyFilenameChange,
  onRenderingAPIChange,
  proxyFilename,
  renderingAPI,
}: ReShadeWizardRuntimeStepProps) => {
  const selectedAPI = apiOptions.find((option) => option.renderingApi === renderingAPI);
  const proxies = selectedAPI?.proxies ?? [];

  return (
    <div className="reshade-wizard-runtime-step">
      <label>
        <span>Rendering API</span>
        <select
          aria-label="Rendering API"
          onChange={(event) => onRenderingAPIChange(event.target.value as RenderingAPI)}
          value={renderingAPI}
        >
          <option value="">Select API</option>
          {apiOptions.map((option) => (
            <option key={option.renderingApi} value={option.renderingApi}>{option.renderingApi}</option>
          ))}
        </select>
      </label>

      <label>
        <span>Proxy filename</span>
        <select
          aria-label="Proxy filename"
          disabled={renderingAPI === ''}
          onChange={(event) => onProxyFilenameChange(event.target.value)}
          value={proxyFilename}
        >
          <option value="">Select proxy</option>
          {proxies.map((proxy) => <option key={proxy} value={proxy}>{proxy}</option>)}
        </select>
      </label>

      <label>
        <span>Build</span>
        <select
          aria-label="Build variant"
          onChange={(event) => onBuildVariantChange(event.target.value as BuildVariant)}
          value={buildVariant}
        >
          <option value={BuildVariant.BuildVariantStandard}>Standard</option>
          <option value={BuildVariant.BuildVariantAddon}>Full add-on</option>
        </select>
      </label>
    </div>
  );
};
