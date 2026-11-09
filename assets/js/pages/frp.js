document.addEventListener('alpine:init', () => {
    Alpine.data('frpPage', () => {
        let life;
        const editors = {};
        return {
            currentTab: 'frps', revisions: {}, statuses: {frps: {state: 'unknown', autostart: false}, frpc: {state: 'unknown', autostart: false}}, busy: false,
            get currentStatus() { return this.statuses[this.currentTab]; },
            get heading() { return Alpine.store('helpers').heading('FRP', 'Manage the Fast Reverse Proxy server and client.', {status: this.currentStatus.state, type: this.currentStatus.state === 'running' ? 'green' : 'gray', actions: [{label: 'Configuration example', href: `https://github.com/fatedier/frp/blob/dev/conf/${this.currentTab}_full_example.toml`, target: '_blank'}]}); },
            init() {
                const page = JSON.parse(this.$el.dataset.page);
                life = Alpine.store('helpers').lifecycle();
                for (const name of ['frps', 'frpc']) {
                    this.revisions[name] = page[name + 'Revision'];
                    this.statuses[name].autostart = page[name + 'Enable'];
                    editors[name] = Alpine.store('helpers').editor(this.$refs[name + 'Editor'], {mode: 'toml', save: () => this.save()}, life);
                    editors[name].setValue(page[name + 'Config'] || '', -1);
                }
                this.readStatus();
                life.interval(() => { if (!document.hidden) return this.readStatus(); }, 5000);
            },
            destroy() { life?.dispose(); },
            switchTab(name) {
                if (!editors[name]) return;
                this.currentTab = name;
                this.$nextTick(() => { if (!life.signal.aborted) editors[name].resize(); });
            },
            async readStatus() {
                try {
                    const statuses = await Promise.all(['frps', 'frpc'].map(name => this.$api(`/frp/${name}/status`, {signal: life.signal})));
                    if (!life.signal.aborted) [this.statuses.frps, this.statuses.frpc] = statuses;
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error'); }
            },
            async reloadDocument() {
                const name = this.currentTab;
                if (!await this.$ask({title: `Read saved ${name} configuration?`, message: 'Unsaved editor changes will be replaced.'})) return;
                try {
                    const saved = await this.$api(`/frp/${name}/config`, {signal: life.signal});
                    if (life.signal.aborted) return;
                    editors[name].setValue(saved.content, -1);
                    this.revisions[name] = saved.revision;
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error'); }
            },
            async save() {
                if (this.busy) return;
                const name = this.currentTab;
                this.busy = true;
                try {
                    const saved = await this.$api(`/frp/${name}/config`, {method: 'PUT', body: {content: editors[name].getValue(), revision: this.revisions[name]}, signal: life.signal});
                    if (life.signal.aborted) return;
                    this.revisions[name] = saved.revision;
                    this.$notify(`${name} configuration saved. Reload to apply it.`);
                    await this.readStatus();
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Save failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async manage(action) {
                if (this.busy) return;
                const name = this.currentTab;
                this.busy = true;
                try {
                    const status = await this.$api(`/frp/${name}/${action}`, {method: 'POST', signal: life.signal});
                    if (life.signal.aborted) return;
                    this.statuses[name] = status;
                    this.$notify(`${name} ${action} accepted. Check status for startup errors.`);
                    await this.readStatus();
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Operation failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async autostart(enabled) {
                if (this.busy || enabled === this.currentStatus.autostart) return;
                const name = this.currentTab;
                this.busy = true;
                try {
                    const status = await this.$api(`/frp/${name}/settings`, {method: 'PATCH', body: {autostart: enabled, revision: this.currentStatus.settings_revision}, signal: life.signal});
                    if (!life.signal.aborted) { this.statuses[name] = status; this.$notify('Autostart preference saved.'); }
                } catch (error) { if (!life.signal.aborted) { this.$notify(error.message, 'error'); await this.readStatus(); } }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
        };
    });
}, {once: true});
