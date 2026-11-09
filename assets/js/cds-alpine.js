(() => {
    const models = {
        'CDS-TEXT-INPUT': ['value', 'input'],
        'CDS-PASSWORD-INPUT': ['value', 'input'],
        'CDS-SEARCH': ['value', 'input'],
        'CDS-TABLE-TOOLBAR-SEARCH': ['value', 'cds-search-input'],
        'CDS-TEXTAREA': ['value', 'input'],
        'CDS-TOGGLE': ['toggled', 'cds-toggle-changed'],
        'CDS-CHECKBOX': ['checked', 'cds-checkbox-changed'],
        'CDS-SELECT': ['value', 'cds-select-selected'],
        'CDS-DROPDOWN': ['value', 'cds-dropdown-selected'],
        'CDS-COMBO-BOX': ['value', 'cds-combo-box-selected'],
        'CDS-NUMBER-INPUT': ['value', 'cds-number-input'],
    };

    const lifecycle = () => {
        const controller = new AbortController();
        const cleanups = [];
        let disposed = false;
        const life = {
            signal: controller.signal,
            add(cleanup) {
                if (disposed) cleanup(); else cleanups.push(cleanup);
                return cleanup;
            },
            listen(target, event, handler, options) {
                target.addEventListener(event, handler, options);
                this.add(() => target.removeEventListener(event, handler, options));
            },
            interval(handler, delay) {
                let running = false;
                const identity = setInterval(async () => {
                    if (disposed || running) return;
                    running = true;
                    try { await handler(); } finally { running = false; }
                }, delay);
                this.add(() => clearInterval(identity));
                return identity;
            },
            timeout(handler, delay) {
                const identity = setTimeout(() => { if (!disposed) handler(); }, delay);
                this.add(() => clearTimeout(identity));
                return identity;
            },
            observe(element, handler) {
                const observer = new ResizeObserver(handler);
                observer.observe(element);
                this.add(() => observer.disconnect());
                return observer;
            },
            dispose() {
                if (disposed) return;
                disposed = true;
                controller.abort();
                for (const cleanup of cleanups.splice(0).reverse()) cleanup();
            },
        };
        life.listen(globalThis, 'pagehide', () => life.dispose(), {once: true});
        return life;
    };

    const bytes = value => {
        const count = Number(value);
        if (!Number.isFinite(count) || count <= 0) return '0 B';
        const index = Math.min(4, Math.floor(Math.log(count) / Math.log(1024)));
        return `${(count / 1024 ** index).toFixed(index ? 1 : 0)} ${['B', 'KiB', 'MiB', 'GiB', 'TiB'][index]}`;
    };
    const token = name => getComputedStyle(document.documentElement).getPropertyValue(`--cds-${name}`).trim();

    document.addEventListener('alpine:init', () => {
        Alpine.directive('cds-prop', (element, {value, expression}, {effect, evaluateLater, cleanup}) => {
            const property = value.replace(/-([a-z])/g, (match, letter) => letter.toUpperCase());
            const read = evaluateLater(expression);
            let latest;
            let alive = true;
            const apply = () => { if (alive && customElements.get(element.localName)) element[property] = latest; };
            effect(() => read(result => { latest = result; apply(); }));
            customElements.whenDefined(element.localName).then(apply);
            cleanup(() => { alive = false; });
        });
        Alpine.directive('cds-model', (element, {expression, modifiers}, {effect, evaluate, evaluateLater, cleanup}) => {
            const [property, event] = models[element.tagName] || ['value', 'input'];
            const read = evaluateLater(expression);
            let latest;
            let alive = true;
            const apply = () => {
                if (alive && customElements.get(element.localName) && element[property] !== latest) element[property] = latest;
            };
            effect(() => read(result => { latest = result; apply(); }));
            customElements.whenDefined(element.localName).then(apply);
            const changed = event => {
                let value = event.detail?.item?.value ?? event.detail?.value ?? element[property];
                if (property === 'checked') value = event.detail?.checked ?? element.checked;
                if (property === 'toggled') value = event.detail?.toggled ?? element.toggled;
                if (modifiers.includes('number') && value !== '') value = Number(value);
                evaluate(`${expression} = __cdsValue`, {scope: {__cdsValue: value}});
            };
            element.addEventListener(event, changed);
            cleanup(() => { alive = false; element.removeEventListener(event, changed); });
        });

        const toastTimers = new Map();
        let toastIdentity = 0;
        Alpine.store('toast', {
            items: [],
            push(message, kind = 'success', title = 'GoPanel') {
                const id = ++toastIdentity;
                this.items.push({id, message: String(message), kind, title});
                toastTimers.set(id, setTimeout(() => this.remove(id), 8000));
            },
            remove(id) {
                clearTimeout(toastTimers.get(id));
                toastTimers.delete(id);
                this.items = this.items.filter(item => item.id !== id);
            },
        });
        let resolveConfirmation;
        Alpine.store('confirm', {
            open: false, title: '', message: '', danger: false,
            request({title = 'Confirm action', message = '', danger = false}) {
                this.finish(false);
                this.title = title;
                this.message = message;
                this.danger = danger;
                this.open = true;
                return new Promise(resolve => { resolveConfirmation = resolve; });
            },
            finish(accepted) {
                this.open = false;
                const resolve = resolveConfirmation;
                resolveConfirmation = undefined;
                resolve?.(accepted);
            },
        });
        addEventListener('pagehide', () => {
            for (const timer of toastTimers.values()) clearTimeout(timer);
            toastTimers.clear();
            Alpine.store('confirm').finish(false);
        }, {once: true});
        Alpine.magic('notify', () => (message, kind, title) => Alpine.store('toast').push(message, kind, title));
        Alpine.magic('ask', () => options => Alpine.store('confirm').request(options));

        Alpine.store('helpers', {
            lifecycle, bytes,
            heading(title, description, options = {}) {
                return {title, description, status: '', type: 'gray', meta: '', actions: [], ...options};
            },
            date(value) {
                const date = new Date(value);
                return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString();
            },
            editor(element, options, life) {
                const editor = ace.edit(element);
                editor.setOptions({fontSize: getComputedStyle(element).getPropertyValue('--cds-code-02-font-size').trim(), showPrintMargin: false, wrap: true, fontFamily: 'IBM Plex Mono, monospace'});
                editor.session.setUseWorker(false);
                editor.session.setMode(`ace/mode/${options.mode || 'text'}`);
                const sync = () => {
                    editor.setTheme({cssClass: 'ace-carbon', cssText: '', isDark: document.documentElement.dataset.carbonTheme === 'g100'});
                    editor.resize();
                };
                sync();
                life.listen(document, 'carbon-theme-change', sync);
                life.listen(globalThis, 'resize', () => editor.resize());
                life.listen(globalThis, 'orientationchange', () => editor.resize());
                life.observe(element, () => editor.resize());
                if (options.save) editor.commands.addCommand({name: 'save', bindKey: {win: 'Ctrl-S', mac: 'Command-S'}, exec: options.save});
                life.add(() => editor.destroy());
                return editor;
            },
            xtermTheme() {
                return {
                    background: token('background'), foreground: token('text-primary'), cursor: token('focus'), cursorAccent: token('background'),
                    selectionBackground: token('layer-selected'), black: token('background'), red: token('support-error'), green: token('support-success'),
                    yellow: token('support-warning'), blue: token('interactive'), magenta: token('link-visited'), cyan: token('support-info'), white: token('text-primary'),
                    brightBlack: token('text-secondary'), brightRed: token('support-error-inverse'), brightGreen: token('support-success-inverse'),
                    brightYellow: token('support-warning-inverse'), brightBlue: token('link-primary'), brightMagenta: token('link-visited'), brightCyan: token('support-info-inverse'), brightWhite: token('text-primary'),
                };
            },
        });

        Alpine.data('shellPage', () => {
            let life;
            return {
                navOpen: false, wide: false,
                init() {
                    life = lifecycle();
                    const width = matchMedia('(min-width: 66rem)');
                    const sync = () => { this.wide = width.matches; this.navOpen = false; };
                    sync();
                    life.listen(width, 'change', sync);
                    life.listen(document, 'keydown', event => { if (event.key === 'Escape') this.navOpen = false; });
                    if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(() => {});
                },
                destroy() { life?.dispose(); },
            };
        });
    }, {once: true});
})();
