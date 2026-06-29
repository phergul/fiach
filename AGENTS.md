# Backend

- Internal exported functions used by `internal/services` must wrap returned errors with a deferred prefix for tracing:
```go
defer func() {
	if err != nil {
		err = fmt.Errorf("parse Steam library folders: %w", err)
	}
}()
```
- If both service and inner layers wrap errors, use distinct messages: service prefixes describe the user operation; storage/helper prefixes describe the lower-level action.
- Nullable/unset storage values use pointers. Treat `nil` as "not set"; do not collapse SQL `NULL` to zero values unless a call site explicitly needs that.
- `internal/storage` is the persistence layer. Group by table or close domain/table cluster when queries cross tables. Do not add repository structs without a concrete need beyond grouping SQL.
- `internal/services` is workflow/domain-oriented, not one service per table. Services expose app-facing operations and coordinate storage/helper packages.
- `internal/services/dto` owns all service-to-frontend generated types. Use mappers for internal -> dto and dto -> internal conversions.
- Keep services thin. Put heavy internal logic in idiomatic packages under `internal` such as `internal/modimport`.
- Place operations on the workflow/domain service that owns the concept, not a table-named service; e.g. `profile_mods` belongs on `ProfileService`, not `ProfileModsService`.
- Service files:
  - `internal/services/mod.go`: `ModService`
  - `internal/services/profile.go`: `ProfileService`
  - `internal/services/profile_mods.go`: `ProfileService` profile-mod membership/state methods
- Expand Go object literals; one-field literals may be inline:
```go
return ApplyResult{
	Success: true,
	Message: "Recovery rollback completed.",
}

return ApplyResult{Success: true} // ok
```

# Frontend

- Desktop app only; no mobile breakpoint.
- Pages only orchestrate routing/data flow. UI logic belongs in components.
- Non-component frontend folders use area/domain subfolders, not flat catch-all packages. Barrels use lowercase section comments followed by that section's exports.
- Components live under `frontend/src/components`.
- Top-level component folders are shared app areas (`Layout`, `Sidebar`, `Navigation`, `Common`) or features/domains (`Games`, `Mods`, `Profiles`).
- Feature folders may nest by specific area, e.g. `Games/Library/GameLibrary`, `Games/Grid/GameGrid`, `Games/Details/GameDetails`, `Games/Details/Metadata/GameDetailsMetadata`.
- Split very large components into smaller same-folder components named for their purpose, e.g. `GameProfilesSectionList`.
- Keep reusable feature components at the nearest shared level: details-only under `Games/Details`, whole-feature under `Games`, cross-feature under `Common`.
- Each component has its own PascalCase directory with matching files:
  - `ComponentName/ComponentName.tsx`
  - `ComponentName/ComponentName.scss`
- Component names describe the component, not the app shell. Avoid prefixes like `AppSidebar`; use `Sidebar`.
- Styling:
  - Root primitive styles only in `frontend/src/styles/_theme.scss`.
  - Variables only in `frontend/src/styles/_variables.scss`.
  - Component-only SCSS variables go at the top of that component's SCSS file.
  - Each styled component has its own scoped SCSS file; omit SCSS files when there are no styles.
  - Use localized class names such as `component-name-style`.
  - Prefer local styles; use globals only when necessary.
  - Avoid arbitrary style numbers; use variables from `_theme.scss` for spacing/sizes/etc.
- Import order: standard library, third-party, local app, SCSS.
```ts
import React from 'react';

import { thirdPartyLibrary } from 'third-party-library';

import { localModule } from './localModule';

import './styles.scss';
```

# Commands

- App: `wails3 task dev`, `wails3 task build`, `wails3 task package`, `wails3 task checks`
- Backend: `go test ./...`, `go vet ./...`, `go fmt ./...`
- Frontend: `cd frontend && bun install`, `cd frontend && bun run test`, `cd frontend && bun run build:dev`, `cd frontend && bun run build`
- Task wrappers: `wails3 task test:frontend`, `wails3 task test:backend`
