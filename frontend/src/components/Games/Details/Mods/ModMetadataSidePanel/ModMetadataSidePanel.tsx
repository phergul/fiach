import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';

import { RotateCcw, X } from 'lucide-react';

import {
  ModMetadataFieldUpdateMode,
  type Mod,
  type ModMetadata,
  type ModMetadataField,
  type UpdateModMetadataInput,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './ModMetadataSidePanel.scss';

interface ModMetadataSidePanelProps {
  error: string | null;
  isBusy: boolean;
  mod: Mod | null;
  onClose: () => void;
  onSave: (input: ModMetadataSaveInput) => Promise<void> | void;
}

export interface ModMetadataSaveInput {
  modID: number;
  name: string;
  metadata: UpdateModMetadataInput;
}

interface FieldState {
  value: string;
  resetToDetected: boolean;
}

interface FieldErrors {
  name?: string;
  sourceURL?: string;
}

const emptyFieldState = (): FieldState => ({
  value: '',
  resetToDetected: false,
});

const valueFromField = (field: ModMetadataField | undefined) => {
  if (field === undefined) {
    return '';
  }

  return field.UserSet ? field.User ?? '' : field.Effective ?? '';
};

const resetValueFromField = (field: ModMetadataField | undefined) => field?.Detected ?? '';

const hasDetectedValue = (field: ModMetadataField | undefined) => (field?.Detected ?? '').trim() !== '';

const buildFieldState = (field: ModMetadataField | undefined): FieldState => ({
  value: valueFromField(field),
  resetToDetected: false,
});

const buildMetadataUpdateField = (field: FieldState) => ({
  Mode: field.resetToDetected
    ? ModMetadataFieldUpdateMode.ModMetadataFieldUpdateModeReset
    : ModMetadataFieldUpdateMode.ModMetadataFieldUpdateModeUser,
  Value: field.resetToDetected ? null : field.value,
});

const isValidSourceURL = (value: string) => {
  const trimmedValue = value.trim();
  if (trimmedValue === '') {
    return true;
  }

  try {
    const parsedURL = new URL(trimmedValue);
    return parsedURL.protocol === 'http:' || parsedURL.protocol === 'https:';
  } catch {
    return false;
  }
};

const createValidationErrors = (name: string, sourceURL: FieldState): FieldErrors => {
  const errors: FieldErrors = {};
  if (name.trim() === '') {
    errors.name = 'Mod name is required.';
  }
  if (!sourceURL.resetToDetected && !isValidSourceURL(sourceURL.value)) {
    errors.sourceURL = 'Source URL must be an absolute http or https URL.';
  }

  return errors;
};

const hasValidationErrors = (errors: FieldErrors) => Object.keys(errors).length > 0;

const getMetadata = (mod: Mod | null): ModMetadata | null => mod?.Metadata ?? null;

export const ModMetadataSidePanel = ({
  error,
  isBusy,
  mod,
  onClose,
  onSave,
}: ModMetadataSidePanelProps) => {
  const metadata = getMetadata(mod);
  const [name, setName] = useState('');
  const [version, setVersion] = useState<FieldState>(() => emptyFieldState());
  const [author, setAuthor] = useState<FieldState>(() => emptyFieldState());
  const [description, setDescription] = useState<FieldState>(() => emptyFieldState());
  const [sourceURL, setSourceURL] = useState<FieldState>(() => emptyFieldState());
  const [notes, setNotes] = useState('');
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const panelTitle = mod === null ? 'Mod Metadata' : `Edit ${mod.Name}`;

  useEffect(() => {
    setName(mod?.Name ?? '');
    setVersion(buildFieldState(metadata?.Version));
    setAuthor(buildFieldState(metadata?.Author));
    setDescription(buildFieldState(metadata?.Description));
    setSourceURL(buildFieldState(metadata?.SourceURL));
    setNotes(metadata?.Notes ?? '');
    setFieldErrors({});
  }, [metadata, mod]);

  if (mod === null) {
    return null;
  }

  const updateField = (
    setter: (value: FieldState) => void,
    nextValue: string,
  ) => {
    setter({
      value: nextValue,
      resetToDetected: false,
    });
  };

  const resetField = (
    field: ModMetadataField | undefined,
    setter: (value: FieldState) => void,
  ) => {
    setter({
      value: resetValueFromField(field),
      resetToDetected: true,
    });
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const errors = createValidationErrors(name, sourceURL);
    setFieldErrors(errors);
    if (hasValidationErrors(errors)) {
      return;
    }

    await onSave({
      modID: mod.ID,
      name,
      metadata: {
        ModID: mod.ID,
        Version: buildFieldStateForSave(version),
        Author: buildFieldStateForSave(author),
        Description: buildFieldStateForSave(description),
        SourceURL: buildFieldStateForSave(sourceURL),
        Notes: notes.trim() === '' ? null : notes,
      },
    });
  };

  const buildTextInput = (
    label: string,
    field: ModMetadataField | undefined,
    state: FieldState,
    setter: (value: FieldState) => void,
    errorMessage?: string,
  ) => (
    <label className="mod-metadata-side-panel-field">
      <span className="mod-metadata-side-panel-label-row">
        <span className="mod-metadata-side-panel-label">{label}</span>
        <button
          className="mod-metadata-side-panel-reset-button"
          disabled={isBusy}
          onClick={() => resetField(field, setter)}
          aria-label={`Reset ${label.toLowerCase()} to detected value`}
          title={`Reset ${label.toLowerCase()} to detected value`}
          type="button"
        >
          <RotateCcw className="mod-metadata-side-panel-button-icon" aria-hidden="true" />
        </button>
      </span>
      <input
        className="mod-metadata-side-panel-input"
        disabled={isBusy}
        onChange={(event) => updateField(setter, event.target.value)}
        type="text"
        value={state.value}
      />
      {state.resetToDetected && hasDetectedValue(field) && (
        <span className="mod-metadata-side-panel-status">Will reset on save.</span>
      )}
      {errorMessage !== undefined && <span className="mod-metadata-side-panel-error">{errorMessage}</span>}
    </label>
  );

  return (
    <aside className="mod-metadata-side-panel" aria-label="Mod metadata editor">
      <form className="mod-metadata-side-panel-form" onSubmit={handleSubmit}>
        <header className="mod-metadata-side-panel-header">
          <div className="mod-metadata-side-panel-heading">
            <h2 className="mod-metadata-side-panel-title">{panelTitle}</h2>
          </div>
          <button
            className="mod-metadata-side-panel-close-button"
            disabled={isBusy}
            onClick={onClose}
            title="Close metadata editor"
            type="button"
          >
            <X className="mod-metadata-side-panel-close-icon" aria-hidden="true" />
          </button>
        </header>

        <div className="mod-metadata-side-panel-body">
          <label className="mod-metadata-side-panel-field">
            <span className="mod-metadata-side-panel-label">Display name</span>
            <input
              className="mod-metadata-side-panel-input"
              disabled={isBusy}
              onChange={(event) => setName(event.target.value)}
              type="text"
              value={name}
            />
            {fieldErrors.name !== undefined && (
              <span className="mod-metadata-side-panel-error">{fieldErrors.name}</span>
            )}
          </label>

          {buildTextInput('Version', metadata?.Version, version, setVersion)}
          {buildTextInput('Author', metadata?.Author, author, setAuthor)}

          <label className="mod-metadata-side-panel-field">
            <span className="mod-metadata-side-panel-label-row">
              <span className="mod-metadata-side-panel-label">Description</span>
              <button
                className="mod-metadata-side-panel-reset-button"
                disabled={isBusy}
                onClick={() => resetField(metadata?.Description, setDescription)}
                aria-label="Reset description to detected value"
                title="Reset description to detected value"
                type="button"
              >
                <RotateCcw className="mod-metadata-side-panel-button-icon" aria-hidden="true" />
              </button>
            </span>
            <textarea
              className="mod-metadata-side-panel-textarea"
              disabled={isBusy}
              onChange={(event) => updateField(setDescription, event.target.value)}
              value={description.value}
            />
            {description.resetToDetected && hasDetectedValue(metadata?.Description) && (
              <span className="mod-metadata-side-panel-status">Will reset on save.</span>
            )}
          </label>

          {buildTextInput('Source URL', metadata?.SourceURL, sourceURL, setSourceURL, fieldErrors.sourceURL)}

          <label className="mod-metadata-side-panel-field">
            <span className="mod-metadata-side-panel-label">Notes</span>
            <textarea
              className="mod-metadata-side-panel-textarea"
              disabled={isBusy}
              onChange={(event) => setNotes(event.target.value)}
              value={notes}
            />
          </label>

          {error !== null && <div className="mod-metadata-side-panel-save-error">{error}</div>}
        </div>

        <footer className="mod-metadata-side-panel-footer">
          <button
            className="mod-metadata-side-panel-save-button button-main"
            disabled={isBusy}
            type="submit"
          >
            {isBusy ? 'Saving...' : 'Save'}
          </button>
          <button
            className="mod-metadata-side-panel-cancel-button"
            disabled={isBusy}
            onClick={onClose}
            type="button"
          >
            Cancel
          </button>
        </footer>
      </form>
    </aside>
  );
};

const buildFieldStateForSave = buildMetadataUpdateField;
