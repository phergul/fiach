# Mod Manager Project Backlog

Project stack: **Wails v3 + React + TypeScript + SCSS + Go + SQLite**  
Project idea: **General Steam-focused mod profile manager with safe file operations, rollback, and curated support for common mod installation patterns.**

---

## Status Legend

| Status | Meaning |
|---|---|
| Todo | Not started |
| In Progress | Currently being worked on |
| Blocked | Cannot continue until another task/decision is resolved |
| Review | Implemented and needs testing/review |
| Done | Completed |

## Priority Legend

| Priority | Meaning |
|---|---|
| P0 | Required for the project to work |
| P1 | Important for a good MVP |
| P2 | Useful but can wait |
| P3 | Stretch / polish |

---

# Epic 1: Project Setup

## MOD-001 — Initialize Wails v3 Project

**Type:** Task  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Create the base Wails v3 project using React, TypeScript, SCSS, Go, and SQLite.

### Acceptance Criteria

- [ ] Wails v3 app runs locally.
- [ ] React frontend loads successfully.
- [ ] TypeScript is enabled.
- [ ] SCSS support is working.
- [ ] Go backend methods can be called from React.
- [ ] SQLite dependency is installed and usable.

### Notes

Keep Wails-specific code thin so the core backend logic is easy to test separately.

---

## MOD-002 — Create Initial Folder Structure

**Type:** Task  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Create a maintainable project folder structure for frontend, backend, storage, Steam detection, mod operations, and backups.

### Suggested Structure

```text
mod-manager/
├── app.go
├── main.go
├── internal/
│   ├── steam/
│   ├── games/
│   ├── mods/
│   ├── operations/
│   ├── backup/
│   ├── reshade/
│   └── storage/
└── frontend/
    └── src/
        ├── app/
        ├── pages/
        ├── components/
        ├── styles/
        ├── hooks/
        └── types/
```

### Acceptance Criteria

- [ ] Backend code is separated into internal packages.
- [ ] Frontend code has pages, components, styles, hooks, and types directories.
- [ ] No major logic is placed directly in `main.go`.
- [ ] Project can still run after restructuring.

---

## MOD-003 — Set Up App-Wide Styling

**Type:** Task  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Create the base SCSS structure and global layout styling.

### Acceptance Criteria

- [ ] Global SCSS file exists.
- [ ] Variables for spacing, colors, border radius, and font sizes exist.
- [ ] App has a basic dark theme.
- [ ] Layout supports sidebar + main content area.
- [ ] Styling is consistent across initial pages.

---

# Epic 2: SQLite Storage

## MOD-004 — Initialize SQLite Database

**Type:** Task  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Set up SQLite database creation and connection management in Go.

### Acceptance Criteria

- [ ] App creates a local database file on startup.
- [ ] Database path is stored in an appropriate app data directory.
- [ ] Connection can be reused by backend services.
- [ ] Errors are returned clearly if the database cannot open.
- [ ] Database code is isolated in `internal/storage`.

---

## MOD-005 — Add Database Migrations

**Type:** Task  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Implement a simple migration system for creating and updating database tables.

### Acceptance Criteria

- [ ] Migrations run automatically on startup.
- [ ] Already-applied migrations are not rerun.
- [ ] Migration failures stop the app from continuing unsafe operations.
- [ ] A `schema_migrations` table exists.

---

## MOD-006 — Create Core Tables

**Type:** Task  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Create the initial database schema for games, mods, profiles, profile mods, and applied manifests.

### Initial Tables

```sql
games
mods
profiles
profile_mods
applied_manifests
settings
```

### Acceptance Criteria

- [ ] `games` table exists.
- [ ] `mods` table exists.
- [ ] `profiles` table exists.
- [ ] `profile_mods` join table exists.
- [ ] `applied_manifests` table exists.
- [ ] `settings` table exists.
- [ ] Foreign keys are enabled.

---

# Epic 3: Steam Game Detection

## MOD-007 — Locate Steam Installation

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Detect the user's Steam installation directory on Windows.

### Acceptance Criteria

