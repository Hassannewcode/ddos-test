document.addEventListener('DOMContentLoaded', () => {
    const ctx = document.getElementById('rps-chart').getContext('2d');
    const chart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Requests per Second',
                data: [],
                borderColor: '#ff2d55',
                backgroundColor: 'rgba(255, 45, 85, 0.1)',
                borderWidth: 3,
                pointRadius: 0,
                tension: 0.4,
                fill: true
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    grid: {
                        color: 'rgba(255, 255, 255, 0.1)'
                    },
                    ticks: {
                        color: '#aaa'
                    }
                },
                x: {
                    grid: {
                        color: 'rgba(255, 255, 255, 0.1)'
                    },
                    ticks: {
                        color: '#aaa'
                    }
                }
            },
            plugins: {
                legend: {
                    labels: {
                        color: '#fff'
                    }
                }
            }
        }
    });
    
    const elements = {
        requestsSent: document.getElementById('requests-sent'),
        successRate: document.getElementById('success-rate'),
        error503: document.getElementById('error-503'),
        rps: document.getElementById('rps'),
        activeWorkers: document.getElementById('active-workers'),
        lastRequest: document.getElementById('last-request'),
        intensityUp: document.getElementById('intensity-up'),
        intensityDown: document.getElementById('intensity-down'),
        toggleAttack: document.getElementById('toggle-attack')
    };
    
    // Store historical data
    const rpsHistory = [];
    const maxDataPoints = 60;
    
    // Fetch stats every second
    setInterval(fetchStats, 1000);
    
    async function fetchStats() {
        try {
            const response = await fetch('/stats');
            const data = await response.json();
            
            // Update UI
            elements.requestsSent.textContent = data.requests_sent.toLocaleString();
            elements.error503.textContent = data.error_503.toLocaleString();
            elements.rps.textContent = data.rps.toFixed(1);
            elements.activeWorkers.textContent = data.active_workers.toLocaleString();
            
            const successRate = data.requests_sent > 0 ? 
                (data.success_count / data.requests_sent * 100) : 0;
            elements.successRate.textContent = successRate.toFixed(1) + '%';
            
            const lastRequestMs = (Date.now() - (data.last_request / 1e6));
            elements.lastRequest.textContent = (lastRequestMs / 1000).toFixed(1) + 's ago';
            
            // Update chart
            rpsHistory.push(data.rps);
            if (rpsHistory.length > maxDataPoints) {
                rpsHistory.shift();
            }
            
            chart.data.datasets[0].data = rpsHistory;
            chart.data.labels = rpsHistory.map((_, i) => `${i}s`);
            chart.update();
            
        } catch (error) {
            console.error('Error fetching stats:', error);
        }
    }
    
    // Control buttons
    elements.intensityUp.addEventListener('click', () => {
        fetch('/control?action=intensity_up');
    });
    
    elements.intensityDown.addEventListener('click', () => {
        fetch('/control?action=intensity_down');
    });
    
    let isAttacking = true;
    elements.toggleAttack.addEventListener('click', () => {
        isAttacking = !isAttacking;
        elements.toggleAttack.textContent = isAttacking ? 'Stop Attack' : 'Start Attack';
        fetch(`/control?action=${isAttacking ? 'start' : 'stop'}`);
    });
    
    // Initial fetch
    fetchStats();
});
