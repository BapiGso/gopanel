document.addEventListener('alpine:init', () => {
    Alpine.data('termPage', () => {
        let life, root;
        const resources = new Map();
        return {
            form: {host: '127.0.0.1', port: 22, username: 'root', password: ''}, remember: true, hasSaved: false, connecting: false, sessions: [], active: '',
            get heading() { return Alpine.store('helpers').heading('Terminal', 'Open SSH sessions without leaving the panel.'); },
            init() {
                root = this.$el;
                JSON.parse(root.dataset.page);
                life = Alpine.store('helpers').lifecycle();
                try {
                    localStorage.removeItem('ssh_pwd');
                    if (localStorage.getItem('ssh_host')) {
                        this.form.host = localStorage.getItem('ssh_host');
                        this.form.port = Number(localStorage.getItem('ssh_port')) || 22;
                        this.form.username = localStorage.getItem('ssh_user') || 'root';
                        this.hasSaved = true;
                    }
                } catch {}
                life.listen(globalThis, 'resize', () => this.fitActive());
                life.listen(globalThis, 'orientationchange', () => this.fitActive());
                life.listen(document, 'carbon-theme-change', () => {
                    for (const resource of resources.values()) resource.terminal.options.theme = Alpine.store('helpers').xtermTheme();
                });
                life.add(() => { this.form.password = ''; for (const resource of resources.values()) resource.life.dispose(); resources.clear(); });
            },
            destroy() { life?.dispose(); },
            async connect() {
                if (this.connecting || !this.form.host || !this.form.username) return;
                if (!Number.isInteger(this.form.port) || this.form.port < 1 || this.form.port > 65535) { this.$notify('Enter a port from 1 to 65535.', 'error'); return; }
                this.connecting = true;
                const label = `${this.form.username}@${this.form.host}:${this.form.port}`;
                try {
                    const created = await this.$api('/terminals', {method: 'POST', body: {...this.form, rows: 24, cols: 80}, signal: life.signal});
                    if (life.signal.aborted) { fetch(`/api/v1/terminals/${encodeURIComponent(created.id)}`, {method: 'DELETE', credentials: 'same-origin', keepalive: true}).catch(() => {}); return; }
                    if (!created.id) throw new Error('Invalid terminal response.');
                    if (this.remember) {
                        try { localStorage.setItem('ssh_host', this.form.host); localStorage.setItem('ssh_port', String(this.form.port)); localStorage.setItem('ssh_user', this.form.username); this.hasSaved = true; } catch {}
                    }
                    this.form.password = '';
                    this.sessions.push({id: created.id, label, state: 'connecting', controlPending: false});
                    this.active = created.id;
                    await this.$nextTick();
                    if (!life.signal.aborted) this.openTerminal(created.id);
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Connection failed'); }
                finally { this.form.password = ''; if (!life.signal.aborted) this.connecting = false; }
            },
            openTerminal(id) {
                const element = Array.from(root.querySelectorAll('[data-session]')).find(element => element.dataset.session === id);
                const session = this.sessions.find(session => session.id === id);
                const sessionLife = Alpine.store('helpers').lifecycle();
                const socket = new WebSocket(`${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/api/v1/terminals/${encodeURIComponent(id)}/data`);
                const terminal = new Terminal({fontFamily: 'IBM Plex Mono, monospace', fontSize: Number.parseFloat(getComputedStyle(element).getPropertyValue('--cds-code-02-font-size')), cursorBlink: true, theme: Alpine.store('helpers').xtermTheme()});
                const fit = new FitAddon.FitAddon();
                terminal.loadAddon(fit);
                terminal.loadAddon(new WebLinksAddon.WebLinksAddon());
                terminal.loadAddon(new AttachAddon.AttachAddon(socket, {bidirectional: false}));
                terminal.open(element);
                const resource = {life: sessionLife, terminal, fit, socket};
                resources.set(id, resource);
                sessionLife.add(() => {
                    socket.close();
                    terminal.dispose();
                    fetch(`/api/v1/terminals/${encodeURIComponent(id)}`, {method: 'DELETE', credentials: 'same-origin', keepalive: true}).catch(() => {});
                });
                sessionLife.listen(socket, 'open', () => { session.state = 'connected'; this.fitActive(); terminal.focus(); });
                sessionLife.listen(socket, 'close', () => { session.state = 'closed'; });
                sessionLife.listen(socket, 'error', () => { session.state = 'error'; this.$notify('Terminal connection closed or failed. Open a new session to reconnect.', 'error'); });
                const input = terminal.onData(data => {
                    if (session.controlPending && data) { data = String.fromCharCode(data[0].toUpperCase().charCodeAt(0) & 31) + data.slice(1); session.controlPending = false; }
                    if (socket.readyState === WebSocket.OPEN) socket.send(data);
                });
                const resize = terminal.onResize(({cols, rows}) => {
                    this.$api(`/terminals/${encodeURIComponent(id)}`, {method: 'PATCH', body: {cols, rows}, signal: sessionLife.signal}).catch(error => { if (!sessionLife.signal.aborted && socket.readyState === WebSocket.OPEN) this.$notify(error.message, 'error', 'Resize failed'); });
                });
                sessionLife.add(() => { input.dispose(); resize.dispose(); });
                sessionLife.observe(element, () => { if (this.active === id) fit.fit(); });
                sessionLife.timeout(() => { if (this.active === id) fit.fit(); }, 200);
            },
            activate(id) { this.active = id; this.$nextTick(() => this.fitActive()); },
            fitActive() { const resource = resources.get(this.active); if (resource && !resource.life.signal.aborted) resource.fit.fit(); },
            sendKey(id, key) { const resource = resources.get(id); if (resource?.socket.readyState === WebSocket.OPEN) { resource.socket.send(key); resource.terminal.focus(); } },
            toggleControl(session) { session.controlPending = !session.controlPending; resources.get(session.id)?.terminal.focus(); },
            async closeSession(id) {
                if (!await this.$ask({title: 'Close SSH session?', message: 'The shell connection will be terminated.', danger: true})) return;
                resources.get(id)?.life.dispose();
                resources.delete(id);
                this.sessions = this.sessions.filter(session => session.id !== id);
                if (this.active === id) this.activate(this.sessions[0]?.id || '');
            },
            async forget() {
                if (!await this.$ask({title: 'Forget saved connection?', message: 'Only the remembered host, port, and username will be removed.'})) return;
                try { for (const name of ['ssh_host', 'ssh_port', 'ssh_user', 'ssh_pwd']) localStorage.removeItem(name); } catch {}
                this.hasSaved = false;
            },
        };
    });
}, {once: true});
