// Not needed for core functionality, but included if you want to offload UI tasks
// This would be used in public/script.js if you need heavy computation in the UI

self.addEventListener('message', (e) => {
  const { type, data } = e.data;
  
  if (type === 'process_stats') {
    // Example processing that could be done in a worker
    const processed = {
      rps: data.rps,
      smoothed: smoothData(data.history, 5),
      prediction: predictNextValue(data.history)
    };
    
    self.postMessage({
      type: 'processed_stats',
      data: processed
    });
  }
});

function smoothData(data, windowSize) {
  if (data.length === 0) return [];
  return data.map((_, i) => {
    const start = Math.max(0, i - windowSize);
    const end = i + 1;
    const slice = data.slice(start, end);
    return slice.reduce((a, b) => a + b, 0) / slice.length;
  });
}

function predictNextValue(data) {
  if (data.length < 2) return 0;
  const last = data[data.length - 1];
  const secondLast = data[data.length - 2];
  return last + (last - secondLast);
}
