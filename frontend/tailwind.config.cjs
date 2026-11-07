module.exports = {
    content: ['./assets/monitor.template', './assets/observatory-layout.template', './assets/js/observatory.js'],
    theme: {
        extend: {
            opacity: {15: '.15', 35: '.35', 45: '.45', 65: '.65', 85: '.85'},
        },
    },
    plugins: [require('daisyui')],
    daisyui: {themes: ['business', 'winter'], darkTheme: 'business', logs: false},
};
