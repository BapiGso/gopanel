document.addEventListener('alpine:init', () => {
    Alpine.data('observatory', () => ({
        overview: {nodes: [], total: 0, online: 0, offline: 0, countries: 0, upload_rate: 0, download_rate: 0, upload_bytes: 0, download_bytes: 0, cluster: {}},
        title: 'GoPanel Observatory',
        apiBase: '/api/v1/public/monitor',
        publicView: true,
        loading: true,
        connected: false,
        preview: false,
        error: '',
        selectedTag: 'All',
        selectedCountry: '',
        search: '',
        sort: 'name',
        showMap: true,
        dark: true,
        mapReady: false,
        mapError: '',
        selectedNode: null,
        history: [],
        historyLoading: false,
        _stream: null,
        _map: null,
        _mapTimer: null,
        _keyHandler: null,
        _regionNodes: {},
        _resizeObserver: null,
        _landColor: '',

        init() {
            const bootstrap = JSON.parse(document.getElementById('observatory-bootstrap').textContent);
            this.title = bootstrap.title;
            this.apiBase = bootstrap.api_base;
            this.publicView = bootstrap.public;
            this.dark = localStorage.getItem('darkMode') !== 'false';
            this.applyTheme();
            this.showMap = localStorage.getItem('observatory-map') !== 'hidden';
            this._keyHandler = event => {
                if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
                    event.preventDefault();
                    this.$refs.search.focus();
                }
            };
            document.addEventListener('keydown', this._keyHandler);
            this.connect();
            this.$nextTick(() => { if (this.showMap) this.createMap(); });
        },

        destroy() {
            this._stream?.close();
            this.disposeMap();
            clearTimeout(this._mapTimer);
            document.removeEventListener('keydown', this._keyHandler);
        },

        async read(path) {
            const response = await fetch(this.apiBase + path, {credentials: 'same-origin', cache: 'no-store'});
            const payload = await response.json();
            if (!response.ok || payload.error) throw new Error(payload.error?.message || `HTTP ${response.status}`);
            return payload.data;
        },

        async connect() {
            this._stream?.close();
            try {
                const view = await this.read('/nodes');
                if (!this.preview) this.ingest(view);
            } catch (failure) {
                this.error = '无法读取监控数据，正在重试。' + failure.message;
            }
            if (this.preview) return;
            this._stream = new EventSource(this.apiBase + '/events');
            this._stream.onopen = () => { this.connected = true; this.error = ''; };
            this._stream.onerror = () => { this.connected = false; this.error = '实时连接已断开，正在自动重连；当前显示最近一次收到的数据。'; };
            this._stream.onmessage = event => {
                if (this.preview) return;
                try {
                    const payload = JSON.parse(event.data);
                    if (payload.error) throw new Error(payload.error.message);
                    this.ingest(payload.data);
                    this.connected = true;
                    this.error = '';
                } catch (failure) {
                    this.error = '采样数据暂不可用。' + failure.message;
                }
            };
        },

        ingest(view) {
            this.overview = view;
            this.loading = false;
            this.title = view.title || this.title;
            document.title = this.title;
            if (this.selectedNode) this.selectedNode = view.nodes.find(node => node.id === this.selectedNode.id) || this.selectedNode;
            this.scheduleMap();
        },

        get tags() {
            return ['All', ...new Set(this.overview.nodes.flatMap(node => node.tags || []).sort())];
        },

        get filteredNodes() {
            const term = this.search.trim().toLowerCase();
            const nodes = this.overview.nodes.filter(node =>
                (this.selectedTag === 'All' || node.tags.includes(this.selectedTag)) &&
                (!this.selectedCountry || node.country === this.selectedCountry) &&
                (!term || [node.name, node.region, node.country, this.countryName(node.country), ...node.tags].join(' ').toLowerCase().includes(term))
            );
            return nodes.slice().sort((first, second) => {
                if (this.sort === 'cpu') return second.metrics.cpu_percent - first.metrics.cpu_percent;
                if (this.sort === 'memory') return second.metrics.memory_percent - first.metrics.memory_percent;
                if (this.sort === 'traffic') return second.metrics.upload_rate + second.metrics.download_rate - first.metrics.upload_rate - first.metrics.download_rate;
                if (this.sort === 'online' && first.online !== second.online) return Number(second.online) - Number(first.online);
                return first.name.localeCompare(second.name);
            });
        },

        get visibleCountries() {
            return new Set(this.filteredNodes.map(node => node.country).filter(Boolean)).size;
        },

        get lastUpdated() {
            return this.overview.updated_at ? this.timestamp(this.overview.updated_at) : '等待采样';
        },

        get clusterLabel() {
            const cluster = this.overview.cluster || {};
            if (cluster.coordination === 'standalone') return '本机采样 · 只读';
            if (!cluster.leader_name) return '候选汇总节点暂不可达';
            return `汇总节点 ${cluster.leader_name} · ${cluster.healthy_candidates}/${cluster.candidate_count} 可达`;
        },

        selectTag(tag) { this.selectedTag = tag; this.scheduleMap(); },
        clearCountry() { this.selectedCountry = ''; this.scheduleMap(); },
        toggleMap() { this.showMap = !this.showMap; localStorage.setItem('observatory-map', this.showMap ? 'visible' : 'hidden'); if (this.showMap) this.$nextTick(() => { if (!this._map) this.createMap(); else this._map.updateSize(); this.scheduleMap(); }); },
        hideMap() { this.showMap = false; localStorage.setItem('observatory-map', 'hidden'); },
        toggleTheme() { this.dark = !this.dark; localStorage.setItem('darkMode', String(this.dark)); this.applyTheme(); if (this._map) { this.disposeMap(); this.createMap(); } },
        applyTheme() { document.documentElement.dataset.theme = this.dark ? 'business' : 'winter'; document.documentElement.classList.toggle('dark', this.dark); },

        disposeMap() {
            this._resizeObserver?.disconnect();
            this._map?.destroy();
            this._map = null;
            this.$refs.map?.replaceChildren();
            this._regionNodes = {};
            this.mapReady = false;
        },

        scheduleMap() {
            clearTimeout(this._mapTimer);
            this._mapTimer = setTimeout(() => this.updateMap(), 80);
        },

        createMap() {
            if (typeof jsVectorMap === 'undefined') {
                this.mapError = '地图组件尚未加载；节点监控仍然可用。';
                return;
            }
            const colors = this.dark ? {land: '#292929', border: '#383838', hover: '#3b3b3b', online: '#14563b', offline: '#5a3030'} : {land: '#e4e8eb', border: '#cbd3d9', hover: '#d2e3da', online: '#80c5a1', offline: '#dfa9a9'};
            this._landColor = colors.land;
            this._resizeObserver?.disconnect();
            try {
                this._map = new jsVectorMap({
                    selector: '#world-map',
                    map: 'world',
                    markers: [],
                    backgroundColor: 'transparent',
                    zoomButtons: false,
                    zoomOnScroll: false,
                    regionStyle: {initial: {fill: colors.land, stroke: colors.border, strokeWidth: 0.7}, hover: {fill: colors.hover}, selected: {fill: colors.online}},
                    markerStyle: {initial: {r: 4, fill: '#5bba8a', stroke: '#d1fae5', strokeWidth: 1.5}, hover: {r: 6}},
                    series: {regions: [{attribute: 'fill', scale: {online: colors.online, offline: colors.offline}, values: {}}]},
                    onRegionTooltipShow: (event, tooltip, code) => {
                        const nodes = this._regionNodes[code] || [];
                        const header = `<strong>${this.escape(this.countryName(code))}</strong><div class="mt-1 text-xs opacity-60">${nodes.length} 台服务器</div>`;
                        const list = nodes.slice(0, 12).map(node => `<div class="mt-1 flex items-center gap-2 text-xs"><span class="${node.online ? 'text-success' : 'text-error'}">●</span>${this.escape(node.name)}</div>`).join('');
                        tooltip.text(header + list, true);
                    },
                    onRegionClick: (event, code) => { if (this._regionNodes[code]?.length) { this.selectedCountry = this.selectedCountry === code ? '' : code; this.scheduleMap(); } },
                    onMarkerTooltipShow: (event, tooltip, index) => { const node = this._markerNodes[index]; if (node) tooltip.text(this.escape(node.name), true); },
                    onMarkerClick: (event, index) => { const node = this._markerNodes[index]; if (node) this.openNode(node); },
                });
                this.mapReady = true;
                this.mapError = '';
                this._resizeObserver = new ResizeObserver(() => { if (this._map && this.showMap) this.scheduleMap(); });
                this._resizeObserver.observe(this.$refs.map);
                this.updateMap();
            } catch (failure) {
                this.mapReady = false;
                this.mapError = '世界地图资源加载失败；节点列表仍可使用。';
                console.error('observatory map:', failure);
            }
        },

        updateMap() {
            if (!this._map) return;
            this._map.updateSize();
            const grouped = {};
            const nodes = this.filteredNodes;
            for (const node of nodes) { if (node.country) (grouped[node.country] ||= []).push(node); }
            const reset = Object.fromEntries(Object.keys(this._regionNodes).map(country => [country, this._landColor]));
            this._map.series.regions[0].setAttributes(reset);
            this._regionNodes = grouped;
            const values = Object.fromEntries(Object.entries(grouped).map(([country, members]) => [country, members.some(node => node.online) ? 'online' : 'offline']));
            this._map.series.regions[0].setValues(values);
            this._markerNodes = nodes.filter(node => Number.isFinite(node.latitude) && Number.isFinite(node.longitude));
            this._map.removeMarkers();
            this._map.addMarkers(this._markerNodes.map(node => ({name: node.name, coords: [node.latitude, node.longitude]})));
        },

        async openNode(node) {
            this.selectedNode = node;
            this.history = [];
            this.$refs.details.showModal();
            this.historyLoading = true;
            if (this.preview) {
                this.history = Array.from({length: 40}, (_, index) => ({time: new Date(Date.now() - (40-index)*2000).toISOString(), cpu_percent: Math.max(0, node.metrics.cpu_percent + Math.sin(index/4)*3), memory_percent: node.metrics.memory_percent, upload_rate: node.metrics.upload_rate, download_rate: node.metrics.download_rate}));
                this.historyLoading = false;
                return;
            }
            const id = node.id;
            try {
                const history = await this.read('/nodes/' + encodeURIComponent(id) + '/history');
                if (this.selectedNode?.id === id) this.history = history;
            } catch (failure) {
                this.history = [];
            } finally {
                if (this.selectedNode?.id === id) this.historyLoading = false;
            }
        },

        togglePreview() {
            this.preview = !this.preview;
            this.$refs.details.close();
            this.selectedNode = null;
            this.selectedCountry = '';
            this.selectedTag = 'All';
            this.search = '';
            this._stream?.close();
            this.connected = false;
            this.error = '';
            if (this.preview) this.ingest(this.demoOverview());
            else { this.loading = true; this.connect(); }
        },

        demoOverview() {
            const locations = [
                ['JP-Tokyo.Edge', 'JP', 'Tokyo', 35.68, 139.69, 'Relay'],
                ['US-LosAngeles.Core', 'US', 'Los Angeles', 34.05, -118.24, 'Relay'],
                ['DE-Frankfurt.Cloud', 'DE', 'Frankfurt', 50.11, 8.68, 'Service'],
                ['CA-Beauharnois.Backup', 'CA', 'Beauharnois', 45.32, -73.87, 'Storage'],
                ['SG-Singapore.Gateway', 'SG', 'Singapore', 1.35, 103.82, 'Relay'],
                ['GB-London.Dev', 'GB', 'London', 51.51, -0.13, 'Dev'],
                ['CN-Shanghai.Outbound', 'CN', 'Shanghai', 31.23, 121.47, 'Relay'],
                ['NL-Amsterdam.Compute', 'NL', 'Amsterdam', 52.37, 4.90, 'Service'],
                ['AU-Sydney.Edge', 'AU', 'Sydney', -33.87, 151.21, 'Relay'],
                ['US-NewYork.Frontend', 'US', 'New York', 40.71, -74.01, 'Frontend'],
                ['JP-Osaka.Archive', 'JP', 'Osaka', 34.69, 135.50, 'Storage'],
            ];
            const now = new Date().toISOString();
            const nodes = locations.map((location, index) => ({
                id: 'preview-node-' + index, name: location[0], country: location[1], region: location[2], latitude: location[3], longitude: location[4], tags: [location[5]],
                online: index !== 10, collected_at: now, last_seen: now, price: index % 3 === 0 ? '$24 / 年' : '',
                metrics: {cpu_percent: [1.5, 6.8, 12.4, 0.8, 3.2, 8.6, 5.1, 17.2, 2.4, 9.3, 0][index], memory_percent: 18 + (index*7)%53, disk_percent: 16 + (index*9)%62, memory_total: 4294967296, memory_used: 4294967296*(18+(index*7)%53)/100, disk_total: 85899345920, disk_used: 85899345920*(16+(index*9)%62)/100, upload_rate: 1800+index*3700, download_rate: 2500+index*5100, upload_bytes: (40+index*7)*1073741824, download_bytes: (52+index*9)*1073741824, uptime_seconds: (13+index*3)*86400, platform: index===5 ? 'ubuntu' : 'debian', architecture: 'amd64', cpu_cores: 2, available: true, rate_available: true, unavailable_fields: []},
            }));
            return {title: 'GoPanel Observatory', updated_at: now, nodes, total: nodes.length, online: 10, offline: 1, countries: 9, upload_rate: nodes.filter(node=>node.online).reduce((total,node)=>total+node.metrics.upload_rate,0), download_rate: nodes.filter(node=>node.online).reduce((total,node)=>total+node.metrics.download_rate,0), upload_bytes: nodes.reduce((total,node)=>total+node.metrics.upload_bytes,0), download_bytes: nodes.reduce((total,node)=>total+node.metrics.download_bytes,0), cluster: {coordination: 'best_effort', role: 'candidate', leader_id: 'preview-node-0', leader_name: 'JP-Tokyo.Edge', healthy_candidates: 3, candidate_count: 3}};
        },

        flag(code) { return /^[A-Z]{2}$/.test(code || '') ? String.fromCodePoint(...[...code].map(letter => 127397 + letter.charCodeAt(0))) : '◎'; },
        countryName(code) { if (!code) return '未配置位置'; try { return new Intl.DisplayNames(['zh-CN'], {type: 'region'}).of(code); } catch { return code; } },
        timestamp(value) { return new Date(value).toLocaleTimeString('zh-CN', {hour12: false}); },
        bytes(value) { const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB']; let amount = Number(value) || 0; let index = 0; while (amount >= 1024 && index < units.length-1) { amount /= 1024; index++; } return amount.toFixed(index ? 2 : 0) + ' ' + units[index]; },
        rate(value) { return this.bytes(value) + '/s'; },
        uptime(seconds) { const days = Math.floor((Number(seconds) || 0)/86400); return days ? days + ' 天' : Math.floor((seconds || 0)/3600) + ' 小时'; },
        percent(node, field) { return node.metrics.unavailable_fields.includes(field) ? '—' : (node.metrics[field + '_percent'] || 0).toFixed(2) + '%'; },
        level(value) { return value >= 90 ? 'progress-error' : value >= 75 ? 'progress-warning' : 'progress-success'; },
        expiry(value) { const days = Math.ceil((new Date(value + 'T23:59:59') - Date.now()) / 86400000); return days >= 0 ? '剩余 ' + days + ' 天' : '已到期'; },
        escape(value) { const element = document.createElement('span'); element.textContent = String(value ?? ''); return element.innerHTML; },
    }));
});
