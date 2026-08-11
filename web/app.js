document.addEventListener('DOMContentLoaded', () => {
    // DOM Elements
    const redisStockValue = document.getElementById('redisStockValue');
    const avgLatencyValue = document.getElementById('avgLatencyValue');
    const ordersProcessedCount = document.getElementById('ordersProcessedCount');
    const overSellingStatus = document.getElementById('overSellingStatus');
    const currentStockDisplay = document.getElementById('currentStockDisplay');
    const initialStockDisplay = document.getElementById('initialStockDisplay');
    const stockProgressBar = document.getElementById('stockProgressBar');

    const prewarmInput = document.getElementById('prewarmInput');
    const btnPrewarm = document.getElementById('btnPrewarm');
    const btnBuySingle = document.getElementById('btnBuySingle');
    const btnSimulate10 = document.getElementById('btnSimulate10');
    const btnSimulate50 = document.getElementById('btnSimulate50');
    const btnSimulate100 = document.getElementById('btnSimulate100');
    const btnClearLogs = document.getElementById('btnClearLogs');
    const consoleLogs = document.getElementById('consoleLogs');

    // App State
    let activeProductID = '66b2671e2fae1a0012e8471a'; // Default mock ID
    let initialStock = 50;
    let currentStock = 50;
    let totalProcessedOrders = 0;
    let totalLatencies = [];

    // Helper: Log message to UI console
    function logToConsole(message, type = 'system') {
        const entry = document.createElement('div');
        const now = new Date().toLocaleTimeString();
        entry.className = `log-entry log-${type}`;
        const timeElement = document.createElement('span');
        timeElement.className = 'log-time';
        timeElement.textContent = `[${now}] `;
        entry.appendChild(timeElement);
        entry.appendChild(document.createTextNode(String(message)));
        consoleLogs.appendChild(entry);
        consoleLogs.scrollTop = consoleLogs.scrollHeight;
    }

    // Helper: Update UI Stock & Meter
    function updateStockUI(stock) {
        currentStock = stock;
        currentStockDisplay.textContent = stock;
        redisStockValue.textContent = stock;

        const percentage = initialStock > 0
            ? Math.max(0, Math.min(100, (stock / initialStock) * 100))
            : 0;
        stockProgressBar.style.width = `${percentage}%`;

        if (percentage <= 20) {
            stockProgressBar.style.background = 'linear-gradient(90deg, #ef4444, #f59e0b)';
        } else {
            stockProgressBar.style.background = 'linear-gradient(90deg, var(--accent-cyan), var(--accent-emerald))';
        }
    }

    // Helper: Update Latency UI
    function addLatency(ms) {
        totalLatencies.push(ms);
        if (totalLatencies.length > 50) totalLatencies.shift();
        const avg = Math.round(totalLatencies.reduce((a, b) => a + b, 0) / totalLatencies.length);
        avgLatencyValue.textContent = `${avg} ms`;
    }

    // API: Fetch Active Products
    async function loadProductData() {
        try {
            const res = await fetch('/api/v1/products');
            if (res.ok) {
                const data = await res.json();
                if (data.data && data.data.length > 0) {
                    const prod = data.data.find(product => product.is_flash_sale) || data.data[0];
                    activeProductID = prod.id;
                    document.getElementById('productTitle').textContent = prod.title;
                    document.getElementById('productDesc').textContent = prod.description || 'Flash sale product';
                    logToConsole(`Loaded active Flash Sale product: ${prod.title} (ID: ${activeProductID})`, 'system');
                } else {
                    await createDemoProduct();
                }
            } else {
                await createDemoProduct();
            }
        } catch (err) {
            logToConsole(`Could not connect to API server (${err.message}). Pre-warming offline demo state.`, 'error');
            // The demo starts with a placeholder product ID. If MongoDB is
            // reachable but the initial product lookup fails transiently, try
            // creating the demo product before polling Redis; otherwise the
            // Pre-warm button would send a request for a non-existent product.
            await createDemoProduct();
        }
        await fetchLiveStock();
    }

    // API: Create Demo Product if database is fresh
    async function createDemoProduct() {
        try {
            const res = await fetch('/api/v1/products', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer admin:dashboard'
                },
                body: JSON.stringify({
                    title: 'iPhone 15 Pro Max 1TB (Titanium)',
                    description: 'Ultra-high demand flagship smartphone Flash Sale event with atomic inventory control.',
                    price: 66900,
                    stock: 50,
                    is_flash_sale: true,
                    flash_price: 9900
                })
            });
            if (res.ok) {
                const data = await res.json();
                if (data.data && data.data.id) {
                    activeProductID = data.data.id;
                    logToConsole(`Created initial Flash Sale product (ID: ${activeProductID})`, 'system');
                }
            }
        } catch (e) {
            console.error(e);
        }
    }

    // API: Fetch Live Stock from Redis
    async function fetchLiveStock() {
        if (!activeProductID) return;
        try {
            const res = await fetch(`/api/v1/products/${activeProductID}/stock`);
            if (res.ok) {
                const data = await res.json();
                if (data.data && typeof data.data.stock === 'number') {
                    updateStockUI(data.data.stock);
                }
            }
        } catch (err) {
            // Silence polling error
        }
    }

    // API: Pre-warm Stock into Redis
    async function prewarmStock(retried = false) {
        const count = Number.parseInt(prewarmInput.value, 10);
        if (!Number.isInteger(count) || count < 0) {
            logToConsole('Stock must be a non-negative whole number.', 'error');
            return;
        }
        if (!retried && currentStock === 0) {
            logToConsole('Current demo product is exhausted; creating a fresh demo product...', 'info');
            await createDemoProduct();
            return prewarmStock(true);
        }
        initialStock = count;
        initialStockDisplay.textContent = count;

        logToConsole(`Sending Pre-warm request to Redis... (Stock: ${count})`, 'info');
        try {
            const startTime = performance.now();
            const res = await fetch('/api/v1/products/prewarm', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer admin:dashboard'
                },
                body: JSON.stringify({
                    product_id: activeProductID,
                    stock: count
                })
            });
            const duration = Math.round(performance.now() - startTime);

            if (res.ok) {
                updateStockUI(count);
                logToConsole(`✅ Pre-warmed Redis stock to ${count} items in ${duration}ms`, 'success');
            } else {
                logToConsole(`❌ Failed to pre-warm stock: ${res.statusText}`, 'error');
            }
        } catch (err) {
            logToConsole(`Error pre-warming stock: ${err.message}`, 'error');
        }
    }

    // Real-Time SSE Order Status Subscription
    async function subscribeOrderStatus(orderID, userID) {
        if (!orderID) return;

        logToConsole(`[Real-Time SSE] Subscribing to Order: ${orderID}...`, 'info');
        try {
            const response = await fetch(`/api/v1/orders/${orderID}/stream`, {
                headers: {
                    'Accept': 'text/event-stream',
                    'Authorization': `Bearer ${userID}`
                }
            });
            if (!response.ok || !response.body) {
                logToConsole(`Order stream failed with HTTP ${response.status}.`, 'error');
                return;
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let buffer = '';

            while (true) {
                const { value, done } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, '\n');
                const events = buffer.split('\n\n');
                buffer = events.pop() || '';

                for (const event of events) {
                    const dataLine = event.split('\n').find(line => line.startsWith('data:'));
                    if (!dataLine) continue;

                    const status = dataLine.slice(5).trim();
                    if (status === 'COMPLETED' || status === 'FAILED') {
                        const type = status === 'COMPLETED' ? 'success' : 'error';
                        logToConsole(`Order ${orderID} status -> ${status}`, type);
                        await reader.cancel();
                        await fetchLiveStock();
                        return;
                    }
                }
            }
        } catch (err) {
            logToConsole(`Order stream disconnected: ${err.message}`, 'error');
        }
    }

    // API: Create Flash Sale Order (Single)
    async function buyFlashSaleOrder(userID = `user-${Math.floor(Math.random() * 1000)}`) {
        const startTime = performance.now();
        try {
            const res = await fetch('/api/v1/orders/flash-sale', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${userID}`
                },
                body: JSON.stringify({
                    product_id: activeProductID,
                    quantity: 1
                })
            });

            const duration = Math.round(performance.now() - startTime);
            addLatency(duration);
            const data = await res.json();

            if (res.status === 202) {
                totalProcessedOrders++;
                ordersProcessedCount.textContent = totalProcessedOrders;
                const orderID = data.data?.order_id || 'N/A';
                logToConsole(`🚀 HTTP 202 Accepted | Latency: ${duration}ms | OrderID: ${orderID} (Queued via SQS)`, 'success');

                if (orderID !== 'N/A') {
                    void subscribeOrderStatus(orderID, userID);
                }

                await fetchLiveStock();
                return { success: true, latency: duration, orderID };
            } else {
                logToConsole(`⛔ HTTP ${res.status} Failed | Latency: ${duration}ms | ${data.message || 'Out of Stock'}`, 'error');
                await fetchLiveStock();
                return { success: false, latency: duration };
            }
        } catch (err) {
            const duration = Math.round(performance.now() - startTime);
            logToConsole(`❌ Network/Server Error: ${err.message}`, 'error');
            return { success: false, latency: duration };
        }
    }

    // Simulate Concurrency Batch
    async function simulateConcurrency(count) {
        logToConsole(`🔥 Initiating High-Concurrency Rush: Firing ${count} simultaneous Requests...`, 'info');
        const promises = [];
        const runId = Date.now();
        for (let i = 0; i < count; i++) {
            promises.push(buyFlashSaleOrder(`user-sim-${runId}-${i + 1}`));
        }

        const results = await Promise.all(promises);
        const succeeded = results.filter(r => r.success).length;
        const failed = results.filter(r => !r.success).length;

        logToConsole(`📊 Concurrency Test Summary: ${succeeded} Succeeded (202 Accepted), ${failed} Rejected (Out of Stock / Rate Limit)`, succeeded > 0 ? 'success' : 'error');
        await fetchLiveStock();
    }

    // Event Listeners
    btnPrewarm.addEventListener('click', prewarmStock);
    btnBuySingle.addEventListener('click', () => buyFlashSaleOrder());
    btnSimulate10.addEventListener('click', () => simulateConcurrency(10));
    btnSimulate50.addEventListener('click', () => simulateConcurrency(50));
    btnSimulate100.addEventListener('click', () => simulateConcurrency(100));
    btnClearLogs.addEventListener('click', () => {
        consoleLogs.replaceChildren();
        logToConsole('Console cleared.', 'system');
    });

    // Periodic Stock Polling
    setInterval(fetchLiveStock, 2500);

    // Initial Load
    loadProductData();
});
