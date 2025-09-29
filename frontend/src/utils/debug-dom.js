// Debug DOM Changes - Detecta re-renderizações e mudanças no DOM
console.log('🔍 DOM Debug Tool carregado');

// 1. Monitor de Re-renderizações React
window.debugReactRenders = (() => {
  const originalRender = React.Component.prototype.render;
  const renderCounts = new Map();
  
  // Hook para componentes funcionais
  const originalUseState = React.useState;
  const originalUseEffect = React.useEffect;
  
  let renderCounter = 0;
  
  // Intercepta useState
  React.useState = function(initialState) {
    const result = originalUseState(initialState);
    console.log('🔄 useState chamado:', { initialState, stackTrace: new Error().stack.split('\n')[2] });
    return result;
  };
  
  // Intercepta useEffect
  React.useEffect = function(effect, deps) {
    console.log('⚡ useEffect executado:', { 
      deps, 
      hasCleanup: typeof effect() === 'function',
      stackTrace: new Error().stack.split('\n')[2]
    });
    return originalUseEffect(effect, deps);
  };
  
  return {
    start() {
      console.log('🚀 Monitoramento de re-renders iniciado');
    },
    stop() {
      React.useState = originalUseState;
      React.useEffect = originalUseEffect;
      console.log('🛑 Monitoramento de re-renders parado');
    },
    getStats() {
      return Array.from(renderCounts.entries()).map(([component, count]) => ({
        component,
        renders: count
      })).sort((a, b) => b.renders - a.renders);
    }
  };
})();

// 2. Monitor de Mudanças no DOM
window.debugDOMChanges = (() => {
  let observer;
  let changeCount = 0;
  const recentChanges = [];
  
  return {
    start() {
      if (observer) {
        observer.disconnect();
      }
      
      observer = new MutationObserver((mutations) => {
        mutations.forEach((mutation) => {
          changeCount++;
          const change = {
            type: mutation.type,
            target: mutation.target,
            tagName: mutation.target.tagName,
            className: mutation.target.className,
            timestamp: Date.now(),
            addedNodes: mutation.addedNodes.length,
            removedNodes: mutation.removedNodes.length,
            attributeName: mutation.attributeName,
            oldValue: mutation.oldValue
          };
          
          recentChanges.push(change);
          if (recentChanges.length > 100) {
            recentChanges.shift();
          }
          
          // Log mudanças suspeitas (muitas mudanças rapidamente)
          const now = Date.now();
          const recentCount = recentChanges.filter(c => now - c.timestamp < 1000).length;
          
          if (recentCount > 10) {
            console.warn('⚠️ DOM mudando muito rapidamente!', {
              recentCount,
              change,
              element: mutation.target
            });
          }
          
          // Log mudanças de estilo que podem causar "piscadas"
          if (mutation.type === 'attributes' && 
              ['class', 'style', 'data-state'].includes(mutation.attributeName)) {
            console.log('🎨 Mudança de estilo/classe:', {
              element: mutation.target,
              attribute: mutation.attributeName,
              oldValue: mutation.oldValue,
              newValue: mutation.target.getAttribute(mutation.attributeName)
            });
          }
        });
      });
      
      observer.observe(document.body, {
        childList: true,
        subtree: true,
        attributes: true,
        attributeOldValue: true,
        characterData: true
      });
      
      console.log('🔍 Monitoramento DOM iniciado');
    },
    
    stop() {
      if (observer) {
        observer.disconnect();
        observer = null;
      }
      console.log('🛑 Monitoramento DOM parado');
    },
    
    getStats() {
      return {
        totalChanges: changeCount,
        recentChanges: recentChanges.slice(-20),
        changeRate: recentChanges.filter(c => Date.now() - c.timestamp < 1000).length
      };
    },
    
    reset() {
      changeCount = 0;
      recentChanges.length = 0;
      console.log('🔄 Stats resetadas');
    }
  };
})();

