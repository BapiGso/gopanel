document.addEventListener('alpine:init', () => {
    Alpine.data('cronPage', () => {
        let life;
        return {
            jobs: [], search: '', busy: false, jobOpen: false, editing: false, form: {}, frequency: 'custom', scriptTemplate: 'custom', time: '00:00', day: 1, weekday: '0',
            get heading() { return Alpine.store('helpers').heading('Cron scheduler', 'Manage scheduled tasks and automation.'); },
            get filteredJobs() { const query = this.search.toLowerCase(); return this.jobs.filter(job => job.name.toLowerCase().includes(query)); },
            init() { this.jobs = JSON.parse(this.$el.dataset.page) || []; life = Alpine.store('helpers').lifecycle(); },
            destroy() { life?.dispose(); },
            openJob(job) {
                this.editing = !!job;
                this.form = job ? {name: job.name, script: job.script, schedule: job.schedule, interval_seconds: job.interval_seconds} : {name: '', script: '', schedule: '0 0 * * *', interval_seconds: 60};
                this.frequency = job?.interval_seconds ? 'interval' : 'custom';
                this.scriptTemplate = 'custom';
                this.jobOpen = true;
            },
            applyTemplate(template) {
                this.scriptTemplate = template;
                if (this.scriptTemplate === 'backup') this.form.script = 'tar -czvf name.tar.gz /home/user/mydir';
                if (this.scriptTemplate === 'curl') this.form.script = 'curl -L https://www.example.com';
            },
            async read() {
                try { const jobs = await this.$api('/cron/jobs', {signal: life.signal}); if (!life.signal.aborted) this.jobs = jobs; }
                catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error'); }
            },
            payload() {
                const body = {name: this.form.name, script: this.form.script};
                if (!/^[A-Za-z0-9._-]{1,64}$/.test(body.name) || !body.script.trim()) throw new Error('Enter a valid job name and a non-empty script.');
                if (this.frequency === 'interval') {
                    body.interval_seconds = this.form.interval_seconds;
                    if (!Number.isInteger(body.interval_seconds) || body.interval_seconds < 1 || body.interval_seconds > 31536000) throw new Error('Interval must be an integer from 1 to 31536000 seconds.');
                } else if (this.frequency === 'custom') body.schedule = this.form.schedule;
                else {
                    if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(this.time)) throw new Error('Enter an execution time in HH:MM format.');
                    const [hour, minute] = this.time.split(':').map(Number);
                    if (this.frequency === 'monthly' && (!Number.isInteger(this.day) || this.day < 1 || this.day > 31)) throw new Error('Enter a day of month from 1 to 31.');
                    body.schedule = this.frequency === 'monthly' ? `${minute} ${hour} ${this.day} * *` : this.frequency === 'weekly' ? `${minute} ${hour} * * ${this.weekday}` : `${minute} ${hour} * * *`;
                }
                return body;
            },
            async save() {
                if (this.busy) return;
                this.busy = true;
                try {
                    await this.$api(this.editing ? `/cron/jobs/${encodeURIComponent(this.form.name)}` : '/cron/jobs', {method: this.editing ? 'PUT' : 'POST', body: this.payload(), signal: life.signal});
                    if (life.signal.aborted) return;
                    this.jobOpen = false;
                    await this.read();
                    this.$notify('Job saved.');
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Save failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
            async act(action, job) {
                if (this.busy || action === 'run' && job.paused) return;
                if (action === 'delete' && !await this.$ask({title: 'Delete ' + job.name + '?', message: 'The schedule and script will be removed.', danger: true})) return;
                this.busy = true;
                try {
                    const path = `/cron/jobs/${encodeURIComponent(job.name)}`;
                    await this.$api(action === 'run' ? path + '/run' : path, {method: action === 'delete' ? 'DELETE' : action === 'run' ? 'POST' : 'PATCH', ...(action === 'pause' ? {body: {paused: !job.paused}} : {}), signal: life.signal});
                    if (life.signal.aborted) return;
                    await this.read();
                    this.$notify(action === 'run' ? 'Run queued. Acceptance does not report script completion.' : action === 'delete' ? 'Job deleted.' : job.paused ? 'Job resumed.' : 'Job paused.');
                    if (action === 'run') life.timeout(() => this.read(), 1000);
                } catch (error) { if (!life.signal.aborted) this.$notify(error.message, 'error', 'Operation failed'); }
                finally { if (!life.signal.aborted) this.busy = false; }
            },
        };
    });
}, {once: true});
