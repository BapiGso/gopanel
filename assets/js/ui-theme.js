(() => {
    const root = document.documentElement;
    let choice;
    try {
        choice = localStorage.getItem('carbon-theme');
        localStorage.removeItem('password');
    } catch {}
    let current = choice === 'white' || choice === 'g100' ? choice : 'g100';
    const apply = theme => {
        current = theme;
        root.classList.remove('dds-theme-zone-white', 'dds-theme-zone-g100');
        root.classList.add(`dds-theme-zone-${theme}`);
        root.dataset.carbonTheme = theme;
        root.style.colorScheme = theme === 'g100' ? 'dark' : 'light';
        document.dispatchEvent(new CustomEvent('carbon-theme-change', {detail: theme}));
    };
    apply(current);
    document.addEventListener('alpine:init', () => {
        const store = {
            current,
            toggle() {
                choice = this.current === 'g100' ? 'white' : 'g100';
                this.current = choice;
                try { localStorage.setItem('carbon-theme', choice); } catch {}
                apply(choice);
            },
        };
        Alpine.store('theme', store);
    }, {once: true});
})();
