document.addEventListener('alpine:init', () => {
    Alpine.data('dockerPage', () => {
        let life, stream;
        return {
            data: {containers: [], images: []}, tab: 'containers', search: '', connected: false, error: '', busy: false,
            get heading() { return Alpine.store('helpers').heading('Docker', 'Manage local containers and images through the system Docker API.', {status: this.connected ? 'Live' : 'Disconnected', type: this.connected ? 'green' : 'gray'}); },
            get filteredContainers() { const query = this.search.toLowerCase(); return this.data.containers.filter(container => [container.Id, container.Image, ...(container.Names || [])].join(' ').toLowerCase().includes(query)); },
            get filteredImages() { const query = this.search.toLowerCase(); return this.data.images.filter(image => [image.Id, ...(image.RepoTags || [])].join(' ').toLowerCase().includes(query)); },
            init() {
                JSON.parse(this.$el.dataset.page);
                life = Alpine.store('helpers').lifecycle();
                life.add(() => this.closeStream());
                life.listen(document, 'visibilitychange', () => { if (document.hidden) this.closeStream(); else { this.read(); this.connect(); } });
                this.read();
                this.connect();
            },
            destroy() { life?.dispose(); },
            containerName(container) { return (container.Names || []).map(name => name.replace(/^\//, '')).join(', ') || container.Id.slice(0, 12); },
            closeStream() { if (stream) { stream.onopen = stream.onmessage = stream.onerror = null; stream.close(); stream = null; } this.connected = false; },
            connect() {
                if (life.signal.aborted || document.hidden || stream) return;
                stream = new EventSource('/api/v1/docker/events');
                stream.onopen = () => { this.connected = true; };
                stream.onmessage = event => {
                    if (life.signal.aborted) return;
                    try { const payload = JSON.parse(event.data); if (payload.data) this.data = payload.data; this.error = payload.error?.message || ''; }
                    catch { this.error = 'The Docker status stream returned invalid data.'; }
                };
                stream.onerror = () => { this.connected = false; this.error = 'Docker status stream disconnected. Retrying…'; };
            },
            async read() {
                try { const data = await this.$api('/docker/status', {signal: life.signal}); if (!life.signal.aborted) { this.data = data; this.error = ''; } }
                catch (error) { if (!life.signal.aborted) { this.error = error.message; if (error.status === 401) this.closeStream(); } }
            },
            portURL(port) {
                if (!Number.isInteger(port.PublicPort) || port.PublicPort < 1 || port.PublicPort > 65535) return '';
                let host = port.IP && !['0.0.0.0', '::'].includes(port.IP) ? port.IP : location.hostname;
                if (host !== location.hostname && !/^[\da-f:.]+$/i.test(host)) return '';
                if (host.includes(':') && !host.startsWith('[')) host = '[' + host + ']';
                return `http://${host}:${port.PublicPort}`;
            },
            async act(action, resource) {
                if (this.busy) return;
                const remove = ['delete', 'deleteImage'].includes(action);
                const image = action === 'deleteImage';
                if (remove && !await this.$ask({title: image ? 'Delete Docker image?' : 'Delete container?', message: image ? 'Remove this local image without forcing deletion. Images in use cannot be removed.' : `Remove ${this.containerName(resource)} without forcing deletion. Running containers must be stopped first.`, danger: true})) return;
                this.busy = true;
                try {
                    const path = `/docker/${image ? 'images' : 'containers'}/${encodeURIComponent(resource.Id)}${remove ? '' : '/' + action}`;
                    await this.$api(path, {method: remove ? 'DELETE' : 'POST', signal: life.signal});
                    if (life.signal.aborted) return;
                    await this.read();
                    this.$notify(remove ? 'Resource deleted.' : `${action} completed.`);
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Docker operation failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
        };
    });
}, {once: true});