- [ ] App checks common Steam install locations.
- [ ] App can read a manually configured Steam path.
- [ ] App handles missing Steam installation gracefully.
- [ ] App returns a clear error if Steam cannot be found.

---

## MOD-008 — Parse Steam Library Folders

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Read Steam's `libraryfolders.vdf` file to discover all Steam library paths.

### Acceptance Criteria

- [ ] App reads the default Steam library.
- [ ] App detects additional Steam libraries.
- [ ] Invalid or missing `libraryfolders.vdf` does not crash the app.
- [ ] Unit tests exist using fake Steam library data.

---

## MOD-009 — Parse Steam App Manifests

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Parse `appmanifest_*.acf` files to detect installed Steam games.

### Acceptance Criteria

- [ ] App extracts Steam App ID.
- [ ] App extracts game name.
- [ ] App extracts install directory.
- [ ] App resolves full game install path.
- [ ] App ignores incomplete or invalid manifests safely.
- [ ] Unit tests exist using sample app manifests.

---

## MOD-010 — Save Detected Games to SQLite

**Type:** Task  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Store detected Steam games in the database.

### Acceptance Criteria

- [ ] New games are inserted.
- [ ] Existing games are updated.
- [ ] Removed games can be marked missing or unavailable.
- [ ] Duplicate game records are avoided.
- [ ] Scan results are returned to the frontend.

---

# Epic 4: Frontend Game Library

## MOD-011 — Create Main App Layout

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Build the main React layout for the app.

### Acceptance Criteria

- [ ] Sidebar navigation exists.
- [ ] Main content area exists.
- [ ] App title/logo area exists.
- [ ] Navigation supports Library, Profiles, Settings, and Logs.
- [ ] Layout works at common desktop window sizes.

---

## MOD-012 — Build Game Library Page

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Display detected Steam games in the frontend.

### Acceptance Criteria

- [ ] User can trigger a Steam scan.
- [ ] Detected games are shown in a list/grid.
- [ ] Each game displays name and install path.
- [ ] Empty state is shown when no games are found.
- [ ] Error state is shown if scan fails.

---

## MOD-013 — Build Game Details Page

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Create a page for viewing a selected game's details and profiles.

### Acceptance Criteria

- [ ] User can select a game from the library.
- [ ] Page shows game name, path, Steam App ID, and availability.
- [ ] Page shows profiles for that game.
- [ ] Page shows imported mods for that game.
- [ ] User can return to library.

---

# Epic 5: Profile System

## MOD-014 — Create Mod Profile Model

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Implement backend data structures and database logic for mod profiles.

### Acceptance Criteria

- [ ] Profile can be created for a game.
- [ ] Profile can be renamed.
- [ ] Profile can be deleted.
- [ ] Only one profile can be active per game.
- [ ] Profiles are persisted in SQLite.

---

## MOD-015 — Create Profile Management UI

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Allow users to create and manage profiles from the frontend.

### Acceptance Criteria

- [ ] User can create a profile.
- [ ] User can rename a profile.
- [ ] User can delete a profile.
- [ ] User can see which profile is active.
- [ ] UI handles empty profile state.

---

## MOD-016 — Add Mods to Profiles

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Allow mods to be assigned to profiles.

### Acceptance Criteria

- [ ] User can add a mod to a profile.
- [ ] User can remove a mod from a profile.
- [ ] User can enable or disable a mod within a profile.
- [ ] Profile mod list is persisted.
- [ ] Load order field exists, even if drag/drop is added later.

---

# Epic 6: Mod Importing

## MOD-017 — Import Mod Folder

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Allow users to import a mod from a local folder.

### Acceptance Criteria

- [ ] User can select a folder.
- [ ] App copies or registers the mod into managed storage.
- [ ] Mod record is created in SQLite.
- [ ] Mod is associated with a selected game.
- [ ] Import errors are shown clearly.

---

## MOD-018 — Import Mod Archive

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Allow users to import mods from `.zip` archives.

### Acceptance Criteria

- [ ] User can select a `.zip` file.
- [ ] App extracts archive into managed mod storage.
- [ ] App preserves original archive name as metadata.
- [ ] App handles invalid archives gracefully.
- [ ] App prevents path traversal during extraction.

