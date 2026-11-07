# Vendored xterm.js

`xterm.json` pins the npm packages used by the WebSSH page, their archive integrity,
and the SHA-256 hashes of the bundled assets. `xterm.LICENSE` preserves the MIT
license and copyright notices from all four packages.

The browser loads local UMD bundles; no npm installation or CDN access is needed
at runtime. Update the core JavaScript, its matching CSS, and all three addons
together. Use the `main` entry from each npm package and verify its archive against
the registry integrity before copying files. Source map references are removed
because maps are not shipped in the embedded assets.

The version query strings in `assets/term.template` prevent browsers from mixing
cached old resources with the new bundles. Keep those versions and the manifest
in sync. Run `node --test tests/xterm.test.cjs` after updating.
