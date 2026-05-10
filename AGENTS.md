# Frontend

- Every thing should be componentised, pages are only orchestrating, while all logic is in components.
- Components and their files are name in PascalCase.tsx and each will have a scss file with the same name, for example: `ComponentName.tsx` and `ComponentName.scss`.
- Styling rules:
    - Root styling is applyed ONLY in the file `frontend/src/styles/_theme.scss`, this is where root styles live. `frontend/src/styles/_variables.scss` is where all variables live.
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
