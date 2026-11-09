const assert = require('node:assert/strict');
const filesystem = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');
const source = filesystem.readFileSync('assets/js/observatory.js', 'utf8');

function observatory() {
    const themeTokens = {
        g100: {'--cds-layer-accent-01': '#393939', '--cds-border-subtle': '#393939', '--cds-layer-hover': '#333333', '--cds-layer-01': '#262626', '--cds-data-viz-cyan': '#33b1ff', '--cds-data-viz-teal': '#08bdba'},
        white: {'--cds-layer-accent-01': '#e0e0e0', '--cds-border-subtle': '#e0e0e0', '--cds-layer-hover': '#e8e8e8', '--cds-layer-01': '#f4f4f4', '--cds-data-viz-cyan': '#1192e8', '--cds-data-viz-teal': '#009d9a'},
    };
    const listeners = new Map();
    const frames = new Map();
    const watchers = new Map();
    const streams = [];
    const maps = [];
    const charts = [];
    const element = () => ({
        offsetWidth: 320, offsetHeight: 288, children: [], parentElement: null, removed: false,
        replaceChildren(...children) {
            for (const child of this.children) child.parentElement = null;
            this.children = children;
            for (const child of children) child.parentElement = this;
        },
        remove() {
            this.removed = true;
            if (this.parentElement) this.parentElement.children = this.parentElement.children.filter(child => child !== this);
            this.parentElement = null;
        },
    });
    const chartClass = kind => class {
        constructor(element, configuration) {
            this.kind = kind;
            this.element = element;
            this.options = configuration.options;
            this.data = configuration.data;
            this.updates = 0;
            this.model = {setData: data => { this.data = data; this.updates++; }};
            charts.push(this);
        }
        destroy() { this.destroyed = true; this.element.remove(); }
    };
    const globalListeners = new Map();
    let frameID = 0;
    let factory;
    const document = {
        hidden: false, title: '', documentElement: {dataset: {carbonTheme: 'g100'}},
        createElement: element,
        addEventListener(name, handler) { listeners.set(name, handler); },
        removeEventListener(name) { listeners.delete(name); },
    };
    const context = {
        document, console, AbortController,
        addEventListener(name, handler) { globalListeners.set(name, handler); },
        removeEventListener(name) { globalListeners.delete(name); },
        getComputedStyle() { return {getPropertyValue(name) { const tokens = themeTokens[document.documentElement.dataset.carbonTheme]; assert.ok(Object.hasOwn(tokens, name), `Unstubbed Carbon token: ${name}`); return tokens[name]; }}; },
        Charts: {DonutChart: chartClass('donut'), AreaChart: chartClass('area'), LineChart: chartClass('line')},
        Alpine: {data(name, create) { factory = create; }, reactive(value) { return value; }},
        localStorage: {getItem() { return null; }, setItem() {}},
        requestAnimationFrame(callback) { const identity = ++frameID; frames.set(identity, callback); return identity; },
        cancelAnimationFrame(identity) { frames.delete(identity); },
        EventSource: class { constructor() { streams.push(this); } close() { this.closed = true; } },
        ResizeObserver: class { constructor(callback) { this.callback = callback; } observe() {} disconnect() {} },
        jsVectorMap: class {
            constructor(options) {
                this.options = options;
                this.calls = {add: 0, remove: 0, resize: 0};
                this.attributes = [];
                this.series = {regions: [{setAttributes: values => this.attributes.push(values)}]};
                maps.push(this);
            }
            updateSize() { this.calls.resize++; }
            addMarkers(markers) { this.markers = markers; this.calls.add++; }
            removeMarkers() { this.markers = []; this.calls.remove++; }
            destroy() { this.destroyed = true; }
        },
    };
    vm.runInNewContext(source, context);
    listeners.get('alpine:init')();
    const data = factory();
    data.$el = {dataset: {page: JSON.stringify({title: 'Test', api_base: '/api/v1/public/monitor', public: true})}};
    data.$refs = {map: {offsetWidth: 1200, offsetHeight: 500, replaceChildren() {}}, search: {focus() {}}, statusChart: element(), cpuHistory: element(), memoryHistory: element(), networkHistory: element()};
    data.$watch = (name, handler) => watchers.set(name, handler);
    data.$nextTick = callback => callback();
    data.init();
    return {
        data, document, streams, maps, charts, frames, listeners, globalListeners,
        flush() { const pending = [...frames.values()]; frames.clear(); pending.forEach(callback => callback()); },
        filter(name, value) { data[name] = value; watchers.get(name)(); },
        visibility(hidden) { document.hidden = hidden; listeners.get('visibilitychange')(); },
    };
}

