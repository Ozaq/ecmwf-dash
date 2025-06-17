function changeStyle(select) {
    const newCSS = select.value;
    const link = document.querySelector('link[rel="stylesheet"]');
    
    // Force reload by adding timestamp to prevent caching
    link.href = `static/${newCSS}?t=${Date.now()}`;
    
    // Apply theme class to body for CSS variable overrides
    document.body.className = '';
    if (newCSS.includes('cyberpunk')) {
        document.body.className = 'cyberpunk-theme';
    } else if (newCSS.includes('retro')) {
        document.body.className = 'retro-theme';
    }
    
    localStorage.setItem('preferredCSS', newCSS);
}

// Immediately set CSS before page renders to prevent flicker
(function() {
    let savedCSS = localStorage.getItem('preferredCSS');
    
    // Set default CSS on first visit
    if (!savedCSS) {
        savedCSS = 'default.css';
        localStorage.setItem('preferredCSS', savedCSS);
    }
    
    // Update the stylesheet link immediately
    const link = document.querySelector('link[rel="stylesheet"]');
    if (link && link.href !== `static/${savedCSS}`) {
        link.href = `static/${savedCSS}`;
    }
    
    // Apply theme class immediately
    if (savedCSS.includes('cyberpunk')) {
        document.body.className = 'cyberpunk-theme';
    } else if (savedCSS.includes('retro')) {
        document.body.className = 'retro-theme';
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
        
        // Set the dropdown to match current CSS
        const currentCSS = localStorage.getItem('preferredCSS') || 'default.css';
        select.value = currentCSS;
    }
});