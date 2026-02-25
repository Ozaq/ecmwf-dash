// Dashboard JS — IIFE-wrapped module
(function() {
    'use strict';

    // Auto-refresh every 60s (all pages)
    setInterval(function() { window.location.reload(); }, 60000);

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
