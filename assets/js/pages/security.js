document.addEventListener('alpine:init', () => {
    Alpine.data('securityPage', () => {
        let life;
        return {
            username: '', password: '', port: '', path: '', revision: '', version: '', busy: false, restartRequired: false,
            get heading() { return Alpine.store('helpers').heading('Security & System', 'Configure panel access and maintenance.', {status: this.version, codeStatus: true}); },
            init() {
                const page = JSON.parse(this.$el.dataset.page || '{}');
                const settings = page.panelConfig || {};
                this.username = settings.Username || '';
                this.port = settings.Port || '';
                this.path = settings.Path || '';
                this.revision = page.settingsRevision;
                this.version = page.version;
                life = Alpine.store('helpers').lifecycle();
                this.read();
            },
            destroy() { this.password = ''; life?.dispose(); },
            async read() {
                try {
                    const settings = await this.$api('/panel/settings', {signal: life.signal});
                    if (life.signal.aborted) return;
                    this.username = settings.username;
                    this.port = settings.port;
                    this.path = settings.path;
                    this.revision = settings.revision;
                    this.restartRequired = settings.restart_required;
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error'); }
            },
            async save() {
                if (this.busy) return;
                this.busy = true;
                try {
                    const body = {username: this.username, port: this.port, path: this.path, revision: this.revision};
                    if (this.password) body.password = this.password;
                    const settings = await this.$api('/panel/settings', {method: 'PATCH', body, signal: life.signal});
                    if (life.signal.aborted) return;
                    this.password = '';
                    this.revision = settings.revision;
                    this.restartRequired = settings.restart_required;
                    this.$notify(settings.restart_required ? 'Settings saved. Restart separately to apply the port or path.' : 'Settings saved.');
                    await this.read();
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Save failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async update() {
                if (this.busy || !await this.$ask({title: 'Update GoPanel?', message: 'Replace the binary with the latest release, without restarting it?'})) return;
                await this.maintenance('update');
            },
            async restart() {
                if (this.busy || !await this.$ask({title: 'Restart GoPanel?', message: 'The panel will become temporarily unavailable. Reconnect using the saved port and login path.', danger: true})) return;
                await this.maintenance('restart');
            },
            async maintenance(action) {
                this.busy = true;
                try {
                    await this.$api(`/panel/${action}`, {method: 'POST', signal: life.signal});
                    if (life.signal.aborted) return;
                    this.$notify(action === 'update' ? 'Binary updated. Restart the panel to use it.' : 'Restart accepted. Reconnect using the saved port and login path.');
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Maintenance failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
        };
    });
}, {once: true});
