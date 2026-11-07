const assert = require('node:assert/strict');
const filesystem = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');
const source = filesystem.readFileSync('assets/js/observatory.js', 'utf8');

function observatory() {
    const listeners = new Map();
    const frames = new Map();
    const watchers = new Map();
    const streams = [];
    const maps = [];
    let frameID = 0;
    let factory;
    const document = {
        hidden: false, title: '', documentElement: {dataset: {}},
        addEventListener(name, handler) { listeners.set(name, handler); },
        removeEventListener(name) { listeners.delete(name); },
        getElementById() { return {textContent: JSON.stringify({title: 'Test', api_base: '/api/v1/public/monitor', public: true})}; },
    };
    const context = {
        document, console, AbortController,
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
    data.$refs = {map: {offsetWidth: 1200, offsetHeight: 500, replaceChildren() {}}, search: {focus() {}}, details: {open: false, showModal() { this.open = true; }, close() { this.open = false; data.closeDetails(); }}};
    data.$watch = (name, handler) => watchers.set(name, handler);
    data.$nextTick = callback => callback();
    data.init();
    return {
        data, document, streams, maps, frames,
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
    assert.equal(view.data.history.length, 8);
    assert.equal(view.data.history[0].time, 19);
    view.data.$refs.details.close();
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

test('the page contains no preview generator or runtime Tailwind compiler', () => {
    const page = filesystem.readFileSync('assets/monitor.template', 'utf8');
    const layout = filesystem.readFileSync('assets/observatory-layout.template', 'utf8');
    assert.doesNotMatch(source + page, /togglePreview|demoOverview|preview-node|预览多节点|演示数据/);
    assert.doesNotMatch(layout, /js\/tailwind\.js|css\/daisyui\.min\.css/);
    assert.match(layout, /css\/observatory\.min\.css/);
    assert.ok(layout.indexOf('jsvectormap.min.css') < layout.indexOf('observatory.min.css'));
});
