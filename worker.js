self.onmessage = (event) => {
    const { delay, maxRequests, targetUrl, apiKey } = event.data;
    let requestCount = 0;
    let successfulCount = 0;
    let failedCount = 0;
    let error429Count = 0;
    let error503Count = 0;
    const startTime = performance.now();

    function sendRequest() {
        if (maxRequests > 0 && requestCount >= maxRequests) {
            self.postMessage({
                requestCount,
                successfulCount,
                failedCount,
                error429Count,
                error503Count
            });
            return;
        }

        if (!targetUrl) {
            self.postMessage({
                requestCount,
                successfulCount,
                failedCount,
                error429Count,
                error503Count
            });
            return;
        }

        requestCount++;
        const url = `https://app.scrapingbee.com/api/v1/?api_key=${apiKey}&url=${encodeURIComponent(targetUrl)}&render_js=true`;

        fetch(url, {
            method: 'GET',
            cache: 'no-store',
            mode: 'no-cors'
        })
        .then(response => {
            if (response.status === 429) {
                error429Count++;
                failedCount++;
            } else if (response.status === 503) {
                error503Count++;
                failedCount++;
            } else if (response.ok || response.status === 0) {
                successfulCount++;
            } else {
                failedCount++;
            }
            self.postMessage({
                requestCount,
                successfulCount,
                failedCount,
                error429Count,
                error503Count
            });
        })
        .catch(error => {
            failedCount++;
            self.postMessage({
                requestCount,
                successfulCount,
                failedCount,
                error429Count,
                error503Count
            });
        });
    }

    // Use setInterval for high-frequency requests
    setInterval(sendRequest, delay);
};