---

## MOD-019 — Create Mod Import Wizard

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Build a frontend wizard to choose how a mod should be installed.

### Install Strategy Options

- Copy to game root
- Copy to specific folder
- Replace files with backup
- BepInEx plugin
- Unreal `.pak`
- ReShade preset
- Custom mapping

### Acceptance Criteria

- [ ] User can choose install strategy.
- [ ] User can choose target path.
- [ ] User can preview detected files.
- [ ] User can save import configuration.
- [ ] Unsupported mods can still be imported manually.

---

# Epic 7: Operation Planner

## MOD-020 — Create Operation Plan Model

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Create a backend model representing planned file operations before applying a profile.

### Operation Types

- Copy
- Replace
- Delete
- Create directory
- Restore backup

### Acceptance Criteria

- [ ] Operation model supports source path.
- [ ] Operation model supports target path.
- [ ] Operation model supports backup path.
- [ ] Operation model supports conflict flag.
- [ ] Operation model can be serialized to JSON for the frontend.

---

## MOD-021 — Generate Operation Plan for Profile

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Given a profile, generate the list of file operations required to apply it.

### Acceptance Criteria

- [ ] App builds operation plan from enabled mods.
- [ ] Plan does not modify files.
- [ ] Missing source files are reported.
- [ ] Missing target folders are reported.
- [ ] Required directory creation operations are included.
- [ ] Plan is returned to frontend for preview.

---

## MOD-022 — Build Operation Preview UI

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Show users exactly what will happen before files are changed.

### Acceptance Criteria

- [ ] Preview lists files to be added.
- [ ] Preview lists files to be replaced.
- [ ] Preview lists folders to be created.
- [ ] Preview lists conflicts and warnings.
- [ ] User must confirm before applying operations.

---

# Epic 8: Safe Apply and Rollback

## MOD-023 — Apply Operation Plan

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Apply a confirmed operation plan to the selected game folder.

### Acceptance Criteria

- [ ] Files are copied to correct locations.
- [ ] Replaced files are backed up first.
- [ ] Operation stops safely on critical failure.
- [ ] Partial failure is reported.
- [ ] Applied operation manifest is generated.
- [ ] Applied manifest is saved to SQLite.

---

## MOD-024 — Create Rollback Manifest

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Record enough information to undo changes made by the app.

### Acceptance Criteria

- [ ] Manifest records files added by the app.
- [ ] Manifest records files replaced by the app.
- [ ] Manifest records backup paths.
- [ ] Manifest records file hashes where possible.
- [ ] Manifest records applied profile and timestamp.

---

## MOD-025 — Restore Vanilla State

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Allow users to undo applied profile changes for a game.

### Acceptance Criteria

- [ ] Added files can be removed.
- [ ] Replaced files can be restored from backup.
- [ ] Missing backups are reported.
- [ ] Hash mismatches are warned about before restore.
- [ ] Game is marked as having no active profile after restore.

---

## MOD-026 — Add Safety Confirmation Dialogs

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Add confirmations before destructive or risky file operations.

### Acceptance Criteria

- [ ] User must confirm profile apply.
- [ ] User must confirm restore vanilla.
- [ ] User must confirm deleting a mod.
- [ ] User must confirm deleting a profile.
- [ ] Dialog clearly explains the consequence.

---

# Epic 9: Conflict Detection

## MOD-027 — Detect File Target Conflicts

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Detect when multiple enabled mods attempt to write to the same target file.

### Acceptance Criteria

- [ ] Same target file is detected as a conflict.
- [ ] Conflict includes all mods involved.
- [ ] Conflicts are shown in operation preview.
- [ ] User cannot apply a profile with unresolved critical conflicts.

---

## MOD-028 — Add Conflict Resolution UI

**Type:** Story  
**Priority:** P2  
**Status:** Todo  
**Scope:** Post-MVP

### Description

Allow users to resolve conflicts by choosing which mod wins.

### Acceptance Criteria

- [ ] User can see conflicting mods.
- [ ] User can choose winning mod.
- [ ] Choice updates operation plan.
- [ ] Resolution is saved for the profile.

