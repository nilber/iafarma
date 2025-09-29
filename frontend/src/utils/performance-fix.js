// Performance Fix - Reduz re-renderizações e "piscadas"
console.log('⚡ Performance Fix carregado');

// 1. Desabilita temporariamente alguns refetch intervals para teste
window.disableAutoRefresh = () => {
  // Injeta CSS para suavizar transições e reduzir "piscadas"
  const style = document.createElement('style');
  style.textContent = `
    /* Suaviza todas as mudanças de estado */
    * {
      transition: opacity 0.15s ease-in-out, 
                  background-color 0.15s ease-in-out,
                  border-color 0.15s ease-in-out,
                  color 0.15s ease-in-out !important;
    }
    
    /* Reduz animações de elementos com data-state */
    [data-state] {
      transition: all 0.1s ease-in-out !important;
    }
    
    /* Suaviza mudanças em badges e status */
    .badge, [class*="badge"] {
      transition: all 0.15s ease-in-out !important;
    }
    
    /* Reduz flicker em elementos loading */
    .animate-pulse {
      animation-duration: 2s !important;
    }
    
    /* Suaviza mudanças em tabelas */
    table tr {
      transition: background-color 0.15s ease-in-out !important;
    }
    
    /* Reduz "jump" em elementos que aparecem/desaparecem */
    .space-y-4 > *, .space-y-6 > * {
      transition: opacity 0.2s ease-in-out, transform 0.2s ease-in-out !important;
    }
  `;
  document.head.appendChild(style);
  
  console.log('🎨 CSS de suavização aplicado');
  
  // Intercepta e controla setInterval para reduzir polling
  const originalSetInterval = window.setInterval;
  const controlledIntervals = new Map();
  
  window.setInterval = function(callback, delay, ...args) {
    // Aumenta delays muito curtos para reduzir "piscadas"
    if (delay < 5000) {
      console.warn(`⚠️ Interval muito curto (${delay}ms) aumentado para 5s`, callback.toString().slice(0, 100));
      delay = 5000;
    }
    
    const id = originalSetInterval.call(this, callback, delay, ...args);
    controlledIntervals.set(id, { callback: callback.toString().slice(0, 100), delay });
    return id;
  };
  
  console.log('⏰ Controle de intervals ativado');
  
  return {
    getActiveIntervals: () => Array.from(controlledIntervals.entries()),
    restoreInterval: () => { window.setInterval = originalSetInterval; }
  };
};

// 2. Função para pausar re-renders temporariamente
window.pauseReactUpdates = (duration = 5000) => {
  const originalSetState = React.Component.prototype.setState;
  const originalDispatch = React.useReducer;
  
  let paused = true;
  const queuedUpdates = [];
  
  // Intercept setState
  React.Component.prototype.setState = function(updater, callback) {
    if (paused) {
      queuedUpdates.push(() => originalSetState.call(this, updater, callback));
      return;
    }
    return originalSetState.call(this, updater, callback);
  };
  
  setTimeout(() => {
    paused = false;
    React.Component.prototype.setState = originalSetState;
    
    // Execute queued updates
    queuedUpdates.forEach(update => update());
    console.log(`🔄 ${queuedUpdates.length} updates executados após pausa`);
  }, duration);
  
  console.log(`⏸️ React updates pausados por ${duration}ms`);
};

// 3. Detecta e reporta componentes que re-renderizam muito
window.detectFrequentRerenders = () => {
  const componentRenders = new Map();
  const RENDER_THRESHOLD = 10; // renders per second
  
  // Intercepta React.createElement para contar renders
  const originalCreateElement = React.createElement;
  
  React.createElement = function(type, props, ...children) {
    if (typeof type === 'function' || typeof type === 'string') {
      const name = type.displayName || type.name || type;
      const now = Date.now();
      
      if (!componentRenders.has(name)) {
        componentRenders.set(name, []);
      }
      
      const renders = componentRenders.get(name);
      renders.push(now);
      
      // Keep only renders from last second
      const recentRenders = renders.filter(time => now - time < 1000);
      componentRenders.set(name, recentRenders);
      
      // Warn about frequent renders
      if (recentRenders.length > RENDER_THRESHOLD) {
        console.warn(`🔥 Componente renderizando muito: ${name} (${recentRenders.length} renders/s)`);
      }
    }
    
    return originalCreateElement.apply(this, arguments);
  };
  
  return {
    getStats: () => Array.from(componentRenders.entries())
      .map(([name, renders]) => ({ name, count: renders.length }))
      .sort((a, b) => b.count - a.count),
    
    reset: () => {
      React.createElement = originalCreateElement;
      componentRenders.clear();
    }
  };
};