// 3. Monitor de Performance e Re-paint
window.debugPaintEvents = (() => {
  let paintCount = 0;
  let animationFrame;
  
  return {
    start() {
      const trackPaints = () => {
        paintCount++;
        if (paintCount % 60 === 0) { // Log a cada segundo (60fps)
          console.log('🎨 Paint events por segundo:', paintCount / (Date.now() / 1000));
        }
        animationFrame = requestAnimationFrame(trackPaints);
      };
      
      trackPaints();
      console.log('🎨 Monitoramento de paint iniciado');
    },
    
    stop() {
      if (animationFrame) {
        cancelAnimationFrame(animationFrame);
      }
      console.log('🛑 Monitoramento de paint parado');
    },
    
    getStats() {
      return { paintCount };
    }
  };
})();

// 4. Monitor de Queries React Query
window.debugReactQuery = (() => {
  return {
    getActiveQueries() {
      // Tenta acessar o QueryClient se disponível
      try {
        const queryClient = window.__REACT_QUERY_CLIENT__;
        if (queryClient) {
          return queryClient.getQueryCache().getAll().map(query => ({
            queryKey: query.queryKey,
            state: query.state.status,
            dataUpdatedAt: query.state.dataUpdatedAt,
            errorUpdatedAt: query.state.errorUpdatedAt,
            fetchStatus: query.state.fetchStatus,
            isStale: query.isStale()
          }));
        }
      } catch (e) {
        console.log('React Query não encontrado');
      }
      return [];
    },
    
    logFrequentRefetches() {
      const queries = this.getActiveQueries();
      const now = Date.now();
      
      queries.forEach(query => {
        if (query.dataUpdatedAt && (now - query.dataUpdatedAt) < 5000) {
          console.warn('🔄 Query atualizando frequentemente:', query);
        }
      });
    }
  };
})();

// 5. Helper para identificar componentes com muitos re-renders
window.identifyProblematicComponents = () => {
  console.group('🔍 Análise de Componentes Problemáticos');
  
  // Procura por elementos que mudam frequentemente
  const elementsWithClasses = document.querySelectorAll('[class*="animate"], [class*="transition"], [data-state]');
  
  elementsWithClasses.forEach(el => {
    console.log('🎭 Elemento com animações/transições:', {
      element: el,
      classes: el.className,
      dataState: el.getAttribute('data-state'),
      parent: el.parentElement?.tagName
    });
  });
  
  // Procura por timers/intervals ativos
  const originalSetInterval = window.setInterval;
  const originalSetTimeout = window.setTimeout;
  const activeTimers = [];
  
  window.setInterval = function(...args) {
    const id = originalSetInterval.apply(this, args);
    activeTimers.push({ type: 'interval', id, callback: args[0].toString() });
    return id;
  };
  
  window.setTimeout = function(...args) {
    const id = originalSetTimeout.apply(this, args);
    activeTimers.push({ type: 'timeout', id, callback: args[0].toString() });
    return id;
  };
  
  console.log('⏰ Timers ativos:', activeTimers);
  console.groupEnd();
};

// 6. Iniciar tudo de uma vez
window.startAllDebug = () => {
  window.debugReactRenders.start();
  window.debugDOMChanges.start();
  window.debugPaintEvents.start();
  console.log('🚀 Todos os debugs iniciados! Use as funções individuais para controlar.');
};

window.stopAllDebug = () => {
  window.debugReactRenders.stop();
  window.debugDOMChanges.stop();
  window.debugPaintEvents.stop();
  console.log('🛑 Todos os debugs parados!');
};

window.getDebugReport = () => {
  console.group('📊 Relatório de Debug');
  console.log('DOM Changes:', window.debugDOMChanges.getStats());
  console.log('Paint Events:', window.debugPaintEvents.getStats());
  console.log('React Query:', window.debugReactQuery.getActiveQueries());
  window.identifyProblematicComponents();
  console.groupEnd();
};

// Comandos disponíveis
console.log(`
🔧 Comandos disponíveis:
- startAllDebug() - Inicia todos os monitores
- stopAllDebug() - Para todos os monitores
- getDebugReport() - Mostra relatório completo
- debugDOMChanges.start/stop() - Monitor DOM
- debugReactRenders.start/stop() - Monitor React
- debugPaintEvents.start/stop() - Monitor Paint
- identifyProblematicComponents() - Identifica problemas
`);