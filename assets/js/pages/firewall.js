document.addEventListener('alpine:init', () => {
    Alpine.data('firewallPage', () => {
        let life;
        return {
            rules: [], search: '', form: {}, createOpen: false, busy: false,
            get heading() { return Alpine.store('helpers').heading('Firewall', 'Manage panel-owned firewall rules.'); },
            get filteredRules() { const query = this.search.toLowerCase(); return this.rules.filter(rule => Object.values(rule).join(' ').toLowerCase().includes(query)); },
            init() { this.rules = JSON.parse(this.$el.dataset.page) || []; life = Alpine.store('helpers').lifecycle(); },
            destroy() { life?.dispose(); },
            openCreate() { this.form = {port: 0, action: 'accept', network: 'ipv4', protocol: 'tcp', direction: 'in'}; this.createOpen = true; },
            async read() {
                try { const rules = await this.$api('/firewall/rules', {signal: life.signal}); if (!life.signal.aborted) this.rules = rules; }
                catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error'); }
            },
            async create() {
                if (this.busy) return;
                if (!Number.isInteger(this.form.port) || this.form.port < 0 || this.form.port > 65535) { this.$notify('Enter an integer port from 0 to 65535.', 'error'); return; }
                this.busy = true;
                try {
                    const rules = await this.$api('/firewall/rules', {method: 'POST', body: this.form, signal: life.signal});
                    if (life.signal.aborted) return;
                    this.rules = rules;
                    this.createOpen = false;
                    this.$notify('Firewall rule added.');
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Add failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async remove(rule) {
                if (this.busy || !await this.$ask({title: 'Delete firewall rule?', message: `${rule.network} ${rule.protocol} port ${rule.port || 'any'}, ${rule.direction}, ${rule.action}. This can change network connectivity.`, danger: true})) return;
                this.busy = true;
                try {
                    await this.$api(`/firewall/rules/${encodeURIComponent(rule.id)}`, {method: 'DELETE', signal: life.signal});
                    if (life.signal.aborted) return;
                    await this.read();
                    this.$notify('Firewall rule deleted.');
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Delete failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
        };
    });
});
