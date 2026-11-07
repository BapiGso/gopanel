window.panelAPI = async function (path, options = {}) {
    const request = {credentials: 'same-origin', ...options};
    if (request.body !== undefined && !(request.body instanceof FormData)) {
        request.headers = {'Content-Type': 'application/json', ...request.headers};
        request.body = JSON.stringify(request.body);
    }
    const response = await fetch(`/api/v1${path}`, request);
    const payload = await response.json();
    if (!response.ok || payload.error) {
        const error = new Error(payload.error?.message || `HTTP ${response.status}`);
        error.code = payload.error?.code;
        error.status = response.status;
        throw error;
    }
    return payload.data;
};

window.panelFilePath = function (resource, path) {
    return `/${resource}${path.split('/').map(encodeURIComponent).join('/')}`;
};
