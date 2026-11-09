document.addEventListener('alpine:init', () => {
    Alpine.data('webdavPage', () => {
        let life;
        return {
            enabled: false, username: '', password: '', revision: '', busy: false, address: '',
            get heading() { return Alpine.store('helpers').heading('WebDAV Manager', 'Access files remotely through WebDAV.', {status: this.enabled ? 'Enabled' : 'Disabled', type: this.enabled ? 'green' : 'gray', actions: [{label: 'File manager', href: '/admin/file'}]}); },
            init() {
                const page = JSON.parse(this.$el.dataset.page || '{}');
                this.enabled = Boolean(page.webdavConfig?.Enable);
                this.username = page.webdavConfig?.Username || '';
                this.revision = page.settingsRevision;
                this.address = `${location.origin}/webdav`;
                life = Alpine.store('helpers').lifecycle();
            },
            destroy() { this.password = ''; life?.dispose(); },
            async save() {
                if (this.busy) return;
                this.busy = true;
                try {
                    const body = {enable: this.enabled, username: this.username, revision: this.revision};
                    if (this.password) body.password = this.password;
                    await this.$api('/webdav/settings', {method: 'PATCH', body, signal: life.signal});
                    const settings = await this.$api('/webdav/settings', {signal: life.signal});
                    if (life.signal.aborted) return;
                    this.revision = settings.revision;
                    this.enabled = settings.enable;
                    this.username = settings.username;
                    this.password = '';
                    this.$notify('WebDAV settings saved.');
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Save failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
        };
    });
}, {once: true});
