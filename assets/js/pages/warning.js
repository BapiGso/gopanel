document.addEventListener('alpine:init', () => {
    Alpine.data('warningPage', () => ({
        message: '', ip: '',
        init() {
            const page = JSON.parse(this.$el.dataset.page || '{}');
            this.message = page.message || '';
            this.ip = page.ip || '';
        },
    }));
}, {once: true});
