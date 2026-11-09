(() => {
    class APIError extends Error {
        constructor(status, error = {}) {
            super(error.message || `HTTP ${status}`);
            this.name = 'APIError';
            this.status = status;
            this.code = error.code || 'request_failed';
            this.details = error.details;
        }
    }

    document.addEventListener('alpine:init', () => {
        const request = async (path, options = {}) => {
            if (!path.startsWith('/') || path.startsWith('//') || /[?\\#]/.test(path)) {
                throw new APIError(400, {code: 'invalid_path', message: 'API requests require a resource path without query parameters.'});
            }
            const url = new URL(`/api/v1${path}`, location.origin);
            if (url.origin !== location.origin || !url.pathname.startsWith('/api/v1/')) {
                throw new APIError(400, {code: 'invalid_path', message: 'Only same-origin API resources are allowed.'});
            }
            const init = {...options, credentials: 'same-origin', headers: new Headers(options.headers)};
            if (init.body !== undefined && !(init.body instanceof FormData)) {
                init.headers.set('Content-Type', 'application/json');
                init.body = JSON.stringify(init.body);
            }
            const response = await fetch(url, init);
            let payload;
            try { payload = await response.json(); } catch {
                throw new APIError(response.status, {message: 'The server returned an invalid API response.'});
            }
            if (!response.ok || payload.error) {
                const error = new APIError(response.status, payload.error);
                if (response.status === 401) Alpine.store('toast').push('Your session has expired. Sign in through your configured login path.', 'error', 'Session expired');
                throw error;
            }
            return payload.data;
        };
        const filePath = (resource, path) => `/${resource}${('/' + path.replace(/^\/+/, '')).split('/').map(encodeURIComponent).join('/')}`;
        Alpine.magic('api', () => request);
        Alpine.magic('filePath', () => filePath);
    }, {once: true});
})();
