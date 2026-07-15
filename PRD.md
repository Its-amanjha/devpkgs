# Product Requirements: pkgview

## Purpose

Give developers one fast terminal screen to browse packages installed through Homebrew, npm, and pip.

## Current release

- List globally installed npm and pip packages, Homebrew formulae, and packages recognised by Windows Package Manager.
- Search, inspect package metadata, and switch colour themes.
- Filter packages whose installed and latest versions differ.
- Refresh package lists without restarting.
- Upgrade or remove a selected Homebrew, npm, or pip package after explicit confirmation.

## Next release

- Show action output in the UI.

## Non-goals

- Replacing npm, pip, or Homebrew.
- Managing project-local dependencies.
- Sending package data to a private server.

## Success criteria

- A user can find an installed package in under ten seconds.
- No package is changed without an explicit confirmation.
- Missing package managers do not stop the app from opening.