function overview(count = 120, step = 0) {
    const nodes = Array.from({length: count}, (_, index) => ({
        id: `node-${index}`, name: `Probe-${index}`, country: index % 2 ? 'JP' : 'DE', region: '', tags: [index % 2 ? 'Relay' : 'Service'],
        latitude: index % 2 ? 35.68 : 50.11, longitude: index % 2 ? 139.69 : 8.68, online: true,
        metrics: {cpu_percent: index + step, memory_percent: 20, upload_rate: 100 + step, download_rate: 200, unavailable_fields: []},
    }));
    return {title: 'Test', nodes, total: count, online: count, offline: 0, cluster: {coordination: 'manual', peer_count: count, healthy_peers: count}};
}

test('metric-only snapshots preserve nodes, rows and map markers', () => {
    const view = observatory();
    view.data.ingest(overview());
    view.flush();
    const nodes = view.data.overview.nodes;
    const filtered = view.data.filteredNodes;
    const page = view.data.pageNodes;
    const tags = view.data.tags;
    const node = nodes[0];
    const metrics = node.metrics;
    const map = view.maps[0];
    const calls = {...map.calls};
    view.data.ingest(overview(120, 3));
    view.flush();
    assert.equal(view.data.overview.nodes, nodes);
    assert.equal(view.data.filteredNodes, filtered);
    assert.equal(view.data.pageNodes, page);
    assert.equal(view.data.tags, tags);
    assert.equal(nodes[0], node);
    assert.equal(node.metrics, metrics);
    assert.equal(node.metrics.cpu_percent, 3);
    assert.deepEqual(map.calls, calls);
    assert.equal('_map' in view.data, false);
    view.data.destroy();
});

test('pagination limits DOM rows without truncating the map or totals', () => {
    const view = observatory();
    view.data.ingest(overview());
    view.flush();
    assert.equal(view.data.overview.total, 120);
    assert.equal(view.data.filteredNodes.length, 120);
    assert.equal(view.data.pageNodes.length, 50);
    assert.equal(view.data.pageCount, 3);
    assert.equal(view.maps[0].markers.length, 120);
    view.data.changePage(2);
    assert.equal(view.data.pageNodes.length, 20);
    assert.equal(view.data.pageNodes[0].id, 'node-100');
    view.filter('selectedTag', 'Relay');
    view.flush();
    assert.equal(view.data.page, 1);
    assert.equal(view.data.filteredNodes.length, 60);
    assert.equal(view.maps[0].markers.length, 60);
    view.data.destroy();
});

test('country, search, sort and topology changes update cached views', () => {
    const view = observatory();
    view.data.ingest(overview());
    view.flush();
    view.filter('selectedCountry', 'JP');
    view.flush();
    assert.equal(view.data.filteredNodes.length, 60);
    assert.equal(view.data.visibleCountries, 1);
    view.filter('search', 'Probe-11');
    view.flush();
    assert.equal(view.data.filteredNodes.length, 6);
    view.filter('sort', 'cpu');
    assert.equal(view.data.filteredNodes[0].id, 'node-119');
    view.filter('selectedCountry', '');
    view.filter('search', '');
    const before = view.maps[0].calls.add;
    const next = overview(120, 1);
    next.nodes[0].latitude = 48;
    view.data.ingest(next);
    view.flush();
    assert.equal(view.maps[0].calls.add, before + 1);
    view.data.destroy();
});