// 4. Otimiza queries React Query
window.optimizeReactQuery = () => {
  try {
    // Tenta encontrar o query client
    const queryClient = window.__REACT_QUERY_CLIENT__ || 
                       window.reactQueryClient ||
                       document.querySelector('[data-react-query-client]')?.__reactQueryClient__;
    
    if (queryClient) {
      // Aumenta staleTime para reduzir refetches
      const queries = queryClient.getQueryCache().getAll();
      
      queries.forEach(query => {
        const currentOptions = query.options;
        
        // Aumenta staleTime se for muito baixo
        if (currentOptions.staleTime < 60000) { // menos de 1 minuto
          query.options.staleTime = 60000;
          console.log(`⏰ StaleTime aumentado para query: ${query.queryKey}`);
        }
        
        // Remove refetchInterval muito frequente
        if (currentOptions.refetchInterval && currentOptions.refetchInterval < 30000) {
          query.options.refetchInterval = 30000;
          console.log(`🔄 RefetchInterval aumentado para query: ${query.queryKey}`);
        }
      });
      
      console.log(`✅ ${queries.length} queries otimizadas`);
    }
  } catch (e) {
    console.log('❌ React Query não encontrado para otimização');
  }
};

// 5. Emergency stop - para tudo
window.emergencyStop = () => {
  // Para todos os intervals
  for (let i = 1; i < 99999; i++) {
    window.clearInterval(i);
    window.clearTimeout(i);
  }
  
  // Remove event listeners que podem causar updates
  const events = ['scroll', 'resize', 'mousemove', 'focus', 'blur'];
  events.forEach(event => {
    const listeners = document.querySelectorAll(`[on${event}]`);
    listeners.forEach(el => el.removeAttribute(`on${event}`));
  });
  
  console.log('🚨 EMERGENCY STOP: Intervals e eventos removidos!');
};

// 6. Relatório de performance
window.getPerformanceReport = () => {
  const report = {
    intervals: [],
    memoryUsage: performance.memory ? {
      used: Math.round(performance.memory.usedJSHeapSize / 1024 / 1024),
      total: Math.round(performance.memory.totalJSHeapSize / 1024 / 1024),
      limit: Math.round(performance.memory.jsHeapSizeLimit / 1024 / 1024)
    } : 'N/A',
    domNodes: document.querySelectorAll('*').length,
    eventListeners: getEventListenerCount(),
    renderCount: window.debugReactRenders?.getStats() || []
  };
  
  console.group('📊 Performance Report');
  console.table(report);
  console.groupEnd();
  
  return report;
};

function getEventListenerCount() {
  // Aproximação do número de event listeners
  const elements = document.querySelectorAll('*');
  let count = 0;
  
  elements.forEach(el => {
    const events = ['click', 'change', 'input', 'focus', 'blur', 'scroll'];
    events.forEach(event => {
      if (el[`on${event}`] !== null || el.getAttribute(`on${event}`)) count++;
    });
  });
  
  return count;
}

// Auto-start básico
console.log(`
🚀 Performance Fix Comandos:
- disableAutoRefresh() - Reduz polling e adiciona transições suaves
- optimizeReactQuery() - Otimiza queries React Query
- detectFrequentRerenders() - Detecta componentes problemáticos
- pauseReactUpdates(5000) - Pausa updates por X ms
- emergencyStop() - Para tudo (emergência)
- getPerformanceReport() - Relatório completo

💡 Execute disableAutoRefresh() primeiro para testar!
`);

// Auto-aplica otimizações básicas
setTimeout(() => {
  window.disableAutoRefresh();
  window.optimizeReactQuery();
}, 1000);