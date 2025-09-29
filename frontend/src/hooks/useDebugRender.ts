import { useEffect, useRef } from 'react';

// Hook para detectar quais props mudaram e causaram re-render
export function useWhyDidYouUpdate(name: string, props: Record<string, any>) {
  const previousProps = useRef<Record<string, any>>();

  useEffect(() => {
    if (previousProps.current) {
      const allKeys = Object.keys({ ...previousProps.current, ...props });
      const changedProps: Record<string, { from: any; to: any }> = {};

      allKeys.forEach((key) => {
        if (previousProps.current![key] !== props[key]) {
          changedProps[key] = {
            from: previousProps.current![key],
            to: props[key],
          };
        }
      });

      if (Object.keys(changedProps).length) {
        console.group(`🔄 [${name}] Props que mudaram:`);
        console.table(changedProps);
        console.groupEnd();
      }
    }

    previousProps.current = props;
  });
}

// Hook para contar renders
export function useRenderCount(componentName: string) {
  const renderCount = useRef(0);
  renderCount.current += 1;

  useEffect(() => {
    console.log(`🎨 [${componentName}] Render #${renderCount.current}`);
  });

  return renderCount.current;
}

// Hook para detectar re-renders desnecessários
export function useRenderTracker(
  componentName: string, 
  deps: Record<string, any> = {},
  options: { showInConsole?: boolean; showVisual?: boolean } = { showInConsole: true, showVisual: false }
) {
  const renderCount = useRef(0);
  const previousDeps = useRef<Record<string, any>>();
  const lastRenderTime = useRef(Date.now());

  renderCount.current += 1;
  const currentTime = Date.now();
  const timeSinceLastRender = currentTime - lastRenderTime.current;
  lastRenderTime.current = currentTime;

  useEffect(() => {
    if (options.showInConsole) {
      const changedDeps: Record<string, { from: any; to: any }> = {};
      
      if (previousDeps.current) {
        Object.keys(deps).forEach((key) => {
          if (previousDeps.current![key] !== deps[key]) {
            changedDeps[key] = {
              from: previousDeps.current![key],
              to: deps[key],
            };
          }
        });
      }

      if (Object.keys(changedDeps).length > 0 || renderCount.current === 1) {
        console.group(`🎨 [${componentName}] Render #${renderCount.current} (+${timeSinceLastRender}ms)`);
        
        if (Object.keys(changedDeps).length > 0) {
          console.log('📊 Dependências que mudaram:');
          console.table(changedDeps);
        }
        
        if (renderCount.current === 1) {
          console.log('🆕 Primeiro render');
        }
        
        console.groupEnd();
      }
    }

    previousDeps.current = { ...deps };
  });

  return {
    renderCount: renderCount.current,
    timeSinceLastRender,
    hasFrequentRenders: timeSinceLastRender < 100, // Renders muito frequentes (< 100ms)
  };
}

// Hook para detectar renders em cascata
export function useCascadeDetector(componentName: string, threshold = 50) {
  const renderTimes = useRef<number[]>([]);
  
  useEffect(() => {
    const now = Date.now();
    renderTimes.current.push(now);
    
    // Manter apenas os últimos 10 renders
    if (renderTimes.current.length > 10) {
      renderTimes.current = renderTimes.current.slice(-10);
    }
    
    // Verificar se há muitos renders em pouco tempo
    const recentRenders = renderTimes.current.filter(time => now - time < threshold);
    if (recentRenders.length >= 3) {
      console.warn(`⚡ [${componentName}] CASCADE DETECTED: ${recentRenders.length} renders em ${threshold}ms`);
    }
  });
}