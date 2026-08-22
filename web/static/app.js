let backToTop;

function initBackToTop() {
    backToTop = document.querySelector('.back-to-top');
    if (!backToTop) return;
    backToTop.addEventListener('click', function () {
        window.scrollTo({top: 0, behavior: 'smooth'});
    });
}

window.addEventListener('scroll', function () {
    if (backToTop) backToTop.classList.toggle('visible', window.scrollY > 360);
}, {passive: true});

document.addEventListener('DOMContentLoaded', initBackToTop);

function refreshRepositories() {
    const sort = document.querySelector('.sort-select').value;
    const direction = document.querySelector('.direction-toggle').closest('form').elements.direction.value;
    const search = document.querySelector('.repository-search input[name="q"]').value;
    htmx.ajax('GET', '/repositories/cards?sort=' + encodeURIComponent(sort) + '&direction=' + direction + '&q=' + encodeURIComponent(search), {
        target: '#repository-cards',
        swap: 'innerHTML'
    });
}

document.addEventListener('change', function (event) {
    if (event.target.matches('.sort-select')) refreshRepositories();
});

document.addEventListener('click', function (event) {
    const button = event.target.closest('.direction-toggle');
    if (!button) return;
    const form = button.closest('form');
    const direction = form.elements.direction;
    direction.value = direction.value === 'desc' ? 'asc' : 'desc';
    button.textContent = direction.value === 'desc' ? '↓' : '↑';
    button.setAttribute('aria-label', direction.value === 'desc' ? 'Sort descending' : 'Sort ascending');
    refreshRepositories();
});

function syncStatusRequest() {
    htmx.ajax('GET', '/repositories/sync/status', {
        target: '#sync-status',
        swap: 'innerHTML'
    });
}

document.body.addEventListener('htmx:beforeRequest', function (event) {
    const path = event.detail.pathInfo && event.detail.pathInfo.requestPath;
    if (path === '/repositories/sync' || path === '/repositories/sync/status') {
        document.getElementById('sync-progress').classList.add('active');
    }
});

document.body.addEventListener('htmx:afterRequest', function (event) {
    const path = event.detail.pathInfo && event.detail.pathInfo.requestPath;
    if (path === '/repositories/sync' && event.detail.successful) {
        syncStatusRequest();
    }
});

document.body.addEventListener('repositories-updated', function () {
    htmx.ajax('GET', '/repositories/cards', {target: '#repository-cards', swap: 'innerHTML'});
});

document.body.addEventListener('htmx:afterSwap', function (event) {
    const path = event.detail.pathInfo && event.detail.pathInfo.requestPath;
    const target = event.detail.target;
    if (path !== '/repositories/sync/status' && !(target && (target.id === 'sync-status' || target.closest('#sync-status')))) return;
    const running = document.querySelector('#sync-status [data-sync-running="true"]');
    const done = document.querySelector('#sync-status [data-sync-done="true"]');
    document.getElementById('sync-progress').classList.toggle('active', !!running);
    if (running && path === '/repositories/sync/status') setTimeout(syncStatusRequest, 300);
    if (done) {
        htmx.ajax('GET', '/repositories/cards', {target: '#repository-cards', swap: 'innerHTML'});
    }
});
