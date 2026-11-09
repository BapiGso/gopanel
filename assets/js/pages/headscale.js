document.addEventListener('alpine:init', () => {
    Alpine.data('headscalePage', () => {
        let life;
        return {
            settings: {}, revision: '', status: {state: 'unknown', stop_supported: false}, busy: false, loaded: false,
            get heading() { return Alpine.store('helpers').heading('Headscale', 'Configure an embedded Tailscale control server.', {status: this.status.state, type: this.status.state === 'running' ? 'green' : 'gray'}); },
            init() {
                const page = JSON.parse(this.$el.dataset.page);
                this.status.state = page.headscaleEnable ? 'running' : 'stopped';
                life = Alpine.store('helpers').lifecycle();
                this.loadConfig();
                this.readStatus();
                life.interval(() => { if (!document.hidden) return this.readStatus(); }, 5000);
                life.listen(document, 'visibilitychange', () => { if (!document.hidden) this.readStatus(); });
            },
            destroy() { life?.dispose(); },
            async loadConfig() {
                try {
                    const saved = await this.$api('/headscale/config', {signal: life.signal});
                    if (life.signal.aborted) return;
                    this.settings = JSON.parse(saved.content);
                    this.revision = saved.revision;
                    this.loaded = true;
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Configuration unavailable'); }
            },
            async readStatus() {
                try {
                    const status = await this.$api('/headscale/status', {signal: life.signal});
                    if (!life.signal.aborted) this.status = status;
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Status unavailable'); }
            },
            async persist() {
                const saved = await this.$api('/headscale/config', {method: 'PUT', body: {content: JSON.stringify(this.settings), revision: this.revision}, signal: life.signal});
                if (!life.signal.aborted) this.revision = saved.revision;
            },
            async save() {
                if (this.busy || !this.loaded) return;
                this.busy = true;
                try { await this.persist(); if (!life.signal.aborted) this.$notify('Configuration saved. Start the service to apply it.'); }
                catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Save failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async start() {
                if (this.busy || !this.loaded) return;
                this.busy = true;
                try {
                    await this.persist();
                    const status = await this.$api('/headscale/start', {method: 'POST', signal: life.signal});
                    if (life.signal.aborted) return;
                    this.status = status;
                    this.$notify('Headscale start accepted. Check status for startup errors.');
                    await this.readStatus();
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Start failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async stop() {
                if (this.busy || !this.status.stop_supported || !await this.$ask({title: 'Stop Headscale?', message: 'Connected clients may lose access.', danger: true})) return;
                this.busy = true;
                try {
                    const status = await this.$api('/headscale/stop', {method: 'POST', signal: life.signal});
                    if (!life.signal.aborted) { this.status = status; this.$notify('Headscale stopped.'); }
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Stop failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
        };
    });
}, {once: true});
