document.addEventListener('DOMContentLoaded', () => {
    
    const fetchStats = async () => {
        try {
            const res = await fetch('/api/stats');
            const data = await res.json();
            
            document.getElementById('val-conns').textContent = data.connections;
            document.getElementById('val-keys').textContent = data.total_keys;
            document.getElementById('val-cmds').textContent = data.total_commands;
            
            let hitRate = 0;
            if (data.hits + data.misses > 0) {
                hitRate = (data.hits / (data.hits + data.misses)) * 100;
            }
            document.getElementById('val-hits').textContent = hitRate.toFixed(1) + '%';
        } catch (e) {
            console.error('Failed to fetch stats:', e);
        }
    };

    const fetchLogs = async () => {
        try {
            const res = await fetch('/api/logs');
            const data = await res.json();
            
            const terminal = document.getElementById('terminal-logs');
            terminal.innerHTML = '';
            
            data.logs.forEach(log => {
                const el = document.createElement('div');
                el.className = 'log-line';
                // Very basic parsing to highlight levels
                if (log.includes('[INFO]')) el.classList.add('level-info');
                else if (log.includes('[ERROR]')) el.classList.add('level-error');
                else if (log.includes('[DEBUG]')) el.classList.add('level-debug');
                
                el.textContent = log;
                terminal.appendChild(el);
            });
            
            terminal.scrollTop = terminal.scrollHeight;
        } catch (e) {
            console.error('Failed to fetch logs:', e);
        }
    };

    const fetchKeys = async () => {
        try {
            const res = await fetch('/api/keys');
            const data = await res.json();
            
            const tbody = document.querySelector('#keys-table tbody');
            tbody.innerHTML = '';
            
            data.keys.forEach(key => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${key}</td>
                    <td>
                        <button class="action-btn" onclick="alert('View/Edit not implemented yet!')">View</button>
                    </td>
                `;
                tbody.appendChild(tr);
            });
        } catch (e) {
            console.error('Failed to fetch keys:', e);
        }
    };

    document.getElementById('refresh-keys').addEventListener('click', fetchKeys);

    // Initial fetch
    fetchStats();
    fetchLogs();
    fetchKeys();

    // Polls
    setInterval(fetchStats, 1000);
    setInterval(fetchLogs, 2000);
});
