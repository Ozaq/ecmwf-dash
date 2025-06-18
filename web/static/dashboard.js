function changeStyle(select) {
    const newCSS = select.value;
    const link = document.querySelector('link[rel="stylesheet"]');
    // Force reload by adding timestamp to prevent caching
    link.href = `static/${newCSS}?t=${Date.now()}`;
    localStorage.setItem('preferredCSS', newCSS);
}

// Immediately set CSS before page renders to prevent flicker
(function() {
    let savedCSS = localStorage.getItem('preferredCSS');
    
    // Set default CSS on first visit
    if (!savedCSS) {
        savedCSS = 'auto.css';
        localStorage.setItem('preferredCSS', savedCSS);
    }
    
    // Update the stylesheet link immediately
    const link = document.querySelector('link[rel="stylesheet"]');
    if (link && link.href !== `static/${savedCSS}`) {
        link.href = `static/${savedCSS}`;
    }
})();

// Refresh functionality
(function() {
    const dropdown = document.getElementById("refresh-select");
    let saved = localStorage.getItem("refresh-interval");
    
    // Set default refresh interval on first visit
    if (!saved) {
        saved = "300000"; // 5 minutes
        localStorage.setItem("refresh-interval", saved);
    }
    
    dropdown.value = saved;

    let timer = null;

    function setRefresh(interval) {
        if (timer) clearInterval(timer);
        if (interval > 0) {
            timer = setInterval(() => {
                window.location.reload();
            }, interval);
        }
    }

    dropdown.addEventListener("change", () => {
        const interval = parseInt(dropdown.value, 10);
        localStorage.setItem("refresh-interval", dropdown.value);
        setRefresh(interval);
    });

    setRefresh(parseInt(dropdown.value, 10));
})();

// Load preferred style on page load
window.addEventListener('DOMContentLoaded', function() {
    // Populate theme selector with CSS files without .css extension
    const select = document.getElementById('style-select');
    if (select) {
        // Update option display text to remove .css extension
        Array.from(select.options).forEach(option => {
            if (option.value.endsWith('.css')) {
                option.textContent = option.value.replace('.css', '');
            }
        });
        
        // Set the dropdown to match current CSS (including default for first-time users)
        const currentCSS = localStorage.getItem('preferredCSS') || 'auto.css';
        select.value = currentCSS;
        
        // Ensure the CSS is set correctly if this is the first visit
        if (!localStorage.getItem('preferredCSS')) {
            localStorage.setItem('preferredCSS', 'auto.css');
        }
    }
});