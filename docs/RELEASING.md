# Release process / 發布流程

All McpNet releases—patch, minor, or major—use the same bilingual GitHub release process. Publishing a GitHub Release automatically starts the Windows x64 build workflow, which builds the frontend and executable from the release tag, embeds Wails' WebView2 Evergreen bootstrapper, uploads `McpNet-<tag>-windows-x64.exe`, and appends its byte size and SHA-256 to the release description. If a clean Windows machine has no WebView2 Runtime, the user can approve its installation; downloading the runtime itself still requires internet access.

The Windows executable is linked with Go's internal PE linker (`-linkmode=internal`). This preserves the UAC manifest embedded in `app.syso` while avoiding a duplicate default manifest from MinGW's external linker. The local `scripts/build.ps1` and GitHub Actions workflow use the same linker mode and embedded WebView2 strategy.

## Before publishing

1. Choose the next semantic version and update `wails.json`, `app.rc`, `CHANGELOG.md`, and the Chinese, Simplified Chinese, and English download tables in `README.md`.
2. Write the release summary in Traditional Chinese and English. Explain user-visible changes and confirm personal plugins/settings are excluded from the public build.
3. Commit and push the release source to `main`.
4. In GitHub Releases, create a release for the matching `vX.Y.Z` tag from the intended commit and publish it. Do not attach a locally built EXE; the workflow builds from the tag and uploads the official asset.
5. Verify the Actions run succeeds, then confirm the release has one Windows x64 EXE asset and a bilingual description with size and SHA-256.

Do not publish a release from a tag before its source commit and bilingual notes have been pushed. Keep prior published releases intact unless their removal is explicitly requested.