test('SSE bursts render only the latest snapshot and hidden tabs release streams', () => {
    const view = observatory();
    for (let step = 0; step < 3; step++) view.streams[0].onmessage({data: JSON.stringify({data: overview(3, step)})});
    assert.equal(view.frames.size, 1);
    view.flush();
    assert.equal(view.data.overview.nodes[0].metrics.cpu_percent, 2);
    view.visibility(true);
    assert.equal(view.streams[0].closed, true);
    assert.equal(view.frames.size, 0);
    view.data.queueSnapshot(JSON.stringify({data: overview(3, 9)}));
    assert.equal(view.frames.size, 0);
    assert.equal(view.data.overview.nodes[0].metrics.cpu_percent, 2);
    view.visibility(false);
    assert.equal(view.streams.length, 2);
    view.streams[1].onmessage({data: JSON.stringify({data: overview(3, 10)})});
    view.flush();
    assert.equal(view.data.overview.nodes[0].metrics.cpu_percent, 10);
    view.data.destroy();
});

test('node details use real, bounded history and stop reacting after close', async () => {
    const view = observatory();
    view.data.ingest(overview(1));
    view.data.read = async () => Array.from({length: 20}, (_, index) => ({time: index}));
    await view.data.openNode(view.data.overview.nodes[0]);
    assert.equal(view.data.detailsOpen, true);
    assert.equal(view.data.history.length, 8);
    assert.equal(view.data.history[0].time, 19);
    view.data.closeDetails();
    assert.equal(view.data.detailsOpen, false);
    assert.equal(view.data.selectedNode, null);
    assert.equal(view.data.history.length, 0);
    view.data.destroy();
});

test('hidden map containers are never passed to the map resize calculation', () => {
    const view = observatory();
    const map = view.maps[0];
    view.data.$refs.map.offsetHeight = 0;
    view.data.resizeMap();
    assert.equal(map.calls.resize, 0);
    view.data.$refs.map.offsetHeight = 500;
    view.data.resizeMap();
    assert.equal(map.calls.resize, 1);
    view.data.hideMap();
    assert.equal(map.destroyed, true);
    view.data.$refs.map.offsetHeight = 0;
    view.data.toggleMap();
    assert.equal(view.maps.length, 1);
    view.data.$refs.map.offsetHeight = 500;
    view.flush();
    assert.equal(view.maps.length, 2);
    assert.equal(view.data.mapReady, true);
    view.data.destroy();
});

test('the page uses local Carbon components and contains no preview generator', () => {
    const page = filesystem.readFileSync('assets/monitor.template', 'utf8');
    const layout = filesystem.readFileSync('assets/observatory-layout.template', 'utf8');
    const head = filesystem.readFileSync('assets/document-head.template', 'utf8');
    assert.doesNotMatch(source + page, /togglePreview|demoOverview|preview-node|预览多节点|演示数据/);
    assert.doesNotMatch(layout + page + head, new RegExp(['tail', 'wind'].join('') + '|' + ['daisy', 'ui'].join('') + '|observatory\\.min\\.css'));
    assert.match(head, /css\/app\.css/);
    assert.match(page, /vendor\/carbon\/v2\.67\.0\/tile\.min\.js/);
    assert.match(layout, /cds-progress-bar/);
    assert.match(layout, /cds-modal/);
    assert.ok(head.indexOf('jsvectormap.min.css') < head.indexOf('css/app.css'));
});

test('Carbon theme changes redraw maps and pagehide tears down live resources', () => {
    const view = observatory();
    view.data.ingest(overview(2));
    view.flush();
    const previous = view.maps[0];
    assert.equal(previous.options.regionStyle.initial.fill, '#393939');
    view.document.documentElement.dataset.carbonTheme = 'white';
    view.listeners.get('carbon-theme-change')();
    assert.equal(previous.destroyed, true);
    assert.equal(view.maps[1].options.regionStyle.initial.fill, '#e0e0e0');
    view.globalListeners.get('pagehide')();
    assert.equal(view.maps[1].destroyed, true);
    assert.equal(view.streams[0].closed, true);
    assert.equal(view.frames.size, 0);
    assert.equal(view.listeners.has('keydown'), false);
    assert.equal(view.listeners.has('visibilitychange'), false);
    assert.equal(view.listeners.has('carbon-theme-change'), false);
    assert.equal(view.globalListeners.size, 0);
    view.data.destroy();
});

