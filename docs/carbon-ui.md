# Carbon UI (No Build Step)

Go renders every page using `html/template`. The handler's data is marshalled
once into an escaped `data-page` attribute, and each Alpine component reads
`JSON.parse(this.$el.dataset.page)`. Go data is never interpolated into JavaScript
or Alpine expressions. Existing panel and WebDAV passwords are not included in
the bootstrap; their empty password inputs omit the field on PATCH to preserve
the current credential. Login still POSTs `username` and `password` to the
configured private path and redirects to `/admin/monitor`.

## Local Assets

Carbon Web Components 2.67.0 and their relative module graph, theme/grid helpers,
Latin IBM Plex Sans/Mono fonts, size-specific Carbon icons, and one empty-state
pictogram are vendored and embedded. Carbon Charts 1.27.27 is loaded only by
the two monitor pages: a status donut and CPU, memory, and network history charts.
Alpine, Ace and its included modes, xterm and addons, jsVectorMap, the PWA files,
and referenced images remain local. Each page imports only the Carbon modules
it uses. `assets/css/app.css` contains token-based layout and integration rules,
not a second component library. There are no npm dependencies or production
Node.js steps. See [sources, licenses, and the exact re-vendoring command](../assets/vendor/README.md).

## Shared Integration

- `assets/js/api.js` exposes `$api` and `$filePath` Alpine magics, unwraps the
  API envelope, rejects query strings and off-origin resource paths, and shows
  a session-expired toast for 401 responses.
- `assets/js/cds-alpine.js` sets Web Component properties through `x-cds-model`
  and `x-cds-prop`, listening to the component's actual Carbon events. `$notify`
  uses Carbon toasts and `$ask` resolves the shared Carbon confirmation modal.
- Page components live in `assets/js/pages/`; monitoring retains its existing
  state/cache/map/history logic in `assets/js/observatory.js` as `monitorPage`.
- Private lifecycle helpers abort requests and dispose timers, observers,
  EventSource/WebSocket connections, editors, terminals, and listeners on Alpine
  destruction or `pagehide`. Docker and monitoring stop SSE in hidden tabs.
  Headscale and service polling do not request snapshots while hidden.
- `assets/js/ui-theme.js` applies the saved white/g100 choice before first paint,
  defaulting to g100 independently of the OS. Header and navigation remain in
  a g100 theme zone. Ace, xterm, the world map, and Carbon Charts re-theme with
  the content. Charts are destroyed on modal close and page teardown.
  Map visibility is a page-local preference;
  SSH may retain only its non-secret connection fields.
- Undefined modal/toast overlays and mobile navigation are hidden before Carbon
  upgrades them. Caddyfile highlighting uses Ace's built-in text-mode defaults
  rather than requiring the absent C-style folding module.

## Responsive Pages

The Carbon 2x Grid surrounds all page content. Below 1056px, the navigation
overlays the page behind the hamburger, and links or Escape close it. Forms
stack on small screens, table wrappers scroll locally, row actions use overflow
menus, file breadcrumbs collapse intermediate folders into an overflow menu,
and Carbon modals/toasts accommodate phones. Ace and xterm have `dvh` heights
and a minimum 60vh editing area, resize/orientation observers, and the terminal
provides touch helpers for Esc, Tab, Ctrl and arrows.

Unsupported-platform responses are preserved. Headscale displays the upstream
unsafe-stop limitation, and the Windows restart API remains unsupported. Service
configuration writes save revisions separately from start/reload; autostart
does not change runtime state. Cron still stores schedules in memory, and Docker
deletes do not force-remove running containers or images in use.

## Checks

```powershell
go vet ./...
go test ./...
$env:CGO_ENABLED = '0'
go build -o .tmp/gopanel.exe .
node --test frontend/observatory.test.cjs
```

The Go tests exercise real rendering and JSON round-trips (including hostile
names/titles), credential comparison, omitted-password settings writes, static
caching/PWA rewrites, local assets, the Carbon manifest/import graph, and the
existing APIs. Monitor tests retain cache identity, filtering, pagination,
topology-sensitive map updates, Chinese formatting, SSE batching/visibility,
and bounded cancellable history assertions.
