# Product Requirements: pkgview

## Purpose

Give developers one fast terminal screen to browse packages installed through Homebrew, npm, and pip.

## Current release

- List globally installed npm and pip packages, plus Homebrew formulae.
- Search, inspect package metadata, and switch colour themes.
- Remain read-only: no package install, update, or removal commands.

## Next release

- Add an explicit package-action screen for update and removal.
- Show the exact command and require confirmation before running it.
- Report command success or failure in the UI.

## Non-goals

- Replacing npm, pip, or Homebrew.
- Managing project-local dependencies.
- Sending package data to a private server.

## Success criteria

- A user can find an installed package in under ten seconds.
- No package is changed without an explicit confirmation.
- Missing package managers do not stop the app from opening.