test('donut updates only when status totals change and re-themes with the page', () => {
    const view = observatory();
    view.data.ingest(overview(2));
    view.flush();
    const donut = view.charts[0];
    assert.equal(donut.kind, 'donut');
    assert.equal(donut.options.theme, 'g100');
    assert.equal(donut.options.color.scale['在线'], '#08bdba');
    assert.equal(donut.options.color.scale['离线'], '#33b1ff');
    assert.equal(donut.data[0].value, 2);
    view.data.ingest(overview(2, 1));
    view.flush();
    assert.equal(donut.updates, 0);
    view.data.ingest(overview(3));
    view.flush();
    assert.equal(donut.updates, 1);
    assert.equal(donut.data[0].value, 3);
    view.document.documentElement.dataset.carbonTheme = 'white';
    view.listeners.get('carbon-theme-change')();
    view.flush();
    assert.equal(donut.destroyed, true);
    assert.equal(view.charts[1].options.theme, 'white');
    assert.equal(view.charts[1].options.color.scale['在线'], '#009d9a');
    assert.equal(view.charts[1].options.color.scale['离线'], '#1192e8');
    view.data.destroy();
    assert.equal(view.charts[1].destroyed, true);
});

test('detail charts use bounded API history, chronological dates and complete teardown', async () => {
    const view = observatory();
    view.data.ingest(overview(1));
    view.flush();
    view.data.read = async () => Array.from({length: 20}, (_, index) => ({time: new Date(1700000000000 + index * 1000).toISOString(), cpu_percent: index, memory_percent: 20 + index, upload_rate: 1024 + index, download_rate: 2048 + index}));
    await view.data.openNode(view.data.overview.nodes[0]);
    view.flush();
    const details = view.charts.slice(1);
    assert.deepEqual(details.map(chart => chart.kind), ['area', 'area', 'line']);
    assert.equal(details[0].data.length, 8);
    assert.equal(details[0].data[0].value, 12);
    assert.equal(details[0].data[7].value, 19);
    assert.equal(details[2].data.length, 16);
    assert.ok(details[0].data[0].date < details[0].data[7].date);
    view.document.documentElement.dataset.carbonTheme = 'white';
    view.listeners.get('carbon-theme-change')();
    view.flush();
    assert.ok(details.every(chart => chart.destroyed));
    assert.ok(view.charts.slice(4).every(chart => chart.options.theme === 'white'));
    for (const name of ['statusChart', 'cpuHistory', 'memoryHistory', 'networkHistory']) {
        assert.equal(view.data.$refs[name].removed, false);
        assert.equal(view.data.$refs[name].children.length, 1);
        assert.equal(view.data.$refs[name].children[0].parentElement, view.data.$refs[name]);
    }
    view.data.closeDetails();
    assert.ok(view.charts.slice(4).filter(chart => chart.kind !== 'donut').every(chart => chart.destroyed));
    for (const name of ['cpuHistory', 'memoryHistory', 'networkHistory']) {
        assert.equal(view.data.$refs[name].removed, false);
        assert.equal(view.data.$refs[name].children.length, 0);
    }
    view.data.destroy();
    assert.ok(view.charts.every(chart => chart.destroyed));
    assert.equal(view.frames.size, 0);
});

test('the prepaint theme defaults to g100 and only an explicit choice changes it', () => {
    const themeSource = filesystem.readFileSync('assets/js/ui-theme.js', 'utf8');
    const callbacks = new Map();
    const saved = new Map();
    const stores = new Map();
    const root = {dataset: {}, style: {}, classList: {remove() {}, add() {}}};
    const context = {document: {documentElement: root, dispatchEvent() {}, addEventListener(name, handler) { callbacks.set(name, handler); }}, localStorage: {getItem(name) { return saved.get(name); }, setItem(name, value) { saved.set(name, value); }, removeItem(name) { saved.delete(name); }}, CustomEvent: class {}, Alpine: {store(name, store) { stores.set(name, store); }}};
    vm.runInNewContext(themeSource, context);
    assert.equal(root.dataset.carbonTheme, 'g100');
    callbacks.get('alpine:init')();
    stores.get('theme').toggle();
    assert.equal(root.dataset.carbonTheme, 'white');
    assert.equal(saved.get('carbon-theme'), 'white');
    stores.get('theme').toggle();
    assert.equal(root.dataset.carbonTheme, 'g100');
    assert.equal(saved.get('carbon-theme'), 'g100');
    saved.set('carbon-theme', 'white');
    vm.runInNewContext(themeSource, context);
    assert.equal(root.dataset.carbonTheme, 'white');
});

