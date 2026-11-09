document.addEventListener('alpine:init', () => {
    Alpine.data('filePage', () => {
        let life, editor;
        return {
            directory: '/', pathInput: '/', files: [], search: '', busy: false, changeOpen: false, editorOpen: false,
            operation: 'create', selected: null, newValue: '', newType: 'file', editingPath: '',
            get heading() { return Alpine.store('helpers').heading('File manager', 'Browse, upload, and manage server files.', {actions: [{label: 'WebDAV access', href: '/admin/webdav'}]}); },
            get filteredFiles() { const query = this.search.toLowerCase(); return this.files.filter(entry => entry.name.toLowerCase().includes(query)); },
            get crumbs() {
                let path = '';
                return this.directory.split('/').filter(Boolean).map(name => { path += '/' + name; return {name, path}; });
            },
            get changeTitle() { return {create: 'Create file or folder', rename: 'Rename', permissions: 'Change permissions'}[this.operation]; },
            init() {
                const page = JSON.parse(this.$el.dataset.page);
                this.directory = this.pathInput = page.Dir || '/';
                this.files = page.Files || [];
                life = Alpine.store('helpers').lifecycle();
                editor = Alpine.store('helpers').editor(this.$refs.editor, {save: () => this.saveEditor()}, life);
            },
            destroy() { life?.dispose(); },
            downloadPath(path) { return '/api/v1' + this.$filePath('downloads', path); },
            join(name) { return (this.directory === '/' ? '' : this.directory) + '/' + name; },
            async navigate(path) {
                if (this.busy) return;
                this.busy = true;
                try {
                    const result = await this.$api(this.$filePath('files', path || '/'), {signal: life.signal});
                    if (life.signal.aborted) return;
                    if (result.type !== 'directory') throw new Error('The requested path is not a directory.');
                    this.directory = this.pathInput = result.path;
                    this.files = result.entries || [];
                    this.search = '';
                    document.cookie = `dirHistory=${encodeURIComponent(result.path)}; path=/; max-age=86400; SameSite=Lax; Secure`;
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Directory unavailable'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            up() { const parent = this.directory.slice(0, this.directory.lastIndexOf('/')) || '/'; return this.navigate(parent); },
            async refresh() {
                const result = await this.$api(this.$filePath('files', this.directory), {signal: life.signal});
                if (!life.signal.aborted) this.files = result.entries || [];
            },
            openCreate() { this.operation = 'create'; this.selected = null; this.newValue = ''; this.newType = 'file'; this.changeOpen = true; },
            openChange(operation, entry) { this.operation = operation; this.selected = entry; this.newValue = operation === 'rename' ? entry.name : entry.permissions; this.changeOpen = true; },
            async applyChange() {
                if (this.busy) return;
                if (this.operation === 'permissions' ? !/^[0-7]{3,4}$/.test(this.newValue) : !this.newValue || /[/\\]/.test(this.newValue) || ['.', '..'].includes(this.newValue)) {
                    this.$notify(this.operation === 'permissions' ? 'Use three or four octal digits.' : 'Enter a single non-empty file or folder name.', 'error'); return;
                }
                this.busy = true;
                try {
                    const create = this.operation === 'create';
                    const body = create ? {path: this.join(this.newValue), type: this.newType} : this.operation === 'rename' ? {destination: this.join(this.newValue)} : {permissions: this.newValue};
                    await this.$api(create ? '/files' : this.$filePath('files', this.selected.path), {method: create ? 'POST' : 'PATCH', body, signal: life.signal});
                    if (life.signal.aborted) return;
                    this.changeOpen = false;
                    await this.refresh();
                    this.$notify('File operation completed.');
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Operation failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async remove(entry) {
                if (this.busy || !await this.$ask({title: 'Delete ' + entry.name + '?', message: entry.type === 'directory' ? 'This recursively deletes the folder and all its contents. This cannot be undone.' : 'This permanently deletes the file. This cannot be undone.', danger: true})) return;
                this.busy = true;
                try {
                    await this.$api(this.$filePath('files', entry.path), {method: 'DELETE', signal: life.signal});
                    if (life.signal.aborted) return;
                    await this.refresh();
                    this.$notify('Entry deleted.');
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Delete failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async openEditor(entry) {
                if (this.busy) return;
                this.busy = true;
                try {
                    const result = await this.$api(this.$filePath('files', entry.path), {signal: life.signal});
                    if (life.signal.aborted) return;
                    this.editingPath = entry.path;
                    editor.session.setMode(`ace/mode/${entry.name.toLowerCase() === 'caddyfile' ? 'caddyfile' : entry.name.endsWith('.toml') ? 'toml' : 'text'}`);
                    editor.setValue(result.content || '', -1);
                    this.editorOpen = true;
                    this.$nextTick(() => { if (!life.signal.aborted) editor.resize(); });
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Editor unavailable'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async saveEditor() {
                if (this.busy || !this.editorOpen) return;
                this.busy = true;
                try {
                    await this.$api(this.$filePath('files', this.editingPath), {method: 'PUT', body: {content: editor.getValue()}, signal: life.signal});
                    if (life.signal.aborted) return;
                    await this.refresh();
                    this.$notify('File saved.');
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Save failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async upload(addedFiles) {
                const files = Array.from(addedFiles || []);
                if (this.busy || !files.length) return;
                if (files.reduce((total, file) => total + file.size, 0) > 32 * 1024 * 1024) { this.$notify('The upload exceeds 32 MiB.', 'error'); return; }
                this.busy = true;
                try {
                    const body = new FormData();
                    for (const file of files) body.append('files', file);
                    await this.$api(this.$filePath('uploads', this.directory), {method: 'POST', body, signal: life.signal});
                    if (life.signal.aborted) return;
                    await this.refresh();
                    this.$notify(`${files.length} file(s) uploaded.`);
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Upload failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
        };
    });
}, {once: true});
