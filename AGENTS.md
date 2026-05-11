# Backend

- Any internal exported functions that are used by a service (internal/services) should always prefix errors returned using a defer instead of manually adding to all returns (for error tracing). Example:
```
defer func() {
    if err != nil {
        err = fmt.Errorf("parse Steam library folders: %w", err)
    }
}()
```
- When both a service and an inner layer wrap errors, the messages should be distinct. Service-level messages should describe the higher-level user operation, while storage/helper-layer messages should describe the lower-level action so errors do not repeat the same prefix twice.

# Frontend

- Everything should be componentised. Pages only orchestrate routing and data flow. UI logic belongs in components.
- Components live under `frontend/src/components`.
- Top-level component folders should represent either:
    - A shared app area, e.g. `Layout`, `Sidebar`, `Navigation`, `Common`
    - A feature/domain, e.g. `Games`, `Mods`, `Profiles`
- Feature/domain folders may contain nested subfolders when components belong to a specific area of that feature. This can be more than one level deep if necessary.
    - Example:
        - `frontend/src/components/Games/Library/GameLibrary/GameLibrary.tsx`
        - `frontend/src/components/Games/Grid/GameGrid/GameGrid.tsx`
        - `frontend/src/components/Games/Details/GameDetails/GameDetails.tsx`
        - `frontend/src/components/Games/Details/Metadata/GameDetailsMetadata/GameDetailsMetadata.tsx`
- Keep reusable feature components at the nearest shared level.
    - If a component is only used by game details, put it under `Games/Details`.
    - If it is reused across the whole games feature, put it directly under `Games`.
    - If it is reused across unrelated features, move it to `Common`.
- Each component must be in its own PascalCase directory with matching `.tsx` and `.scss` files:
    - `ComponentName/ComponentName.tsx`
    - `ComponentName/ComponentName.scss`
- Component names should describe the component itself, not the app shell. Avoid prefixes like `AppSidebar`. use `Sidebar`.
- Styling rules:
    - Root styling is applyed ONLY in the file `frontend/src/styles/_theme.scss`, this is where root styles live. `frontend/src/styles/_variables.scss` is where all variables live.
    - Component-specific SCSS variables used only by one component should be defined at the top of that component's SCSS file.
    - Each component should have its own scss file, and styles should be scoped to that component.
    - Don't use arbitrary class names, use localised `component-name-style`
    - Always prefer using local styles over global styles, and avoid using global styles unless necessary.
    - Don't use arbitrary numbers in styles, use variables instead, and define them in the `_theme.scss` file (e.g. padding, sizes etc).

- Imports should be structured as follows:
  - Standard library imports
  - Third-party imports
  - Local application imports
  - SCSS imports

  ```
  //Like this...

  import React from 'react';
  import { useState } from 'react';

  import { thirdPartyLibrary } from 'third-party-library';

  import { localModule } from './localModule';

  import './styles.scss';

  ```