test('history requests cannot update a closed or destroyed detail view', async () => {
    const view = observatory();
    view.data.ingest(overview(1));
    let finish;
    let requestSignal;
    view.data.read = (path, signal) => { requestSignal = signal; return new Promise(resolve => { finish = resolve; }); };
    const pending = view.data.openNode(view.data.overview.nodes[0]);
    view.data.closeDetails();
    assert.equal(requestSignal.aborted, true);
    finish([{time: 'late'}]);
    await pending;
    assert.equal(view.data.history.length, 0);
    assert.equal(view.data.selectedNode, null);
    assert.equal(view.data.historyLoading, false);
    view.data.destroy();
});

test('Chinese formatting and tooltip escaping remain intact', () => {
    const view = observatory();
    assert.equal(view.data.countryName('DE'), '德国');
    assert.equal(view.data.uptime(172800), '2 天');
    assert.equal(view.data.escape('<img onerror="attack">'), '&lt;img onerror=&quot;attack&quot;&gt;');
    assert.equal(view.data.metricValue({metrics: {cpu_percent: 120}}, 'cpu'), 100);
    assert.equal(view.data.metricValue({metrics: {cpu_percent: 80, unavailable_fields: ['cpu']}}, 'cpu'), 0);
    view.data.destroy();
});

test('Carbon model events use their new value before the host property changes', () => {
    const callbacks = new Map();
    const directives = new Map();
    const stores = new Map();
    const context = {
        document: {addEventListener(name, handler) { callbacks.set(name, handler); }},
        addEventListener() {},
        customElements: {get() { return class {}; }, whenDefined() { return Promise.resolve(); }},
        Alpine: {
            directive(name, handler) { directives.set(name, handler); },
            store(name, value) { if (value) stores.set(name, value); return stores.get(name); },
            magic() {}, data() {},
        },
    };
    vm.runInNewContext(filesystem.readFileSync('assets/js/cds-alpine.js', 'utf8'), context);
    callbacks.get('alpine:init')();
    const cases = [
        {tagName: 'CDS-NUMBER-INPUT', property: 'value', initial: 22, event: 'cds-number-input', detail: {value: '2222'}, modifiers: ['number'], expected: 2222},
        {tagName: 'CDS-SELECT', property: 'value', initial: 'name', event: 'cds-select-selected', detail: {value: 'cpu'}, modifiers: [], expected: 'cpu'},
        {tagName: 'CDS-DROPDOWN', property: 'value', initial: 'before', event: 'cds-dropdown-selected', detail: {item: {value: 'after'}}, modifiers: [], expected: 'after'},
        {tagName: 'CDS-CHECKBOX', property: 'checked', initial: false, event: 'cds-checkbox-changed', detail: {checked: true}, modifiers: [], expected: true},
        {tagName: 'CDS-TOGGLE', property: 'toggled', initial: false, event: 'cds-toggle-changed', detail: {toggled: true}, modifiers: [], expected: true},
    ];
    for (const entry of cases) {
        const events = new Map();
        let assigned;
        const element = {tagName: entry.tagName, localName: entry.tagName.toLowerCase(), [entry.property]: entry.initial, addEventListener(name, handler) { events.set(name, handler); }};
        directives.get('cds-model')(element, {expression: 'model', modifiers: entry.modifiers}, {
            effect(handler) { handler(); },
            evaluateLater() { return callback => callback(entry.initial); },
            evaluate(expression, options) { assigned = options.scope.__cdsValue; },
            cleanup() {},
        });
        events.get(entry.event)({detail: entry.detail});
        assert.equal(assigned, entry.expected, entry.tagName);
    }
});

