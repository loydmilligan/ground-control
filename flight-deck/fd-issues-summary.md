# Flight Deck Issues Summary

## Issues Encountered During LRC+ Editor Development

### 1. TypeScript Strict Mode Conflicts

**Problem**: The serializer and parser tests required all properties of `LyricLine` to be specified, but when adding new properties (isSpoken, isQualitySync, wordStyles), existing test data became invalid.

**Solution**: Updated all test fixtures to include the new required properties with default values.

**FD Impact**: When adding new fields to data models, FD should have a migration strategy or use partial types in tests.

---

### 2. Unused Import/Variable Errors

**Problem**: TypeScript's noUnusedLocals/noUnusedParameters caught unused imports after refactoring (e.g., `ATTRIBUTE_PATTERN` regex, `getLyricsById` in component).

**Solution**: Removed unused imports immediately after refactoring.

**FD Impact**: Could benefit from a pre-commit hook that runs `tsc --noEmit` to catch these before attempting a full build.

---

### 3. File Path Configuration

**Problem**: Initial spec referenced `/data/cache` as the LRC path (for lyric-scroll's Home Assistant context), but user clarified the actual path is `/mnt/c/Users/mmariani/Music/lrc/{artist}_{song_title}.lrc`.

**FD Impact**: Path configuration should be project-specific and documented clearly. FD should surface project-specific defaults in onboarding docs.

---

### 4. Test Framework Setup

**Problem**: No test framework was pre-configured in the scaffolded project, requiring manual setup of Vitest, jsdom, and testing-library.

**Solution**: Created vitest.config.ts, test setup file, and added test scripts to package.json.

**FD Impact**: Project scaffolding templates should include test infrastructure by default, or FD should auto-detect and suggest test setup during onboarding.

---

### 5. Parser/Serializer Round-Trip Issues

**Problem**: The parser used regex-based attribute extraction that consumed text content when attributes were present. The serializer didn't add trailing `|` after attributes, breaking round-trips.

**Solution**: Rewrote parser to use split-based attribute extraction. Added trailing `|` to serializer attribute output.

**FD Impact**: Round-trip tests are critical for any serialization format. FD should flag this as a required test pattern.

---

### 6. API Integration (LRCLIB)

**Problem**: LRCLIB.net API docs were not easily accessible via standard web fetch (returned minimal HTML). Had to use web search and third-party JS wrapper docs to understand API structure.

**Solution**: Used web search to find API endpoint documentation, then implemented based on discovered patterns.

**FD Impact**: API integrations would benefit from a discovery/validation step. FD could maintain a registry of common API patterns.

---

## Recommendations for FD Improvements

1. **Test Infrastructure by Default**: Scaffolded projects should include test setup
2. **Migration Guides**: When data models change, provide migration tooling
3. **Pre-commit Hooks**: Auto-configure TypeScript checks in pre-commit
4. **API Discovery Tool**: Add utility for discovering API endpoints
5. **Round-trip Test Template**: Include serialization round-trip test patterns in templates

---

## Status

- Issues: 6 identified
- Severity: All were Low-Medium (workarounds available)
- No blocking issues encountered
