import type { ContentRequest } from '@bindings/github.com/phergul/fiach/internal/reshade/models';
import type { ManagedReShadeContentCatalogue, ManagedReShadePresetInspectionResult } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './ReShadeWizardContentStep.scss';

interface ReShadeWizardContentStepProps {
  catalogue: ManagedReShadeContentCatalogue | null;
  content: ContentRequest;
  inspection: ManagedReShadePresetInspectionResult | null;
  isInspectingPreset: boolean;
  onContentChange: (content: ContentRequest) => void;
  onInspectPreset: (path: string) => void;
  onRefreshCatalogue: () => void;
  presetPath: string;
  setPresetPath: (path: string) => void;
}

const selectedPackage = (content: ContentRequest, id: string) =>
  content.effectPackages?.find((selection) => selection.id === id) ?? null;

const selectedAddon = (content: ContentRequest, id: string) =>
  content.addons?.some((selection) => selection.id === id) ?? false;

const togglePackage = (content: ContentRequest, id: string, enabled: boolean): ContentRequest => ({
  ...content,
  effectPackages: enabled
    ? [...(content.effectPackages ?? []), { id }]
    : (content.effectPackages ?? []).filter((selection) => selection.id !== id),
});

const toggleEffect = (content: ContentRequest, packageID: string, effect: string, enabled: boolean): ContentRequest => ({
  ...content,
  effectPackages: (content.effectPackages ?? []).map((selection) => {
    if (selection.id !== packageID) {
      return selection;
    }
    const current = selection.effectFiles ?? [];
    return {
      ...selection,
      effectFiles: enabled
        ? [...current, effect]
        : current.filter((item) => item !== effect),
    };
  }),
});

const toggleAddon = (content: ContentRequest, id: string, enabled: boolean): ContentRequest => ({
  ...content,
  addons: enabled
    ? [...(content.addons ?? []), { id }]
    : (content.addons ?? []).filter((selection) => selection.id !== id),
});

export const ReShadeWizardContentStep = ({
  catalogue,
  content,
  inspection,
  isInspectingPreset,
  onContentChange,
  onInspectPreset,
  onRefreshCatalogue,
  presetPath,
  setPresetPath,
}: ReShadeWizardContentStepProps) => (
  <div className="reshade-wizard-content">
    <div className="reshade-wizard-content-step">
      <div className="reshade-wizard-content-toolbar">
        <button onClick={onRefreshCatalogue} type="button">Refresh catalogue</button>
      </div>

      <section className="reshade-wizard-content-section" aria-labelledby="reshade-effects-heading">
        <h3 id="reshade-effects-heading">Effect packages</h3>
        {catalogue === null || catalogue.effects.length === 0 ? (
          <p>No effect packages are available.</p>
        ) : catalogue.effects.map((pkg) => {
          const selection = selectedPackage(content, pkg.id);
          const isSelected = selection !== null;
          return (
            <div className="reshade-wizard-content-item" key={pkg.id}>
              <label>
                <input
                  checked={isSelected}
                  onChange={(event) => onContentChange(togglePackage(content, pkg.id, event.target.checked))}
                  type="checkbox"
                />
                <span>{pkg.name}</span>
              </label>
              {pkg.description !== '' && <p>{pkg.description}</p>}
              {isSelected && pkg.effectFiles.length > 0 && pkg.modifiable && (
                <div className="reshade-wizard-content-effects">
                  {pkg.effectFiles.map((effect) => (
                    <label key={effect}>
                      <input
                        checked={(selection.effectFiles ?? []).includes(effect)}
                        onChange={(event) =>
                          onContentChange(toggleEffect(content, pkg.id, effect, event.target.checked))}
                        type="checkbox"
                      />
                      <span>{effect}</span>
                    </label>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </section>

      <section className="reshade-wizard-content-section" aria-labelledby="reshade-addons-heading">
        <h3 id="reshade-addons-heading">Add-ons</h3>
        {catalogue === null || catalogue.addons.length === 0 ? (
          <p>No add-ons are available.</p>
        ) : catalogue.addons.map((addon) => (
          <div className="reshade-wizard-content-item" key={addon.id}>
            <label>
              <input
                checked={selectedAddon(content, addon.id)}
                onChange={(event) => onContentChange(toggleAddon(content, addon.id, event.target.checked))}
                type="checkbox"
              />
              <span>{addon.name}</span>
            </label>
            {addon.description !== '' && <p>{addon.description}</p>}
          </div>
        ))}
      </section>

      <section className="reshade-wizard-content-section" aria-labelledby="reshade-preset-heading">
        <h3 id="reshade-preset-heading">Preset inspection</h3>
        <div className="reshade-wizard-preset-form">
          <input
            aria-label="Preset path"
            onChange={(event) => setPresetPath(event.target.value)}
            placeholder="ReShadePreset.ini"
            value={presetPath}
          />
          <button
            disabled={isInspectingPreset || presetPath.trim() === ''}
            onClick={() => onInspectPreset(presetPath)}
            type="button"
          >
            {isInspectingPreset ? 'Inspecting' : 'Inspect'}
          </button>
        </div>
        {inspection !== null && (
          <div className="reshade-wizard-preset-result">
            <p>{inspection.referencedEffects.length} referenced effects</p>
            {inspection.recommendations.map((recommendation) => (
              <button
                key={recommendation.packageId}
                onClick={() => onContentChange(togglePackage(content, recommendation.packageId, true))}
                type="button"
              >
                Add {recommendation.packageName}
              </button>
            ))}
          </div>
        )}
      </section>
    </div>
  </div>
);
