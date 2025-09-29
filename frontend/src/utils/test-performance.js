// Test Performance - Execute no console do browser
console.log('🧪 Teste de Performance iniciado');

// 1. Executa análise completa
async function runCompleteAnalysis() {
  console.group('🔍 Análise Completa de Performance');
  
  // Carrega os scripts de debug
  try {
    await loadScript('/src/utils/debug-dom.js');
    await loadScript('/src/utils/performance-fix.js');
  } catch (e) {
    console.warn('Scripts de debug não carregados, continuando...');
  }
  
  // Inicia monitoramento
  if (window.startAllDebug) {
    window.startAllDebug();
  }
  
  // Aguarda 10 segundos coletando dados
  console.log('⏳ Coletando dados por 10 segundos...');
  
  await new Promise(resolve => setTimeout(resolve, 10000));
  
  // Coleta relatório
  const report = {
    timestamp: new Date().toISOString(),
    performance: window.getPerformanceReport ? window.getPerformanceReport() : null,
    domStats: window.debugDOMChanges ? window.debugDOMChanges.getStats() : null,
    paintStats: window.debugPaintEvents ? window.debugPaintEvents.getStats() : null,
    
    // Análise manual
    totalElements: document.querySelectorAll('*').length,
    activeAnimations: document.querySelectorAll('.animate-pulse, .animate-spin, [data-state]').length,
    reactQueries: window.debugReactQuery ? window.debugReactQuery.getActiveQueries().length : 0,
    memoryUsage: performance.memory ? {
      used: Math.round(performance.memory.usedJSHeapSize / 1024 / 1024) + 'MB',
      total: Math.round(performance.memory.totalJSHeapSize / 1024 / 1024) + 'MB'
    } : 'N/A'
  };
  
  console.log('📊 Relatório Final:', report);
  
  // Identifica possíveis culpados
  identifyPerformanceIssues(report);
  
  console.groupEnd();
  return report;
}

// 2. Identifica problemas específicos
function identifyPerformanceIssues(report) {
  console.group('🚨 Problemas Identificados');
  
  const issues = [];
  
  if (report.domStats && report.domStats.changeRate > 5) {
    issues.push(`🔥 DOM mudando muito rapidamente: ${report.domStats.changeRate} mudanças/segundo`);
  }
  
  if (report.totalElements > 5000) {
    issues.push(`📏 Muitos elementos DOM: ${report.totalElements} elementos`);
  }
  
  if (report.activeAnimations > 20) {
    issues.push(`🎭 Muitas animações ativas: ${report.activeAnimations} elementos animados`);
  }
  
  // Verifica elementos específicos problemáticos
  const problematicElements = [
    { selector: '[data-state]', name: 'Elementos com data-state' },
    { selector: '.animate-pulse', name: 'Elementos pulsando' },
    { selector: '.loading', name: 'Elementos loading' },
    { selector: '[class*="transition"]', name: 'Elementos com transição' }
  ];
  
  problematicElements.forEach(({ selector, name }) => {
    const count = document.querySelectorAll(selector).length;
    if (count > 10) {
      issues.push(`🎨 Muitos ${name}: ${count} elementos`);
    }
  });
  
  // Verifica queries React Query frequentes
  if (window.debugReactQuery) {
    const queries = window.debugReactQuery.getActiveQueries();
    const frequentQueries = queries.filter(q => {
      const lastUpdate = Date.now() - (q.dataUpdatedAt || 0);
      return lastUpdate < 30000; // Atualizou nos últimos 30s
    });
    
    if (frequentQueries.length > 3) {
      issues.push(`🔄 Muitas queries atualizando: ${frequentQueries.length} queries recentes`);
      frequentQueries.forEach(q => {
        console.log(`  - ${q.queryKey.join('/')}: ${q.state}`);
      });
    }
  }
  
  if (issues.length === 0) {
    console.log('✅ Nenhum problema óbvio identificado');
  } else {
    issues.forEach(issue => console.warn(issue));
  }
  
  console.groupEnd();
  return issues;
}

// 3. Load script helper
function loadScript(src) {
  return new Promise((resolve, reject) => {
    const script = document.createElement('script');
    script.src = src;
    script.onload = resolve;
    script.onerror = reject;
    document.head.appendChild(script);
  });
}

// 4. Test específico para "piscadas"
function testFlickeringElements() {
  console.group('👁️ Teste de Elementos Piscando');
  
  const observers = [];
  const flickerCounts = new Map();
  
  // Monitora mudanças visuais em elementos suspeitos
  const suspiciousElements = document.querySelectorAll(`
    [data-state],
    .badge,
    [class*="badge"],
    .animate-pulse,
    .loading,
    [role="button"],
    .status
  `);
  
  suspiciousElements.forEach((element, index) => {
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.type === 'attributes') {
          const key = `${element.tagName}-${index}`;
          flickerCounts.set(key, (flickerCounts.get(key) || 0) + 1);
          
          if (flickerCounts.get(key) > 5) {
            console.warn('🔥 Elemento piscando detectado:', {
              element,
              mutations: flickerCounts.get(key),
              attribute: mutation.attributeName,
              oldValue: mutation.oldValue,
              newValue: element.getAttribute(mutation.attributeName)
            });
          }
        }
      });
    });
    
    observer.observe(element, {
      attributes: true,
      attributeOldValue: true,
      attributeFilter: ['class', 'data-state', 'style', 'aria-expanded']
    });
    
    observers.push(observer);
  });
  
  console.log(`👀 Monitorando ${suspiciousElements.length} elementos por 15 segundos...`);
  
  // Para após 15 segundos
  setTimeout(() => {
    observers.forEach(obs => obs.disconnect());
    
    console.log('📊 Resultado do teste de piscadas:');
    Array.from(flickerCounts.entries()).forEach(([element, count]) => {
      if (count > 3) {
        console.warn(`🚨 ${element}: ${count} mudanças`);
      }
    });
    
    if (flickerCounts.size === 0) {
      console.log('✅ Nenhuma piscada detectada');
    }
    
    console.groupEnd();
  }, 15000);
}

// 5. Comandos disponíveis
window.testPerformance = {
  runCompleteAnalysis,
  testFlickeringElements,
  
  // Quick fixes
  reduceAnimations: () => {
    document.querySelectorAll('.animate-pulse').forEach(el => {
      el.style.animationDuration = '3s';
    });
    console.log('⚡ Animações reduzidas');
  },
  
  stopAllPolling: () => {
    // Para todos os intervals
    for (let i = 1; i < 9999; i++) {
      clearInterval(i);
      clearTimeout(i);
    }
    console.log('🛑 Todos os intervals parados');
  },
  
  enableGPUAcceleration: () => {
    const style = document.createElement('style');
    style.textContent = `
      .animate-pulse, [data-state], .badge {
        will-change: transform, opacity;
        transform: translateZ(0);
      }
    `;
    document.head.appendChild(style);
    console.log('🚀 GPU acceleration habilitada');
  }
};

// Auto-start
console.log(`
🧪 Comandos de Teste Disponíveis:
- testPerformance.runCompleteAnalysis() - Análise completa (10s)
- testPerformance.testFlickeringElements() - Teste de piscadas (15s)
- testPerformance.reduceAnimations() - Reduz animações
- testPerformance.stopAllPolling() - Para todos os intervals
- testPerformance.enableGPUAcceleration() - Habilita GPU acceleration

💡 Execute testPerformance.runCompleteAnalysis() para começar!
`);

// Auto-run basic analysis
setTimeout(() => {
  console.log('🚀 Executando análise automática...');
  testPerformance.runCompleteAnalysis();
}, 2000);