---

# Epic 10: Install Strategy Adapters

## MOD-029 — Generic Copy Adapter

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Support mods that copy files or folders into a selected target directory.

### Acceptance Criteria

- [ ] User can set target directory.
- [ ] Files are copied preserving relative paths.
- [ ] Target directory can be inside game folder.
- [ ] Operation planner supports this adapter.

---

## MOD-030 — Replace Files Adapter

**Type:** Story  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Support mods that replace existing game files.

### Acceptance Criteria

- [ ] Existing target files are backed up.
- [ ] Missing target files are treated as warnings or errors.
- [ ] Restore vanilla can undo replacements.
- [ ] Operation preview clearly marks replacements as risky.

---

## MOD-031 — BepInEx Adapter

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Support common BepInEx-style plugin installation.

### Acceptance Criteria

- [ ] App detects existing `BepInEx` folder.
- [ ] App supports installing `.dll` files into `BepInEx/plugins`.
- [ ] App supports installing config files into `BepInEx/config`.
- [ ] App warns if BepInEx is not installed.
- [ ] App can later support assisted BepInEx installation.

---

## MOD-032 — Unreal PAK Adapter

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Support Unreal Engine `.pak`, `.ucas`, and `.utoc` style mods.

### Acceptance Criteria

- [ ] App detects common Unreal `Content/Paks` folder.
- [ ] App supports installing files into `Content/Paks/~mods`.
- [ ] App preserves related `.pak`, `.ucas`, and `.utoc` groups.
- [ ] App warns if target folder cannot be detected.
- [ ] App supports manual override.

---

## MOD-033 — ReShade Preset Adapter

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Support managing ReShade preset files for a selected game.

### Acceptance Criteria

- [ ] App can detect likely game executable.
- [ ] User can select executable manually.
- [ ] User can import `.ini` preset.
- [ ] Preset can be copied to game folder.
- [ ] ReShade files are tracked in profile manifest.
- [ ] App can remove or restore previous preset.

---

# Epic 11: ReShade Assistant

## MOD-034 — Detect ReShade Installation

**Type:** Story  
**Priority:** P2  
**Status:** Todo  
**Scope:** Post-MVP

### Description

Detect whether ReShade appears to be installed for a selected game.

### Acceptance Criteria

- [ ] App checks for common ReShade DLL names.
- [ ] App checks for ReShade preset files.
- [ ] App checks for `reshade-shaders` folder.
- [ ] App reports detected status to frontend.
- [ ] App does not assume ReShade is installed from one file alone.

---

## MOD-035 — Assisted ReShade Install Flow

**Type:** Story  
**Priority:** P2  
**Status:** Todo  
**Scope:** Post-MVP

### Description

Create a guided ReShade installation flow without silently modifying unsupported games.

### Acceptance Criteria

- [ ] User can open official ReShade download page.
- [ ] App shows selected game executable path.
- [ ] App explains graphics API choice must be confirmed by user.
- [ ] App can install/copy preset after ReShade setup.
- [ ] App warns about multiplayer or anti-cheat risks.

---

# Epic 12: Logs and Diagnostics

## MOD-036 — Add Backend Logging

**Type:** Task  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Add structured logging for scan, import, apply, restore, and error events.

### Acceptance Criteria

- [ ] Logs include timestamp.
- [ ] Logs include operation type.
- [ ] Logs include relevant game/profile/mod IDs.
- [ ] Errors include enough detail for debugging.
- [ ] Sensitive local paths are handled carefully in exported logs.

---

## MOD-037 — Build Logs Page

**Type:** Story  
**Priority:** P2  
**Status:** Todo  
**Scope:** Post-MVP

### Description

Show recent app logs in the frontend.

### Acceptance Criteria

- [ ] Logs page exists.
- [ ] User can filter logs by level.
- [ ] User can copy logs.
- [ ] User can clear logs.
- [ ] User can export logs to file.

---

# Epic 13: Settings

## MOD-038 — Add Settings Storage

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Persist app settings in SQLite.

### Acceptance Criteria

