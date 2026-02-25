// Dashboard JS — IIFE-wrapped module
(function() {
    'use strict';

    // Auto-refresh every 60s — fetch-and-replace to preserve scroll + expanded state
    // TV mode has its own countdown-based reload, so skip it here.
    if (!document.documentElement.classList.contains('tv-mode')) {
        function scheduleRefresh() {
            setTimeout(function() {
                fetch(window.location.href).then(function(resp) {
                    if (!resp.ok) throw new Error('HTTP ' + resp.status);
                    return resp.text();
                }).then(function(html) {
                    var currentContainer = document.querySelector('.container');
                    if (!currentContainer) return;

                    var doc = new DOMParser().parseFromString(html, 'text/html');
                    var newContainer = doc.querySelector('.container');
                    if (!newContainer) return;

                    // Capture state just before swap (minimizes race window)
                    var scrollY = window.scrollY;
                    var expandedRepos = [];
                    var expanded = currentContainer.querySelectorAll('.build-row.is-expanded');
                    for (var i = 0; i < expanded.length; i++) {
                        var repo = expanded[i].querySelector('.build-repo');
                        if (repo) expandedRepos.push(repo.textContent.trim());
                    }

                    // Swap DOM nodes (no innerHTML — adopt parsed nodes directly)
                    while (currentContainer.firstChild) {
                        currentContainer.removeChild(currentContainer.firstChild);
                    }
                    while (newContainer.firstChild) {
                        currentContainer.appendChild(document.adoptNode(newContainer.firstChild));
                    }

                    // Restore expanded build rows
                    if (expandedRepos.length > 0) {
                        var rows = currentContainer.querySelectorAll('.build-row.has-details');
                        for (var j = 0; j < rows.length; j++) {
                            var repoEl = rows[j].querySelector('.build-repo');
                            if (repoEl && expandedRepos.indexOf(repoEl.textContent.trim()) !== -1) {
                                rows[j].classList.add('is-expanded');
                                rows[j].setAttribute('aria-expanded', 'true');
                            }
                        }
                    }

                    // Restore scroll position
                    window.scrollTo(0, scrollY);
                }).catch(function(err) {
                    console.warn('Auto-refresh failed:', err);
                }).then(scheduleRefresh, scheduleRefresh);
            }, 60000);
        }
        scheduleRefresh();
    }

    // Avatar fallback — hide broken images
    document.addEventListener('error', function(e) {
        if (e.target.tagName === 'IMG' &&
            (e.target.classList.contains('author-avatar') || e.target.classList.contains('reviewer-avatar'))) {
            e.target.style.display = 'none';
        }
    }, true);

    // Build row expand/collapse (event delegation)
    document.addEventListener('click', function(e) {
        if (e.target.closest('a')) return;
        var row = e.target.closest('.build-row.has-details');
        if (!row) return;
        var expanded = row.classList.toggle('is-expanded');
        row.setAttribute('aria-expanded', expanded);
    });

    // Repo filter — auto-submit on change (CSP-compliant, no inline handler)
    document.addEventListener('change', function(e) {
        if (e.target.id !== 'repo-filter') return;
        var form = e.target.closest('form');
        if (form) form.submit();
    });

    document.addEventListener('keydown', function(e) {
        if (e.key !== 'Enter' && e.key !== ' ') return;
        var row = e.target.closest('.build-row.has-details');
        if (!row) return;
        if (e.target.closest('a')) return;
        e.preventDefault();
        var expanded = row.classList.toggle('is-expanded');
        row.setAttribute('aria-expanded', expanded);
    });

    // Dashboard mode: relative update time + countdown refresh + adaptive grid
    if (document.documentElement.classList.contains('tv-mode')) {
        var REFRESH_MS = 60000;
        var countdownEl = document.getElementById('tv-countdown');

        // Align to the next whole minute boundary
        var now = Date.now();
        var msToNextMinute = REFRESH_MS - (now % REFRESH_MS);
        var reloadAt = now + msToNextMinute;

        // Build DOM structure once so we only update the number text
        var numSpan = null;
        if (countdownEl) {
            countdownEl.textContent = '';
            var pre = document.createTextNode('Refresh in ');
            numSpan = document.createElement('span');
            numSpan.className = 'tv-countdown-num';
            var suf = document.createTextNode('s');
            countdownEl.appendChild(pre);
            countdownEl.appendChild(numSpan);
            countdownEl.appendChild(suf);
        }

        function tick() {
            var left = Math.max(0, Math.ceil((reloadAt - Date.now()) / 1000));
            if (numSpan) {
                numSpan.textContent = left;
            }
            if (left <= 0) {
                window.location.reload();
            }
        }

        tick();
        setInterval(tick, 1000);

        // Viewport-adaptive grid: compute optimal cols/rows for card count
        var cards = document.querySelectorAll('.build-card');
        var n = cards.length;
        if (n > 0) {
            function computeGrid() {
                var main = document.querySelector('.tv-main');
                if (!main) return;
                var w = main.clientWidth;
                var h = main.clientHeight;

                var bestCols = 1, bestScore = Infinity;
                for (var c = 1; c <= n; c++) {
                    var r = Math.ceil(n / c);
                    var cellW = w / c;
                    var cellH = h / r;
                    var cellAspect = cellW / cellH;
                    // Target ~1.6:1 aspect ratio (landscape cards suit horizontal content)
                    var score = Math.abs(cellAspect - 1.6);
                    if (score < bestScore) {
                        bestScore = score;
                        bestCols = c;
                    }
                }
                var bestRows = Math.ceil(n / bestCols);

                document.documentElement.style.setProperty('--tv-cols', bestCols);
                document.documentElement.style.setProperty('--tv-rows', bestRows);
            }

            computeGrid();
            window.addEventListener('resize', computeGrid);
        }
    }
})();
