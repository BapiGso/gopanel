# Vendored Browser Assets

## IBM Carbon

All administration, login, warning, unavailable, and public monitoring pages use
Carbon Web Components **2.67.0**, local IBM Plex fonts, and Carbon SVG icons.
Only the administration and public monitor load Carbon Charts **1.27.27**.
No dependency installation, frontend compiler, or runtime CDN is required.
Each page loads only its component entry modules; shared shell modules are
ui-shell, notification, modal, and button. Go embeds every entry and relative
chunk, so plain `go build` is the entire production build.

From the repository root, run this exact PowerShell command:

```powershell
go run ./tools/vendorcarbon -version v2.67.0 -components breadcrumb,button,checkbox,code-snippet,data-table,file-uploader,inline-loading,link,modal,notification,number-input,overflow-menu,password-input,progress-bar,search,select,tag,tabs,text-input,textarea,tile,toggle,ui-shell -icons add,arrow--down,arrow--left,arrow--right,arrow--up,close,document,download,folder,overflow-menu--vertical,renew,save -header-icons moon,sun -icon-version 11.92.0 -pictograms warning--01 -pictogram-version 12.88.0 -charts-version 1.27.27 -styles-url https://1.www.s81c.com/common/carbon-for-ibm-dotcom/tag/v1/latest/
```

`tools/vendorcarbon` follows relative static imports, exports-from, and literal
dynamic imports. It rejects imports outside the pinned distribution, removes
source-map references, downloads only the requested entries and their graph,
and writes deterministic asset-relative SHA-256 records to
`carbon/v2.67.0/manifest.json`. A repeat with the same inputs must not change
the files or manifest, including Charts, fonts, icons, and pictograms. When intentionally reducing the component or icon list,
remove old files that are no longer recorded in the new manifest.

For an offline replay of already downloaded original sources, append
`-source-cache <directory>` to that same command. Cache entries are named by
the lowercase SHA-256 of the complete source URL; a `.notfound` suffix records
a verified HTTP 404. Missing entries fail instead of falling back to the network.
This transport option does not change the manifest or embedded output bytes.

Sources and transformations:

- Component graph: `https://1.www.s81c.com/common/carbon/web-components/version/v2.67.0/`.
- Theme/grid/Plex helpers: `https://1.www.s81c.com/common/carbon-for-ibm-dotcom/tag/v1/latest/`.
  The probed version-pinned helper URLs returned 404; the documented latest
  endpoint is used and its actual bytes are recorded in the manifest. This
  source is not immutable upstream, so review any future hash changes.
  The helper grid's legacy `bx--` class prefix is rewritten to `cds--` to match
  the Carbon v11 templates. The app supplies the `layer-01` and
  `layer-accent-01` aliases for the helper's generic layer tokens in each theme zone,
  plus the g100 low-contrast notification background tokens on the Carbon layer.
- Plex **6.4.1**: `https://unpkg.com/@ibm/plex@6.4.1/`, Latin1-only Sans light,
  regular and semibold and Mono regular WOFF2. The alternative package CDN refused
  this oversized package. All remote `@font-face` rules in `plex.css` are
  replaced by these four local files with `font-display: swap`; the reset and
  typography rules are retained. Chinese uses system CJK font fallback.
- Icons **11.92.0**: `https://cdn.jsdelivr.net/npm/@carbon/icons@11.92.0/svg/`;
  16px artwork lives in `../icons/carbon/16/` and 20px header artwork in
  `../icons/carbon/20/`. Only a size-specific HTTP 404 permits 32px artwork
  fallback. Go's trusted `icon` function sets the exact requested dimensions,
  scoped `ui-icon` class, `aria-hidden`, and `focusable`.
- Pictograms **12.88.0**: `https://cdn.jsdelivr.net/npm/@carbon/pictograms@12.88.0/`;
  only `warning--01` at 64px in `../icons/carbon/pictograms/`, for empty states.
- Charts **1.27.27**: `https://cdn.jsdelivr.net/npm/@carbon/charts@1.27.27/`;
  `carbon-charts/1.27.27/bundle.umd.js` is self-contained and defines `Charts`.
  No separate d3 dependency is needed. The bundle, `styles.min.css`, and license
  add 810,618 bytes before manifests. Their SHA-256 records appear in both the
  Charts manifest and the main Carbon manifest.

Licenses: Carbon Web Components and icons retain Apache-2.0 licenses at
`carbon/v2.67.0/LICENSE` and `../icons/carbon/LICENSE`. Plex retains the OFL
license at `carbon/fonts/OFL.txt`. The helper license is covered by Carbon's
Apache-2.0 distribution. Charts and pictograms retain their own Apache-2.0
licenses. No full styles package is vendored. Immutable caching applies only to
the versioned Carbon Web Components and Charts directories, not unversioned
font, xterm, or jsVectorMap URLs.

`go test ./assets ./tools/vendorcarbon` verifies the import graph, file hashes,
embedding, component/icon usage, local font URLs, and template asset paths.

## xterm.js

`xterm.json` pins the npm packages used by the WebSSH page, their archive integrity,
and the SHA-256 hashes of the bundled assets. `xterm.LICENSE` preserves the MIT
license and copyright notices from all four packages.

The browser loads local UMD bundles; no npm installation or CDN access is needed
at runtime. Update the core JavaScript, its matching CSS, and all three addons
together. Use the `main` entry from each npm package and verify its archive against
the registry integrity before copying files. Source map references are removed
because maps are not shipped in the embedded assets.

Keep the core, CSS, addons, and manifest versions in sync. Their unchanged UMD
paths are referenced by `assets/term.template`; the versioned Carbon cache policy
does not apply to these unversioned paths. The terminal remembers only host,
port, and username, never a password, and does not auto-connect.

## jsVectorMap

`jsvectormap/metadata.json` pins jsVectorMap **1.7.0**, its world map, and CSS
using SHA-256. The existing Go integrity test remains in place. Map colors are
read from the active Carbon theme; tooltips escape server-supplied names.
