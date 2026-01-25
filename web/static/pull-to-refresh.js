(function() {
    const ptrContainer = document.getElementById('ptr-container');
    if (!ptrContainer) return;

    const ptrSpinner = ptrContainer.querySelector('.ptr-spinner');
    const threshold = 80;
    const maxPull = 120;
    
    let startY = 0;
    let currentY = 0;
    let isPulling = false;
    let isRefreshing = false;

    function getScreen() {
        return document.querySelector('.screen');
    }

    function isAtTop() {
        const screen = getScreen();
        if (!screen) return true;
        
        // Find the scrollable element within the screen
        const scrollable = screen.querySelector('.expenses') || 
                           screen.querySelector('.stats-content');
        if (scrollable) {
            return scrollable.scrollTop <= 0;
        }
        return true;
    }

    function handleTouchStart(e) {
        if (isRefreshing) return;
        if (document.body.classList.contains('modal-open')) return;
        if (!isAtTop()) return;
        
        startY = e.touches[0].clientY;
        isPulling = false;
    }

    function handleTouchMove(e) {
        if (isRefreshing) return;
        if (startY === 0) return;
        
        currentY = e.touches[0].clientY;
        const pullDistance = currentY - startY;

        // Only allow pulling down when at the top
        if (pullDistance > 0 && isAtTop()) {
            // Prevent native scroll
            e.preventDefault();
            
            isPulling = true;
            const screen = getScreen();
            
            // Apply resistance to pull
            const resistance = 0.5;
            const adjustedDistance = Math.min(pullDistance * resistance, maxPull);
            
            // Update UI
            ptrContainer.classList.add('pulling');
            ptrContainer.style.height = adjustedDistance + 'px';
            
            // Calculate rotation based on pull distance (0 to 360 degrees)
            const rotation = Math.min((adjustedDistance / threshold) * 360, 360);
            ptrSpinner.style.setProperty('--pull-rotation', rotation + 'deg');
            
            // Move screen down
            if (screen) {
                screen.classList.add('ptr-pulling');
                screen.style.setProperty('--ptr-offset', adjustedDistance + 'px');
            }
        }
    }

    function handleTouchEnd() {
        if (!isPulling) {
            startY = 0;
            return;
        }

        const pullDistance = (currentY - startY) * 0.5;
        const screen = getScreen();
        
        if (pullDistance >= threshold && !isRefreshing) {
            // Trigger refresh
            isRefreshing = true;
            ptrContainer.classList.remove('pulling');
            ptrContainer.classList.add('refreshing');
            ptrContainer.style.height = '50px';
            
            if (screen) {
                screen.classList.remove('ptr-pulling');
                screen.classList.add('ptr-releasing');
                screen.style.setProperty('--ptr-offset', '50px');
            }
            
            // Perform refresh
            doRefresh();
        } else {
            // Cancel pull
            resetPullState();
        }
        
        startY = 0;
        currentY = 0;
        isPulling = false;
    }

    function resetPullState() {
        const screen = getScreen();
        
        ptrContainer.classList.remove('pulling', 'refreshing');
        ptrContainer.style.height = '0';
        
        if (screen) {
            screen.classList.remove('ptr-pulling');
            screen.classList.add('ptr-releasing');
            screen.style.setProperty('--ptr-offset', '0px');
            
            // Clean up after transition
            setTimeout(() => {
                screen.classList.remove('ptr-releasing');
                screen.style.removeProperty('--ptr-offset');
            }, 300);
        }
        
        isRefreshing = false;
    }

    function doRefresh() {
        // Full page reload to pick up any updated static files from cache
        // The service worker uses stale-while-revalidate, so fresh files
        // are fetched in background and available on next reload
        setTimeout(function() {
            location.reload();
        }, 200);
    }

    // Only enable for standalone PWA mode (iOS home screen app)
    const isStandalone = window.navigator.standalone || 
                        window.matchMedia('(display-mode: standalone)').matches;
    
    // Enable for all mobile devices for better testing/UX
    const isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);
    
    if (isMobile || isStandalone) {
        document.addEventListener('touchstart', handleTouchStart, { passive: true });
        document.addEventListener('touchmove', handleTouchMove, { passive: false });
        document.addEventListener('touchend', handleTouchEnd, { passive: true });
    }
})();