- [ ] Settings can be read from database.
- [ ] Settings can be updated.
- [ ] Default settings are created on first launch.
- [ ] Settings are exposed to frontend.

---

## MOD-039 — Build Settings Page

**Type:** Story  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Create a frontend settings page.

### Initial Settings

- Steam path override
- Mod storage path
- Backup storage path
- Theme preference
- Safety confirmations enabled

### Acceptance Criteria

- [ ] User can view settings.
- [ ] User can update paths.
- [ ] User can reset settings to defaults.
- [ ] Invalid paths are rejected or warned about.

---

# Epic 14: Testing

## MOD-040 — Add Steam Scanner Unit Tests

**Type:** Task  
**Priority:** P1  
**Status:** Todo  
**Scope:** MVP

### Description

Test Steam library and app manifest parsing using fixture files.

### Acceptance Criteria

- [ ] Test fixtures exist for `libraryfolders.vdf`.
- [ ] Test fixtures exist for `appmanifest_*.acf`.
- [ ] Tests cover multiple Steam libraries.
- [ ] Tests cover invalid files.
- [ ] Tests run on macOS and Windows.

---

## MOD-041 — Add Operation Planner Tests

**Type:** Task  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Test operation plan generation without modifying real game files.

### Acceptance Criteria

- [ ] Tests cover copy operations.
- [ ] Tests cover replace operations.
- [ ] Tests cover missing source files.
- [ ] Tests cover conflicts.
- [ ] Tests cover directory creation.
- [ ] Tests use temporary directories.

---

## MOD-042 — Add Rollback Tests

**Type:** Task  
**Priority:** P0  
**Status:** Todo  
**Scope:** MVP

### Description

Test restoring files from applied manifests.

### Acceptance Criteria

- [ ] Added files are removed.
- [ ] Replaced files are restored.
- [ ] Missing backup files are reported.
- [ ] Hash mismatch warnings are generated.
- [ ] Partial rollback failure is handled safely.

---

# Epic 15: Packaging and Release

## MOD-043 — Create Windows Build Workflow

**Type:** Task  
**Priority:** P2  
**Status:** Todo  
**Scope:** Post-MVP

### Description

Create a repeatable process for building the Windows version of the app.

### Acceptance Criteria

- [ ] Windows build command is documented.
- [ ] App icon is configured.
- [ ] App metadata is configured.
- [ ] Build output is easy to find.
- [ ] Build runs successfully on Windows machine or VM.

---

## MOD-044 — Create First Internal Test Release

**Type:** Task  
**Priority:** P2  
**Status:** Todo  
**Scope:** Post-MVP

### Description

Create a first test build for personal use on the Windows PC.

### Acceptance Criteria

- [ ] App installs or runs on Windows.
- [ ] App scans real Steam library.
- [ ] App can import a test mod.
- [ ] App can apply and restore a test profile.
- [ ] Known issues are documented.

---

# Epic 16: Future Scope

## MOD-045 — Drag-and-Drop Load Order

**Type:** Story  
**Priority:** P2  
**Status:** Todo  
**Scope:** Future

### Description

Allow users to reorder mods in a profile using drag and drop.

### Acceptance Criteria

- [ ] User can drag mods in profile list.
- [ ] Load order updates visually.
- [ ] Load order is persisted.
- [ ] Operation planner respects load order.

---

## MOD-046 — Community Game Rules

**Type:** Story  
**Priority:** P3  
**Status:** Todo  
**Scope:** Future

### Description

Allow game support rules to be imported/exported as JSON.

### Acceptance Criteria

- [ ] Game rule JSON format is documented.
- [ ] User can import a game rule.
- [ ] User can export a game rule.
- [ ] Invalid rules are rejected safely.
- [ ] Rule versioning exists.

---

## MOD-047 — Mod Metadata Parser

**Type:** Story  
**Priority:** P3  
**Status:** Todo  
**Scope:** Future

### Description

Detect mod metadata from common files such as `manifest.json`, `mod.json`, or loader-specific metadata.

### Acceptance Criteria

