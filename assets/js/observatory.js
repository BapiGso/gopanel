document.addEventListener('alpine:init', () => {
    Alpine.data('observatory', () => {
        let stream = null;
        let map = null;
        let mapFrame = null;
        let snapshotFrame = null;
        let pendingSnapshot = null;
        let keyHandler = null;
        let visibilityHandler = null;
        let resizeObserver = null;
        let historyRequest = null;
        let historySequence = 0;
        let destroyed = false;
        let regionNodes = {};
        let regionStates = {};
        let markerNodes = [];
        let markerSignature = '';
        let mapSignature = '';
        let mapColors = {};
        const nodeCache = new Map();
        const searchIndex = new Map();
        const countryNames = new Map();
        const collator = new Intl.Collator('zh-CN', {numeric: true});
        const timeFormatter = new Intl.DateTimeFormat('zh-CN', {hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false});
        let countryFormatter;
        try { countryFormatter = new Intl.DisplayNames(['zh-CN'], {type: 'region'}); } catch {}

        const sameItems = (first, second) => first.length === second.length && first.every((value, index) => value === second[index]);
        const updateFields = (target, source) => {
            for (const [name, value] of Object.entries(source)) {
                if (Array.isArray(value)) {
                    if (!Array.isArray(target[name]) || !sameItems(target[name], value)) target[name] = value.slice();
                } else if (target[name] !== value) target[name] = value;
            }
            for (const name of Object.keys(target)) { if (!(name in source)) delete target[name]; }
        };

        return {
            overview: {nodes: [], total: 0, online: 0, offline: 0, countries: 0, upload_rate: 0, download_rate: 0, upload_bytes: 0, download_bytes: 0, cluster: {}},
            title: 'GoPanel Observatory',
            apiBase: '/api/v1/public/monitor',
            publicView: true,
            loading: true,
            connected: false,
            error: '',
            selectedTag: 'All',
            selectedCountry: '',
            search: '',
            sort: 'name',
            tags: ['All'],
            filteredNodes: [],
            pageNodes: [],
            page: 1,
            pageSize: 50,
            visibleCountries: 0,
            showMap: true,
            dark: true,
            mapReady: false,
            mapError: '',
            selectedNode: null,
            history: [],
            historyLoading: false,

            init() {
                const bootstrap = JSON.parse(document.getElementById('observatory-bootstrap').textContent);
                this.title = bootstrap.title;
                this.apiBase = bootstrap.api_base;
                this.publicView = bootstrap.public;
                this.dark = localStorage.getItem('darkMode') !== 'false';
                this.applyTheme();
                this.showMap = localStorage.getItem('observatory-map') !== 'hidden';
                for (const field of ['selectedTag', 'selectedCountry', 'search', 'sort']) this.$watch(field, () => this.refreshFilters(true));
                keyHandler = event => {
                    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
                        event.preventDefault();
                        this.$refs.search.focus();
                    }
                };
                visibilityHandler = () => {
                    if (document.hidden) {
                        stream?.close();
                        stream = null;
                        this.connected = false;
                        cancelAnimationFrame(snapshotFrame);
                        cancelAnimationFrame(mapFrame);
                        snapshotFrame = mapFrame = null;
                        pendingSnapshot = null;
                    } else {
                        this.connect();
                        if (this.showMap) this.$nextTick(() => { if (!map) this.createMap(); else this.resizeMap(); this.scheduleMap(); });
                    }
                };
                document.addEventListener('keydown', keyHandler);
                document.addEventListener('visibilitychange', visibilityHandler);
                this.connect();
                this.$nextTick(() => { if (this.showMap) this.createMap(); });
            },

            destroy() {
                destroyed = true;
                stream?.close();
                historyRequest?.abort();
                cancelAnimationFrame(snapshotFrame);
                this.disposeMap();
                nodeCache.clear();
                searchIndex.clear();
                document.removeEventListener('keydown', keyHandler);
                document.removeEventListener('visibilitychange', visibilityHandler);
            },

            async read(path, signal) {
                const response = await fetch(this.apiBase + path, {credentials: 'same-origin', cache: 'no-store', signal});
                const payload = await response.json();
                if (!response.ok || payload.error) throw new Error(payload.error?.message || `HTTP ${response.status}`);
                return payload.data;
            },

            connect() {
                if (destroyed || document.hidden) return;
                stream?.close();
                stream = new EventSource(this.apiBase + '/events');
                stream.onopen = () => { this.connected = true; this.error = ''; };
                stream.onerror = () => { this.connected = false; this.error = '实时连接已断开，正在自动重连；当前显示最近一次收到的数据。'; };
                stream.onmessage = event => this.queueSnapshot(event.data);
            },

            queueSnapshot(content) {
                pendingSnapshot = content;
                if (snapshotFrame !== null || document.hidden || destroyed) return;
                snapshotFrame = requestAnimationFrame(() => {
                    snapshotFrame = null;
                    const content = pendingSnapshot;
                    pendingSnapshot = null;
                    try {
                        const payload = JSON.parse(content);
                        if (payload.error) throw new Error(payload.error.message);
                        this.ingest(payload.data);
                        this.connected = true;
                        this.error = '';
                    } catch (failure) {
                        this.error = '采样数据暂不可用。' + failure.message;
                    }
                });
            },

            ingest(view) {
                if (!Array.isArray(view?.nodes)) throw new Error('无效的节点快照');
                const present = new Set();
                const nodes = view.nodes.map(incoming => {
                    present.add(incoming.id);
                    const {metrics, ...metadata} = incoming;
                    metadata.tags ||= [];
                    let node = nodeCache.get(incoming.id);
                    if (!node) {
                        node = Alpine.reactive({...metadata, metrics: {...metrics}});
                        nodeCache.set(incoming.id, node);
                    } else {
                        for (const [name, value] of Object.entries(metadata)) {
                            if (Array.isArray(value)) {
                                if (!sameItems(node[name] || [], value)) node[name] = value.slice();
                            } else if (node[name] !== value) node[name] = value;
                        }
                        for (const name of Object.keys(node)) { if (name !== 'metrics' && !(name in metadata)) delete node[name]; }
                        updateFields(node.metrics, metrics);
                    }
                    searchIndex.set(node.id, [node.name, node.region, node.country, this.countryName(node.country), ...node.tags].join(' ').toLowerCase());
                    return node;
                });
                for (const id of nodeCache.keys()) {
                    if (!present.has(id)) { nodeCache.delete(id); searchIndex.delete(id); }
                }
                if (!sameItems(this.overview.nodes, nodes)) this.overview.nodes = nodes;
                for (const [name, value] of Object.entries(view)) {
                    if (name === 'cluster') updateFields(this.overview.cluster, value || {});
                    else if (name !== 'nodes' && this.overview[name] !== value) this.overview[name] = value;
                }
                this.loading = false;
                const title = view.title || this.title;
                if (this.title !== title) this.title = title;
                if (document.title !== title) document.title = title;
                if (this.selectedNode && !present.has(this.selectedNode.id)) this.$refs.details.close();
                this.refreshFilters();
            },

            refreshFilters(resetPage = false) {
                const tags = ['All', ...new Set(this.overview.nodes.flatMap(node => node.tags).filter(tag => tag !== 'All').sort(collator.compare))];
                if (!sameItems(this.tags, tags)) this.tags = tags;
                const term = this.search.trim().toLowerCase();
                const nodes = this.overview.nodes.filter(node =>
                    (this.selectedTag === 'All' || node.tags.includes(this.selectedTag)) &&
                    (!this.selectedCountry || node.country === this.selectedCountry) &&
                    (!term || searchIndex.get(node.id)?.includes(term))
                );
                nodes.sort((first, second) => {
                    if (this.sort === 'cpu') return second.metrics.cpu_percent - first.metrics.cpu_percent || collator.compare(first.name, second.name);
                    if (this.sort === 'memory') return second.metrics.memory_percent - first.metrics.memory_percent || collator.compare(first.name, second.name);
                    if (this.sort === 'traffic') return second.metrics.upload_rate + second.metrics.download_rate - first.metrics.upload_rate - first.metrics.download_rate || collator.compare(first.name, second.name);
                    if (this.sort === 'online' && first.online !== second.online) return Number(second.online) - Number(first.online);
                    return collator.compare(first.name, second.name);
                });
                if (!sameItems(this.filteredNodes, nodes)) this.filteredNodes = nodes;
                this.visibleCountries = new Set(nodes.map(node => node.country).filter(Boolean)).size;
                if (resetPage) this.page = 1;
                this.refreshPage();
                this.scheduleMap();
            },

            refreshPage() {
                this.page = Math.min(this.page, this.pageCount);
                const nodes = this.filteredNodes.slice((this.page - 1) * this.pageSize, this.page * this.pageSize);
                if (!sameItems(this.pageNodes, nodes)) this.pageNodes = nodes;
            },

            changePage(offset) { this.page = Math.max(1, Math.min(this.page + offset, this.pageCount)); this.refreshPage(); },
            get pageCount() { return Math.max(1, Math.ceil(this.filteredNodes.length / this.pageSize)); },
            get lastUpdated() { return this.overview.updated_at ? this.timestamp(this.overview.updated_at) : '等待采样'; },
            get clusterLabel() {
                const cluster = this.overview.cluster;
                if (cluster.coordination === 'manual') return `人工指定汇总 · ${cluster.healthy_peers || 0}/${cluster.peer_count || 0} 探针可达`;
                if (cluster.coordination === 'standalone') return '本机采样 · 只读';
                if (cluster.coordination === 'probe') return '独立探针 · 只读采样';
                if (!cluster.leader_name) return '候选汇总节点暂不可达';
                return `汇总节点 ${cluster.leader_name} · ${cluster.healthy_candidates}/${cluster.candidate_count} 可达`;
            },

            selectTag(tag) { this.selectedTag = tag; },
            clearCountry() { this.selectedCountry = ''; },
            toggleMap() {
                this.showMap = !this.showMap;
                localStorage.setItem('observatory-map', this.showMap ? 'visible' : 'hidden');
                if (this.showMap) this.$nextTick(() => { if (!map) this.createMap(); else this.resizeMap(); this.scheduleMap(); });
                else this.disposeMap();
            },
            hideMap() { this.showMap = false; localStorage.setItem('observatory-map', 'hidden'); this.disposeMap(); },
            toggleTheme() {
                this.dark = !this.dark;
                localStorage.setItem('darkMode', String(this.dark));
                this.applyTheme();
                this.disposeMap();
                if (this.showMap) this.createMap();
            },
            applyTheme() { document.documentElement.dataset.theme = this.dark ? 'business' : 'winter'; },

            disposeMap() {
                cancelAnimationFrame(mapFrame);
                mapFrame = null;
                resizeObserver?.disconnect();
                resizeObserver = null;
                map?.destroy();
                map = null;
                this.$refs.map?.replaceChildren();
                regionNodes = {};
                regionStates = {};
                markerNodes = [];
                markerSignature = mapSignature = '';
                this.mapReady = false;
            },

            scheduleMap() {
                if (!map || !this.showMap || document.hidden || mapFrame !== null) return;
                mapFrame = requestAnimationFrame(() => { mapFrame = null; this.updateMap(); });
            },

            resizeMap() {
                if (map && this.showMap && !document.hidden && this.$refs.map.offsetWidth > 0 && this.$refs.map.offsetHeight > 0) map.updateSize();
            },

            createMap() {
                if (map || !this.showMap || document.hidden || destroyed) return;
                if (this.$refs.map.offsetWidth === 0 || this.$refs.map.offsetHeight === 0) {
                    if (mapFrame === null && !document.hidden) mapFrame = requestAnimationFrame(() => { mapFrame = null; this.createMap(); });
                    return;
                }
                if (typeof jsVectorMap === 'undefined') { this.mapError = '地图组件尚未加载；节点监控仍然可用。'; return; }
                mapColors = this.dark ? {land: '#292929', border: '#383838', hover: '#3b3b3b', online: '#14563b', offline: '#5a3030'} : {land: '#e4e8eb', border: '#cbd3d9', hover: '#d2e3da', online: '#80c5a1', offline: '#dfa9a9'};
                try {
                    map = new jsVectorMap({
                        selector: '#world-map', map: 'world', markers: [], backgroundColor: 'transparent', zoomButtons: false, zoomOnScroll: false,
                        regionStyle: {initial: {fill: mapColors.land, stroke: mapColors.border, strokeWidth: 0.7}, hover: {fill: mapColors.hover}, selected: {fill: mapColors.online}},
                        markerStyle: {initial: {r: 4, fill: '#5bba8a', stroke: '#d1fae5', strokeWidth: 1.5}, hover: {r: 6}},
                        series: {regions: [{attribute: 'fill', scale: {online: mapColors.online, offline: mapColors.offline}, values: {}}]},
                        onRegionTooltipShow: (event, tooltip, code) => {
                            const nodes = regionNodes[code] || [];
                            const header = `<strong>${this.escape(this.countryName(code))}</strong><div class="mt-1 text-xs opacity-60">${nodes.length} 台服务器</div>`;
                            const list = nodes.slice(0, 12).map(node => `<div class="mt-1 flex items-center gap-2 text-xs"><span class="${node.online ? 'text-success' : 'text-error'}">●</span>${this.escape(node.name)}</div>`).join('');
                            tooltip.text(header + list, true);
                        },
                        onRegionClick: (event, code) => { if (regionNodes[code]?.length) this.selectedCountry = this.selectedCountry === code ? '' : code; },
                        onMarkerTooltipShow: (event, tooltip, index) => { const node = markerNodes[index]; if (node) tooltip.text(this.escape(node.name), true); },
                        onMarkerClick: (event, index) => { const node = markerNodes[index]; if (node) this.openNode(node); },
                    });
                    this.mapReady = true;
                    this.mapError = '';
                    let previousWidth = 0;
                    let previousHeight = 0;
                    resizeObserver = new ResizeObserver(entries => {
                        const {width, height} = entries[0].contentRect;
                        if (map && this.showMap && width > 0 && height > 0 && (width !== previousWidth || height !== previousHeight)) {
                            previousWidth = width;
                            previousHeight = height;
                            this.resizeMap();
                        }
                    });
                    resizeObserver.observe(this.$refs.map);
                    this.updateMap();
                } catch (failure) {
                    this.disposeMap();
                    this.mapError = '世界地图资源加载失败；节点列表仍可使用。';
                    console.error('observatory map:', failure);
                }
            },

            updateMap() {
                if (!map || !this.showMap) return;
                const nodes = this.filteredNodes.slice().sort((first, second) => collator.compare(first.id, second.id));
                const signature = JSON.stringify(nodes.map(node => [node.id, node.name, node.country, node.online, node.latitude, node.longitude]));
                if (mapSignature === signature) return;
                mapSignature = signature;
                const grouped = {};
                for (const node of nodes) { if (node.country) (grouped[node.country] ||= []).push(node); }
                const states = Object.fromEntries(Object.entries(grouped).map(([country, members]) => [country, members.some(node => node.online) ? 'online' : 'offline']));
                const attributes = {};
                for (const country of new Set([...Object.keys(regionStates), ...Object.keys(states)])) {
                    if (regionStates[country] !== states[country]) attributes[country] = states[country] ? mapColors[states[country]] : mapColors.land;
                }
                if (Object.keys(attributes).length) map.series.regions[0].setAttributes(attributes);
                regionNodes = grouped;
                regionStates = states;
                const positioned = nodes.filter(node => Number.isFinite(node.latitude) && Number.isFinite(node.longitude));
                const nextMarkers = JSON.stringify(positioned.map(node => [node.id, node.name, node.latitude, node.longitude]));
                if (nextMarkers !== markerSignature) {
                    markerSignature = nextMarkers;
                    markerNodes = positioned;
                    map.removeMarkers();
                    map.addMarkers(positioned.map(node => ({name: node.name, coords: [node.latitude, node.longitude]})));
                }
            },

            async openNode(node) {
                historyRequest?.abort();
                historyRequest = new AbortController();
                const sequence = ++historySequence;
                this.selectedNode = node;
                this.history = [];
                this.historyLoading = true;
                if (!this.$refs.details.open) this.$refs.details.showModal();
                try {
                    const history = await this.read('/nodes/' + encodeURIComponent(node.id) + '/history', historyRequest.signal);
                    if (historySequence === sequence) this.history = history.slice(-8).reverse();
                } catch (failure) {
                    if (historySequence === sequence) this.history = [];
                } finally {
                    if (historySequence === sequence) this.historyLoading = false;
                }
            },

            closeDetails() { historySequence++; historyRequest?.abort(); this.selectedNode = null; this.history = []; this.historyLoading = false; },
            flag(code) { return /^[A-Z]{2}$/.test(code || '') ? String.fromCodePoint(...[...code].map(letter => 127397 + letter.charCodeAt(0))) : '◎'; },
            countryName(code) {
                if (!code) return '未配置位置';
                if (!countryNames.has(code)) { try { countryNames.set(code, countryFormatter?.of(code) || code); } catch { countryNames.set(code, code); } }
                return countryNames.get(code);
            },
            timestamp(value) { try { return timeFormatter.format(new Date(value)); } catch { return '—'; } },
            bytes(value) { const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB']; let amount = Number(value) || 0; let index = 0; while (amount >= 1024 && index < units.length - 1) { amount /= 1024; index++; } return amount.toFixed(index ? 2 : 0) + ' ' + units[index]; },
            rate(value) { return this.bytes(value) + '/s'; },
            uptime(seconds) { const days = Math.floor((Number(seconds) || 0) / 86400); return days ? days + ' 天' : Math.floor((seconds || 0) / 3600) + ' 小时'; },
            percent(node, field) { return node.metrics.unavailable_fields.includes(field) ? '—' : (node.metrics[field + '_percent'] || 0).toFixed(2) + '%'; },
            level(value) { return value >= 90 ? 'progress-error' : value >= 75 ? 'progress-warning' : 'progress-success'; },
            expiry(value) { const days = Math.ceil((new Date(value + 'T23:59:59') - Date.now()) / 86400000); return days >= 0 ? '剩余 ' + days + ' 天' : '已到期'; },
            escape(value) { return String(value ?? '').replace(/[&<>"']/g, character => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[character])); },
        };
    });
});
