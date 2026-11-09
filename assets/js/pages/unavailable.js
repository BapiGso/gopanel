document.addEventListener('alpine:init', () => {
    Alpine.data('unavailablePage', () => ({
        get heading() { return Alpine.store('helpers').heading('Feature unavailable', 'Your operating system does not support this feature. Other GoPanel services remain available.'); },
        init() { JSON.parse(this.$el.dataset.page || '{}'); },
    }));
}, {once: true});
