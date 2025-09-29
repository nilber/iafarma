/**
 * Utilitário para monitorar invalidações inteligentes do WebSocket
 * Só ativo em desenvolvimento para debug
 */

let invalidationCount = 0;
let lastInvalidation = 0;

export const monitorSmartInvalidations = (type: string) => {
  if (!import.meta.env.DEV) return;
  
  const now = Date.now();
  const timeSinceLastInvalidation = now - lastInvalidation;
  
  invalidationCount++;
  lastInvalidation = now;
  
  console.log(`🔄 Smart Invalidation #${invalidationCount}: ${type} (${timeSinceLastInvalidation}ms since last)`);
  
  // Alerta se invalidações estão muito frequentes
  if (timeSinceLastInvalidation < 1000 && invalidationCount > 1) {
    console.warn('⚠️ Invalidations happening too frequently!', {
      type,
      timeSinceLastInvalidation,
      count: invalidationCount
    });
  }
};