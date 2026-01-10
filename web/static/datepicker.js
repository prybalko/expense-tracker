(function() {
    let viewDate = new Date();
    
    window.openDatePicker = function() {
        const input = document.getElementById('date-input');
        const val = input.value;
        if (val) viewDate = new Date(val);
        else viewDate = new Date();
        
        renderCalendar();
        const overlay = document.querySelector('.date-modal-overlay');
        if (overlay) {
            overlay.classList.add('open');
        }
    };

    window.closeDatePicker = function() {
        const overlay = document.querySelector('.date-modal-overlay');
        if (overlay) {
            overlay.classList.remove('open');
        }
    };

    window.changeMonth = function(delta) {
        viewDate.setMonth(viewDate.getMonth() + delta);
        renderCalendar();
    };

    window.selectDate = function(year, month, day) {
        const input = document.getElementById('date-input');
        // Preserve time if editing, else default to current time or 12:00:00
        let timePart = '12:00:00';
        if (input.value && input.value.includes('T')) {
            timePart = input.value.split('T')[1];
            // Ensure seconds are present
            if (timePart.length === 5) {
                timePart += ':00';
            }
        } else {
            const now = new Date();
            timePart = now.toTimeString().slice(0, 8);
        }

        const d = new Date(year, month, day);
        const dateStr = d.getFullYear() + '-' + 
                       String(d.getMonth() + 1).padStart(2, '0') + '-' + 
                       String(d.getDate()).padStart(2, '0');
        
        input.value = dateStr + 'T' + timePart;
        updateDateDisplay(input);
        closeDatePicker();
    };

    window.renderCalendar = function() {
        const year = viewDate.getFullYear();
        const month = viewDate.getMonth();
        
        const monthNames = ["January", "February", "March", "April", "May", "June",
            "July", "August", "September", "October", "November", "December"
        ];
        const monthYearEl = document.getElementById('calendar-month-year');
        if (monthYearEl) {
            monthYearEl.textContent = monthNames[month] + ' ' + year;
        }

        const firstDay = new Date(year, month, 1);
        const lastDay = new Date(year, month + 1, 0);
        const daysInMonth = lastDay.getDate();
        // Adjust for Monday start: (day + 6) % 7
        const startDay = (firstDay.getDay() + 6) % 7;

        const grid = document.getElementById('calendar-grid');
        if (!grid) return;

        // Clear existing days (keep headers)
        const headers = grid.querySelectorAll('.calendar-day-header');
        grid.innerHTML = '';
        headers.forEach(h => grid.appendChild(h));

        // Empty slots
        for (let i = 0; i < startDay; i++) {
            const div = document.createElement('div');
            div.className = 'calendar-day empty';
            grid.appendChild(div);
        }

        // Days
        const today = new Date();
        const inputEl = document.getElementById('date-input');
        const inputDate = new Date(inputEl && inputEl.value ? inputEl.value : new Date());

        for (let d = 1; d <= daysInMonth; d++) {
            const div = document.createElement('div');
            div.className = 'calendar-day';
            div.textContent = d;
            
            // Check if today
            if (year === today.getFullYear() && month === today.getMonth() && d === today.getDate()) {
                div.classList.add('today');
            }
            
            // Check if selected
            if (year === inputDate.getFullYear() && month === inputDate.getMonth() && d === inputDate.getDate()) {
                div.classList.add('selected');
            }

            div.onclick = () => selectDate(year, month, d);
            grid.appendChild(div);
        }
    };

    window.updateDateDisplay = function(input) {
        if (!input) return;
        const d = new Date(input.value || new Date());
        const isToday = d.toDateString() === new Date().toDateString();
        const displayEl = document.getElementById('date-display');
        if (displayEl) {
            displayEl.textContent = isToday ? 'Today' : d.toLocaleDateString(undefined, {day:'numeric',month:'short'});
        }
    };
})();