- [ ] App checks for known metadata files.
- [ ] App extracts mod name where possible.
- [ ] App extracts version where possible.
- [ ] App falls back to folder/archive name.
- [ ] User can edit detected metadata.

---

## MOD-048 — Nexus Mods Integration Research

**Type:** Research  
**Priority:** P3  
**Status:** Todo  
**Scope:** Future

### Description

Research whether Nexus Mods API integration is feasible and appropriate.

### Acceptance Criteria

- [ ] API requirements are documented.
- [ ] Authentication requirements are documented.
- [ ] Rate limits are documented.
- [ ] Legal/terms considerations are documented.
- [ ] Decision is made whether to include this feature.

---

# MVP Definition

The MVP is complete when the app can:

- [ ] Run as a Wails v3 desktop app.
- [ ] Detect installed Steam games.
- [ ] Display games in a React UI.
- [ ] Create profiles for a selected game.
- [ ] Import a local mod folder.
- [ ] Assign mods to a profile.
- [ ] Preview file operations before applying.
- [ ] Apply a profile safely.
- [ ] Back up replaced files.
- [ ] Restore the game to vanilla state.
- [ ] Store games, mods, profiles, and manifests in SQLite.

---

# Suggested Build Order

1. MOD-001 — Initialize Wails v3 Project
2. MOD-002 — Create Initial Folder Structure
3. MOD-004 — Initialize SQLite Database
4. MOD-005 — Add Database Migrations
5. MOD-006 — Create Core Tables
6. MOD-007 — Locate Steam Installation
7. MOD-008 — Parse Steam Library Folders
8. MOD-009 — Parse Steam App Manifests
9. MOD-010 — Save Detected Games to SQLite
10. MOD-011 — Create Main App Layout
11. MOD-012 — Build Game Library Page
12. MOD-014 — Create Mod Profile Model
13. MOD-015 — Create Profile Management UI
14. MOD-017 — Import Mod Folder
15. MOD-020 — Create Operation Plan Model
16. MOD-021 — Generate Operation Plan for Profile
17. MOD-022 — Build Operation Preview UI
18. MOD-023 — Apply Operation Plan
19. MOD-024 — Create Rollback Manifest
20. MOD-025 — Restore Vanilla State

---

# Open Decisions

## OD-001 — App Name

**Decision Needed:** Choose a project/app name.

Options:

- ModForge
- ModDock
- ProfileForge
- SteamMod Profiles
- GameMod Manager

---

## OD-002 — Storage Location

**Decision Needed:** Decide where managed mods and backups are stored.

Possible default:

```text
%APPDATA%/ModManager/
├── mods/
├── backups/
└── logs/
```

---

## OD-003 — First Supported Game

**Decision Needed:** Choose one real game for initial testing.

Good candidates:

- Hollow Knight
- Risk of Rain 2
- Lethal Company
- Valheim
- Stardew Valley

---

## OD-004 — First Adapter After Generic Copy

**Decision Needed:** Choose which adapter to build first after generic file/folder copy.

Options:

- BepInEx
- Unreal PAK
- ReShade preset
- MelonLoader

---

# Risks

## RISK-001 — File Operation Safety

Changing real game files is risky. The app must always preview operations and create rollback manifests.

## RISK-002 — Game Compatibility

Different games use different modding systems. The app should avoid promising universal automatic support.

## RISK-003 — ReShade and Anti-Cheat

ReShade and similar tools may cause problems with multiplayer or anti-cheat games. The app should show warnings and avoid unsafe automation.

## RISK-004 — Wails v3 Churn

Wails v3 is still alpha. Keep framework-specific code isolated from core business logic.

## RISK-005 — Windows-Specific Testing

The app can be developed partly on macOS, but Steam detection, game paths, ReShade, and real mod operations must be tested on Windows.

---

# Notes for Editing

Use this backlog as a starting point. You can edit ticket IDs, priorities, and scope as the project becomes clearer.

Recommended workflow:

- Keep `P0` small and strict.
- Move uncertain ideas to `P2` or `P3`.
- Do not add game-specific support until the operation planner and rollback system are reliable.
- Treat every real file modification feature as risky until tested with temporary folders.
