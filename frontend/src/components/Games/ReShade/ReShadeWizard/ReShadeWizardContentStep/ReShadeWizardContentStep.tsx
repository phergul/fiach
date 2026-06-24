import { type ChangeEvent, useEffect, useMemo, useState } from 'react';

import { ChevronDown, ChevronRight, Search, RefreshCw } from 'lucide-react';

import { BuildVariant, type ContentRequest } from '@bindings/github.com/phergul/fiach/internal/reshade/models';
import type {
  ManagedReShadeContentCatalogue,
  ManagedReShadePresetInspectionResult,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './ReShadeWizardContentStep.scss';

type AddonPackage = NonNullable<ManagedReShadeContentCatalogue>['addons'][number];
type ContentTab = 'effects' | 'addons';
type EffectPackage = NonNullable<ManagedReShadeContentCatalogue>['effects'][number];

interface ReShadeWizardContentStepProps {
  buildVariant: BuildVariant;
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

const effectFileNames = (pkg: EffectPackage) =>
  pkg.effectFiles.map((effect) => effect.trim()).filter((effect) => effect !== '');

const upsertPackageSelection = (content: ContentRequest, id: string, effectFiles?: string[]): ContentRequest => {
  const nextSelection = effectFiles !== undefined && effectFiles.length > 0
    ? { id, effectFiles }
    : { id };
  const currentSelections = content.effectPackages ?? [];
  const hasPackage = currentSelections.some((selection) => selection.id === id);

  return {
    ...content,
    effectPackages: hasPackage
      ? currentSelections.map((selection) => selection.id === id ? nextSelection : selection)
      : [...currentSelections, nextSelection],
  };
};

const removePackageSelection = (content: ContentRequest, id: string): ContentRequest => ({
  ...content,
  effectPackages: (content.effectPackages ?? []).filter((selection) => selection.id !== id),
});

const togglePackage = (content: ContentRequest, id: string, enabled: boolean): ContentRequest =>
  enabled ? upsertPackageSelection(content, id) : removePackageSelection(content, id);

const toggleEffect = (
  content: ContentRequest,
  pkg: EffectPackage,
  effect: string,
  enabled: boolean,
): ContentRequest => {
  const effects = effectFileNames(pkg);
  const selection = selectedPackage(content, pkg.id);
  const selectedEffects = selection?.effectFiles !== undefined && selection.effectFiles.length > 0
    ? selection.effectFiles
    : effects;
  const nextEffects = enabled
    ? [...selectedEffects, effect]
    : selectedEffects.filter((item) => item !== effect);
  const uniqueEffects = effects.filter((item) => nextEffects.includes(item));

  if (uniqueEffects.length === 0) {
    return removePackageSelection(content, pkg.id);
  }

  if (uniqueEffects.length === effects.length) {
    return upsertPackageSelection(content, pkg.id);
  }

  return upsertPackageSelection(content, pkg.id, uniqueEffects);
};

const selectRecommendation = (
  content: ContentRequest,
  recommendation: ManagedReShadePresetInspectionResult['recommendations'][number],
): ContentRequest =>
  recommendation.effectFiles.length > 0
    ? upsertPackageSelection(content, recommendation.packageId, recommendation.effectFiles)
    : upsertPackageSelection(content, recommendation.packageId);

const toggleAddon = (content: ContentRequest, id: string, enabled: boolean): ContentRequest => ({
  ...content,
  addons: enabled
    ? [...(content.addons ?? []), { id }]
    : (content.addons ?? []).filter((selection) => selection.id !== id),
});

const searchableText = (parts: string[]) => parts.join(' ').toLowerCase();

const packageMatchesSearch = (pkg: EffectPackage, query: string) =>
  query === '' || searchableText([
    pkg.id,
    pkg.name,
    pkg.description,
    pkg.repositoryUrl,
    pkg.downloadUrl,
    ...pkg.effectFiles,
  ]).includes(query);

const addonMatchesSearch = (addon: AddonPackage, query: string) =>
  query === '' || searchableText([
    addon.id,
    addon.name,
    addon.description,
    addon.repositoryUrl,
    addon.downloadUrl ?? '',
    addon.downloadUrl32 ?? '',
    addon.downloadUrl64 ?? '',
  ]).includes(query);

const selectedEffectCount = (content: ContentRequest, pkg: EffectPackage) => {
  const selection = selectedPackage(content, pkg.id);
  if (selection === null) {
    return 0;
  }
  if (selection.effectFiles === undefined || selection.effectFiles.length === 0) {
    return effectFileNames(pkg).length;
  }
  return selection.effectFiles.length;
};

const firstAvailableTab = (showAddons: boolean): ContentTab => showAddons ? 'effects' : 'effects';

export const ReShadeWizardContentStep = ({
  buildVariant,
  catalogue,
  content,
  inspection,
  isInspectingPreset,
  onContentChange,
  onInspectPreset,
  onRefreshCatalogue,
  presetPath,
  setPresetPath,
}: ReShadeWizardContentStepProps) => {
  const effects = catalogue?.effects ?? [];
  const addons = catalogue?.addons ?? [];
  const showAddons = buildVariant === BuildVariant.BuildVariantAddon;
  const [activeTab, setActiveTab] = useState<ContentTab>(firstAvailableTab(showAddons));
  const [activeEffectID, setActiveEffectID] = useState('');
  const [activeAddonID, setActiveAddonID] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [isEffectsOpen, setIsEffectsOpen] = useState(true);
  const [isPresetHelperOpen, setIsPresetHelperOpen] = useState(false);
  const trimmedSearchQuery = searchQuery.trim().toLowerCase();
  const filteredEffects = useMemo(
    () => effects.filter((pkg) => packageMatchesSearch(pkg, trimmedSearchQuery)),
    [effects, trimmedSearchQuery],
  );
  const filteredAddons = useMemo(
    () => addons.filter((addon) => addonMatchesSearch(addon, trimmedSearchQuery)),
    [addons, trimmedSearchQuery],
  );
  const activeEffect = trimmedSearchQuery === ''
    ? effects.find((pkg) => pkg.id === activeEffectID) ?? effects[0] ?? null
    : filteredEffects.find((pkg) => pkg.id === activeEffectID) ?? filteredEffects[0] ?? null;
  const activeAddon = trimmedSearchQuery === ''
    ? addons.find((addon) => addon.id === activeAddonID) ?? addons[0] ?? null
    : filteredAddons.find((addon) => addon.id === activeAddonID) ?? filteredAddons[0] ?? null;
  const activeType = activeTab === 'addons' && showAddons ? 'addon' : 'effect';
  const hasAddons = showAddons && addons.length > 0;
  const currentSelection = activeEffect === null ? null : selectedPackage(content, activeEffect.id);
  const currentEffectFiles = activeEffect === null ? [] : effectFileNames(activeEffect);
  const currentSelectedEffects = currentSelection === null
    ? []
    : currentSelection.effectFiles !== undefined && currentSelection.effectFiles.length > 0
      ? currentSelection.effectFiles
      : currentEffectFiles;

  useEffect(() => {
    if (!showAddons && activeTab === 'addons') {
      setActiveTab('effects');
    }
  }, [activeTab, showAddons]);

  useEffect(() => {
    if (effects.length > 0 && !effects.some((pkg) => pkg.id === activeEffectID)) {
      setActiveEffectID(effects[0].id);
    }
  }, [activeEffectID, effects]);

  useEffect(() => {
    if (addons.length > 0 && !addons.some((addon) => addon.id === activeAddonID)) {
      setActiveAddonID(addons[0].id);
    }
  }, [activeAddonID, addons]);

  const chooseTab = (tab: ContentTab) => {
    setActiveTab(tab);
  };

  const chooseEffectPackage = (id: string) => {
    setActiveTab('effects');
    setActiveEffectID(id);
  };

  const chooseAddon = (id: string) => {
    setActiveTab('addons');
    setActiveAddonID(id);
  };

  const changePackageSelection = (
    event: ChangeEvent<HTMLInputElement>,
    pkg: EffectPackage,
  ) => {
    event.stopPropagation();
    chooseEffectPackage(pkg.id);
    onContentChange(togglePackage(content, pkg.id, event.target.checked));
  };

  const changeAddonSelection = (
    event: ChangeEvent<HTMLInputElement>,
    addon: AddonPackage,
  ) => {
    event.stopPropagation();
    chooseAddon(addon.id);
    onContentChange(toggleAddon(content, addon.id, event.target.checked));
  };

  const changeEffectSelection = (
    event: ChangeEvent<HTMLInputElement>,
    effect: string,
  ) => {
    event.stopPropagation();
    if (activeEffect !== null) {
      onContentChange(toggleEffect(content, activeEffect, effect, event.target.checked));
    }
  };

  const selectAllEffects = () => {
    if (activeEffect !== null) {
      onContentChange(upsertPackageSelection(content, activeEffect.id));
    }
  };

  const clearAllEffects = () => {
    if (activeEffect !== null) {
      onContentChange(removePackageSelection(content, activeEffect.id));
    }
  };

  return (
    <div className="reshade-wizard-content reshade-wizard-content-layout">
      <div className="reshade-wizard-content-step">
        <div className="reshade-wizard-content-browser">
          <aside className="reshade-wizard-content-sidebar" aria-label="ReShade content catalogue">
            <div className="reshade-wizard-content-search-container">
              <div className="reshade-wizard-content-search">
                <Search className="reshade-wizard-content-search-icon" aria-hidden="true" />
                <input
                  aria-label="Search ReShade content"
                  onChange={(event) => setSearchQuery(event.target.value)}
                  placeholder="Search content"
                  type="search"
                  value={searchQuery}
                />
              </div>
              <button
                aria-label="Refresh catalogue"
                onClick={onRefreshCatalogue}
                type="button"
                className="reshade-wizard-content-refresh-button"
              >
                <RefreshCw aria-hidden="true" />
              </button>
            </div>

            {showAddons && (
              <div className="reshade-wizard-content-tabs" role="tablist" aria-label="Content type">
                <button
                  aria-selected={activeTab === 'effects'}
                  onClick={() => chooseTab('effects')}
                  role="tab"
                  type="button"
                >
                  Effect packages
                </button>
                <button
                  aria-selected={activeTab === 'addons'}
                  disabled={!hasAddons}
                  onClick={() => chooseTab('addons')}
                  role="tab"
                  type="button"
                >
                  Add-ons
                </button>
              </div>
            )}

            <div className="reshade-wizard-content-list">
              {(activeTab === 'effects' || !showAddons) && (
                filteredEffects.length === 0 ? (
                  <p className="reshade-wizard-content-empty">No effect packages match this search.</p>
                ) : filteredEffects.map((pkg) => (
                  <div
                    className={activeType === 'effect' && activeEffect?.id === pkg.id
                      ? 'reshade-wizard-content-row reshade-wizard-content-row-active'
                      : 'reshade-wizard-content-row'}
                    key={pkg.id}
                    onClick={() => chooseEffectPackage(pkg.id)}
                  >
                    <label
                      className="reshade-wizard-content-option"
                      onClick={(event) => event.stopPropagation()}
                    >
                      <input
                        aria-label={`Select ${pkg.name}`}
                        checked={selectedPackage(content, pkg.id) !== null}
                        onChange={(event) => changePackageSelection(event, pkg)}
                        onClick={(event) => event.stopPropagation()}
                        type="checkbox"
                      />
                      <span className="reshade-wizard-content-option-control" aria-hidden="true" />
                    </label>
                    <button
                      className="reshade-wizard-content-row-button"
                      onClick={() => chooseEffectPackage(pkg.id)}
                      type="button"
                    >
                      <span className="reshade-wizard-content-row-title">{pkg.name}</span>
                      <span className="reshade-wizard-content-row-description">{pkg.description || pkg.id}</span>
                    </button>
                  </div>
                ))
              )}

              {activeTab === 'addons' && showAddons && (
                filteredAddons.length === 0 ? (
                  <p className="reshade-wizard-content-empty">No add-ons match this search.</p>
                ) : filteredAddons.map((addon) => (
                  <div
                    className={activeType === 'addon' && activeAddon?.id === addon.id
                      ? 'reshade-wizard-content-row reshade-wizard-content-row-active'
                      : 'reshade-wizard-content-row'}
                    key={addon.id}
                    onClick={() => chooseAddon(addon.id)}
                  >
                    <label
                      className="reshade-wizard-content-option"
                      onClick={(event) => event.stopPropagation()}
                    >
                      <input
                        aria-label={`Select ${addon.name}`}
                        checked={selectedAddon(content, addon.id)}
                        onChange={(event) => changeAddonSelection(event, addon)}
                        onClick={(event) => event.stopPropagation()}
                        type="checkbox"
                      />
                      <span className="reshade-wizard-content-option-control" aria-hidden="true" />
                    </label>
                    <button
                      className="reshade-wizard-content-row-button"
                      onClick={() => chooseAddon(addon.id)}
                      type="button"
                    >
                      <span className="reshade-wizard-content-row-title">{addon.name}</span>
                      <span className="reshade-wizard-content-row-description">{addon.description || addon.id}</span>
                    </button>
                  </div>
                ))
              )}
            </div>
          </aside>

          <section className="reshade-wizard-content-details" aria-label="Selected ReShade content">
            {catalogue === null ? (
              <p className="reshade-wizard-content-empty">No catalogue is loaded.</p>
            ) : activeType === 'effect' && activeEffect !== null ? (
              <>
                <div className="reshade-wizard-content-details-header">
                  <div>
                    <h3>{activeEffect.name}</h3>
                    <p>{activeEffect.description || activeEffect.id}</p>
                  </div>
                  <button className="reshade-wizard-content-preset-button" onClick={() => setIsPresetHelperOpen((current) => !current)} type="button">
                    Preset helper
                  </button>
                </div>

                {isPresetHelperOpen && (
                  <section className="reshade-wizard-preset-helper" aria-label="Preset helper">
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
                        {inspection.missingEffects.length > 0 && (
                          <p>{inspection.missingEffects.length} missing effects</p>
                        )}
                        {inspection.warnings.map((warning) => (
                          <p key={warning}>{warning}</p>
                        ))}
                        {inspection.recommendations.map((recommendation) => (
                          <button
                            key={recommendation.packageId}
                            onClick={() => {
                              onContentChange(selectRecommendation(content, recommendation));
                              setActiveTab('effects');
                              setActiveEffectID(recommendation.packageId);
                            }}
                            type="button"
                          >
                            Add {recommendation.packageName}
                          </button>
                        ))}
                      </div>
                    )}
                  </section>
                )}

                <dl className="reshade-wizard-content-meta">
                  <div><dt>Install path</dt><dd>{activeEffect.installPath || '.'}</dd></div>
                  <div><dt>Textures</dt><dd>{activeEffect.textureInstallPath || 'Default'}</dd></div>
                  <div><dt>Source</dt><dd>{activeEffect.repositoryUrl || activeEffect.downloadUrl || 'Catalogue'}</dd></div>
                </dl>

                <section className="reshade-wizard-content-effect-panel" aria-label="Package effects">
                  <button
                    aria-expanded={isEffectsOpen}
                    className="reshade-wizard-content-fold"
                    onClick={() => setIsEffectsOpen((current) => !current)}
                    type="button"
                  >
                    {isEffectsOpen ? <ChevronDown aria-hidden="true" /> : <ChevronRight aria-hidden="true" />}
                    <span>Effects</span>
                    <span>{selectedEffectCount(content, activeEffect)} / {currentEffectFiles.length}</span>
                  </button>

                  {isEffectsOpen && (
                    <div className="reshade-wizard-content-effect-body">
                      <div className="reshade-wizard-content-effect-actions">
                        <button
                          disabled={!activeEffect.modifiable || currentEffectFiles.length === 0}
                          onClick={selectAllEffects}
                          type="button"
                        >
                          Select all effects
                        </button>
                        <button
                          disabled={!activeEffect.modifiable || currentSelection === null}
                          onClick={clearAllEffects}
                          type="button"
                        >
                          Clear all effects
                        </button>
                      </div>

                      {currentEffectFiles.length === 0 ? (
                        <p className="reshade-wizard-content-empty">No effect files are listed for this package.</p>
                      ) : (
                        <div className="reshade-wizard-content-effects">
                          {currentEffectFiles.map((effect) => {
                            const isEffectDisabled = !activeEffect.modifiable || currentSelection === null;

                            return (
                              <label
                                className={isEffectDisabled ? 'reshade-wizard-content-effect-disabled' : undefined}
                                key={effect}
                              >
                              <input
                                checked={currentSelectedEffects.includes(effect)}
                                disabled={isEffectDisabled}
                                onChange={(event) => changeEffectSelection(event, effect)}
                                onClick={(event) => event.stopPropagation()}
                                type="checkbox"
                              />
                              <span className="reshade-wizard-content-option-control" aria-hidden="true" />
                              <span>{effect}</span>
                            </label>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  )}
                </section>
              </>
            ) : activeType === 'addon' && activeAddon !== null ? (
              <>
                <div className="reshade-wizard-content-details-header">
                  <div>
                    <h3>{activeAddon.name}</h3>
                    <p>{activeAddon.description || activeAddon.id}</p>
                  </div>
                </div>

                <dl className="reshade-wizard-content-meta">
                  <div><dt>Install path</dt><dd>{activeAddon.effectInstallPath || 'Addons'}</dd></div>
                  <div><dt>Source</dt><dd>{activeAddon.repositoryUrl || activeAddon.downloadUrl || 'Catalogue'}</dd></div>
                </dl>
              </>
            ) : (
              <p className="reshade-wizard-content-empty">No content is available.</p>
            )}
          </section>
        </div>
      </div>
    </div>
  );
};
