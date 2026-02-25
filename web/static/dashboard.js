// Dashboard JS — IIFE-wrapped module
(function() {
    'use strict';

    var CSS_RE = /^[\w.-]+\.css$/;

    // Theme switching via id="theme-link"
    function changeStyle(newCSS) {
        if (!CSS_RE.test(newCSS)) return;

        var themeLink = document.getElementById('theme-link');
        if (themeLink) {
            themeLink.href = '/static/' + newCSS;
        } else {
            var link = document.createElement('link');
            link.rel = 'stylesheet';
            link.id = 'theme-link';
            link.href = '/static/' + newCSS;
            document.head.appendChild(link);
        }
        localStorage.setItem('preferredCSS', newCSS);
    }

    // Refresh functionality
    var dropdown = document.getElementById('refresh-select');
    if (dropdown) {
        var saved = localStorage.getItem('refresh-interval');
        if (!saved) {
            saved = '300000';
            localStorage.setItem('refresh-interval', saved);
        }
        dropdown.value = saved;

        var timer = null;

        function setRefresh(interval) {
            if (timer) clearInterval(timer);
            if (interval > 0) {
                timer = setInterval(function() {
                    window.location.reload();
                }, interval);
            }
        }

        dropdown.addEventListener('change', function() {
            var interval = parseInt(dropdown.value, 10);
            localStorage.setItem('refresh-interval', dropdown.value);
            setRefresh(interval);
        });

        setRefresh(parseInt(dropdown.value, 10));
    }

    // Theme dropdown init + event listener (replaces inline onchange)
    document.addEventListener('DOMContentLoaded', function() {
        var select = document.getElementById('style-select');
        if (select) {
            var currentCSS = localStorage.getItem('preferredCSS') || 'auto.css';
            select.value = currentCSS;

            select.addEventListener('change', function() {
                changeStyle(select.value);
            });
        }
    });

    // Avatar fallback — hide broken images
    document.addEventListener('error', function(e) {
        if (e.target.tagName === 'IMG' &&
            (e.target.classList.contains('author-avatar') || e.target.classList.contains('reviewer-avatar'))) {
            e.target.style.display = 'none';
        }
    }, true);
})();