test('terminal button handlers keep the page mount and dispose their session resources', () => {
    let initialize, factory, opened, terminal, socket;
    const requests = [];
    const host = {dataset: {session: 'test-session'}};
    const root = {dataset: {page: '{}'}, querySelectorAll() { return [host]; }};
    const lifecycle = () => {
        const cleanups = [];
        return {
            signal: {aborted: false},
            add(cleanup) { cleanups.push(cleanup); }, listen() {}, observe() {}, timeout() {},
            dispose() { this.signal.aborted = true; for (const cleanup of cleanups.splice(0).reverse()) cleanup(); },
        };
    };
    const context = {
        document: {addEventListener(name, handler) { initialize = handler; }},
        localStorage: {getItem() { return null; }, removeItem() {}},
        location: {protocol: 'https:', host: '127.0.0.1:18443'},
        fetch(url, options) { requests.push({url, ...options}); return Promise.resolve(); },
        getComputedStyle(element) { assert.equal(element, host); return {getPropertyValue() { return '14px'; }}; },
        Alpine: {data(name, create) { factory = create; }, store() { return {lifecycle, xtermTheme() { return {}; }}; }},
        WebSocket: class {
            static OPEN = 1;
            constructor() { socket = this; }
            close() { this.closed = true; }
        },
        Terminal: class {
            constructor() { terminal = this; }
            loadAddon() {} open(element) { opened = element; }
            onData() { return {dispose() {}}; } onResize() { return {dispose() {}}; }
            dispose() { this.disposed = true; }
        },
        FitAddon: {FitAddon: class {}}, AttachAddon: {AttachAddon: class {}}, WebLinksAddon: {WebLinksAddon: class {}},
    };
    vm.runInNewContext(filesystem.readFileSync('assets/js/pages/term.js', 'utf8'), context);
    initialize();
    const data = factory();
    data.$el = root;
    data.init();
    data.sessions.push({id: 'test-session'});
    data.$el = {querySelectorAll() { return []; }};
    data.openTerminal('test-session');
    assert.equal(opened, host);
    data.destroy();
    assert.equal(terminal.disposed, true);
    assert.equal(socket.closed, true);
    assert.equal(requests.length, 1);
    assert.equal(requests[0].method, 'DELETE');
    assert.equal(requests[0].keepalive, true);
});

test('stacked Carbon tiles size to their content without doubling their gaps', () => {
    const styles = filesystem.readFileSync('assets/css/app.css', 'utf8');
    assert.match(styles, /cds-tile\s*\{[^}]*height:\s*auto/);
    assert.match(styles, /\.cds--row\s*>\s*\[class\*="cds--col-"\]\s*>\s*cds-tile:only-child\s*\{\s*height:\s*100%/);
    assert.match(styles, /cds-tile:not\(\.stack\)\s*>\s*\*\s*\+\s*\*\s*\{\s*margin-block-start:\s*var\(--cds-spacing-05\)/);
});

test('slotted overflow icons use the content theme rather than a white button fallback', () => {
    const styles = filesystem.readFileSync('assets/css/app.css', 'utf8');
    assert.match(styles, /cds-overflow-menu\s*\[slot="icon"\]\s*\{\s*color:\s*var\(--cds-icon-primary\)/);
});

test('wrapped Carbon table toolbars grow above their tables without squeezing cell text', () => {
    const styles = filesystem.readFileSync('assets/css/app.css', 'utf8');
    for (const selector of ['cds-table-toolbar', 'cds-table-toolbar-content']) {
        assert.match(styles, new RegExp(`${selector}\\s*\\{[^}]*height:\\s*auto;\\s*min-height:\\s*var\\(--cds-spacing-09\\)`));
    }
    assert.match(styles, /\.table-scroll cds-table-cell, \.table-scroll cds-table-header-cell\s*\{\s*overflow-wrap:\s*break-word/);
});

test('phone page padding and Carbon grid gutters use the 16px spacing token', () => {
    const styles = filesystem.readFileSync('assets/css/app.css', 'utf8');
    assert.match(styles, /\.app-content, \.public-content\s*\{[^}]*padding:\s*var\(--cds-spacing-05\);/);
    assert.match(styles, /@media \(max-width: 41\.99rem\)\s*\{\s*\.cds--row\s*\{[^}]*margin-inline:\s*calc\(-1 \* var\(--cds-spacing-03\)\);\s*\}\s*\.cds--row > \[class\*="cds--col-"\]\s*\{\s*padding-inline:\s*var\(--cds-spacing-03\)/);
});

test('dark low-contrast notifications use the Carbon layer instead of a white fallback', () => {
    const styles = filesystem.readFileSync('assets/css/app.css', 'utf8');
    for (const kind of ['error', 'info', 'success', 'warning']) {
        assert.match(styles, new RegExp(`\\.dds-theme-zone-g100\\s*\\{[^}]*--cds-notification-background-${kind}:\\s*var\\(--cds-layer-01\\)`));
    }
});
