document.addEventListener('alpine:init', () => {
    Alpine.data('loginPage', () => {
        let life;
        return {
            username: '', password: '', busy: false, error: '',
            init() {
                JSON.parse(this.$el.dataset.page || '{}');
                life = Alpine.store('helpers').lifecycle();
            },
            destroy() { this.password = ''; life?.dispose(); },
            async submit() {
                if (this.busy) return;
                if (!this.username || !this.password) { this.error = '请输入用户名和密码。'; return; }
                this.busy = true;
                this.error = '';
                try {
                    const response = await fetch(location.pathname, {
                        method: 'POST', credentials: 'same-origin', signal: life.signal,
                        body: new URLSearchParams({username: this.username, password: this.password}),
                    });
                    this.password = '';
                    if (response.ok && response.redirected && new URL(response.url).origin === location.origin) {
                        location.assign('/admin/monitor');
                        return;
                    }
                    this.error = response.status === 429 ? '请求过于频繁，请稍后重试。' : '用户名或密码错误，请重新输入。';
                } catch (error) {
                    if (!life.signal.aborted) this.error = '无法连接服务器，请检查网络后重试。';
                } finally { if (!life.signal.aborted) this.busy = false; }
            },
        };
    });
}, {once: true});
