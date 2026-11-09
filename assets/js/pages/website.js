document.addEventListener('alpine:init', () => {
    Alpine.data('websitePage', () => {
        let life, editor;
        return {
            revision: '', status: {state: 'unknown', autostart: false}, busy: false,
            get heading() { return Alpine.store('helpers').heading('Website', 'Configure Caddy and manage the website service.', {status: this.status.state, type: this.status.state === 'running' ? 'green' : 'gray', actions: [{label: 'Documentation', href: 'https://caddyserver.com/docs/caddyfile', target: '_blank'}]}); },
            init() {
                const page = JSON.parse(this.$el.dataset.page);
                this.revision = page.caddyRevision;
                this.status.autostart = page.caddyEnable;
                life = Alpine.store('helpers').lifecycle();
                editor = Alpine.store('helpers').editor(this.$refs.editor, {mode: 'caddyfile', save: () => this.save()}, life);
                editor.setValue(page.caddyFile || '', -1);
                this.readStatus();
                life.interval(() => { if (!document.hidden) return this.readStatus(); }, 5000);
            },
            destroy() { life?.dispose(); },
            async readStatus() {
                try {
                    const status = await this.$api('/caddy/status', {signal: life.signal});
                    if (!life.signal.aborted) this.status = status;
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error'); }
            },
            async reloadDocument() {
                if (!await this.$ask({title: 'Read the saved Caddyfile?', message: 'Unsaved editor changes will be replaced.'})) return;
                try {
                    const saved = await this.$api('/caddy/config', {signal: life.signal});
                    if (life.signal.aborted) return;
                    editor.setValue(saved.content, -1);
                    this.revision = saved.revision;
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error'); }
            },
            async save() {
                if (this.busy) return;
                this.busy = true;
                try {
                    const saved = await this.$api('/caddy/config', {method: 'PUT', body: {content: editor.getValue(), revision: this.revision}, signal: life.signal});
                    if (life.signal.aborted) return;
                    this.revision = saved.revision;
                    this.$notify('Configuration saved. Reload Caddy to apply it.');
                    await this.readStatus();
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Save failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async manage(action) {
                if (this.busy) return;
                this.busy = true;
                try {
                    const status = await this.$api(`/caddy/${action}`, {method: 'POST', signal: life.signal});
                    if (life.signal.aborted) return;
                    this.status = status;
                    this.$notify(`Caddy ${action} completed.`);
                    await this.readStatus();
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Operation failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async autostart(enabled) {
                if (this.busy || enabled === this.status.autostart) return;
                this.busy = true;
                try {
                    const status = await this.$api('/caddy/settings', {method: 'PATCH', body: {autostart: enabled, revision: this.status.settings_revision}, signal: life.signal});
                    if (!life.signal.aborted) { this.status = status; this.$notify('Autostart preference saved.'); }
                } catch (error) { if (!life.signal.aborted) { this.$notify(error.message, 'error'); await this.readStatus(); } }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
        };
    });
}, {once: true});
