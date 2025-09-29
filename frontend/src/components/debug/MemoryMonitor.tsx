import { useEffect, useRef, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Activity, AlertTriangle, Eye, EyeOff, Trash2 } from 'lucide-react';

interface MemoryUsage {
  usedJSHeapSize: number;
  totalJSHeapSize: number;
  jsHeapSizeLimit: number;
  timestamp: number;
}

interface MemoryMonitorProps {
  enabled?: boolean;
  position?: 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right';
  interval?: number; // em ms
  threshold?: number; // em MB para alerta
}

export function MemoryMonitor({ 
  enabled = true, 
  position = 'top-left',
  interval = 5000, // 5 segundos
  threshold = 500 // 500MB
}: MemoryMonitorProps) {
  const [isVisible, setIsVisible] = useState(false);
  const [memoryHistory, setMemoryHistory] = useState<MemoryUsage[]>([]);
  const [currentMemory, setCurrentMemory] = useState<MemoryUsage | null>(null);
  const intervalRef = useRef<NodeJS.Timeout>();

  // Verificar se a API de memória está disponível
  const isMemoryAPIAvailable = 'memory' in performance;

  const getMemoryUsage = (): MemoryUsage | null => {
    if (!isMemoryAPIAvailable) return null;
    
    const memory = (performance as any).memory;
    return {
      usedJSHeapSize: memory.usedJSHeapSize,
      totalJSHeapSize: memory.totalJSHeapSize,
      jsHeapSizeLimit: memory.jsHeapSizeLimit,
      timestamp: Date.now(),
    };
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getMBFromBytes = (bytes: number): number => {
    return bytes / (1024 * 1024);
  };

  const triggerGarbageCollection = () => {
    // Force garbage collection se disponível (apenas em dev tools)
    if ((window as any).gc) {
      (window as any).gc();
      console.log('🗑️ Garbage Collection forçada');
    } else {
      console.warn('⚠️ Garbage Collection não disponível (abra Dev Tools → Settings → Advanced → Enable "Expose garbage collection")');
    }
  };

  useEffect(() => {
    if (!enabled || !isMemoryAPIAvailable) return;

    const updateMemory = () => {
      const usage = getMemoryUsage();
      if (usage) {
        setCurrentMemory(usage);
        setMemoryHistory(prev => [...prev.slice(-19), usage]); // Manter últimos 20 registros
      }
    };

    // Primeira medição imediata
    updateMemory();

    // Configurar intervalo
    // ❌ DESABILITADO: setInterval que estava causando re-renders periódicos
    // intervalRef.current = setInterval(updateMemory, interval);
    updateMemory(); // Executar apenas uma vez

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [enabled, interval, isMemoryAPIAvailable]);

  const positionClasses = {
    'top-left': 'top-4 left-4',
    'top-right': 'top-4 right-4',
    'bottom-left': 'bottom-4 left-4',
    'bottom-right': 'bottom-4 right-4',
  };

  if (!enabled || !isMemoryAPIAvailable) return null;

  const isHighMemory = currentMemory && getMBFromBytes(currentMemory.usedJSHeapSize) > threshold;
  const memoryPercentage = currentMemory 
    ? (currentMemory.usedJSHeapSize / currentMemory.totalJSHeapSize) * 100 
    : 0;

  return (
    <div className={`fixed ${positionClasses[position]} z-[9997] max-w-md`}>
      {/* Botão toggle */}
      <Button
        variant="outline"
        size="sm"
        onClick={() => setIsVisible(!isVisible)}
        className={`mb-2 ${isHighMemory ? 'bg-red-500 text-white border-red-500 hover:bg-red-600' : 'bg-green-500 text-white border-green-500 hover:bg-green-600'} shadow-lg`}
        title="Monitor de Memória"
      >
        <Activity className="w-4 h-4 mr-1" />
        {isVisible ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
        {currentMemory && (
          <Badge variant={isHighMemory ? "destructive" : "secondary"} className="ml-1 text-xs">
            {formatBytes(currentMemory.usedJSHeapSize)}
          </Badge>
        )}
      </Button>

      {/* Painel de monitoramento */}
      {isVisible && currentMemory && (
        <Card className="w-80 shadow-xl border-green-200 bg-white/95 backdrop-blur-sm">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm flex items-center gap-2">
                <Activity className="w-4 h-4 text-green-500" />
                Monitor de Memória
                {isHighMemory && (
                  <Badge variant="destructive" className="text-xs">
                    ⚠️ Alta
                  </Badge>
                )}
              </CardTitle>
              <Button
                variant="ghost"
                size="sm"
                onClick={triggerGarbageCollection}
                title="Forçar Garbage Collection"
              >
                <Trash2 className="w-4 h-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent className="p-3 space-y-3">
            {/* Uso atual */}
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Usado:</span>
                <span className={isHighMemory ? 'text-red-600 font-semibold' : 'text-green-600'}>
                  {formatBytes(currentMemory.usedJSHeapSize)}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span>Total:</span>
                <span>{formatBytes(currentMemory.totalJSHeapSize)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span>Limite:</span>
                <span>{formatBytes(currentMemory.jsHeapSizeLimit)}</span>
              </div>
              
              {/* Barra de progresso */}
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div 
                  className={`h-2 rounded-full transition-all duration-300 ${
                    memoryPercentage > 80 ? 'bg-red-500' :
                    memoryPercentage > 60 ? 'bg-yellow-500' : 'bg-green-500'
                  }`}
                  style={{ width: `${Math.min(memoryPercentage, 100)}%` }}
                />
              </div>
              <div className="text-xs text-center text-muted-foreground">
                {memoryPercentage.toFixed(1)}% usado
              </div>
            </div>

            {/* Histórico (últimos 5 registros) */}
            <div className="border-t pt-2">
              <div className="text-xs font-semibold text-gray-600 mb-1">
                Histórico recente:
              </div>
              <div className="space-y-1 max-h-32 overflow-y-auto">
                {memoryHistory.slice(-5).reverse().map((usage, index) => (
                  <div key={usage.timestamp} className="flex justify-between text-xs text-gray-600">
                    <span>{new Date(usage.timestamp).toLocaleTimeString()}</span>
                    <span className={getMBFromBytes(usage.usedJSHeapSize) > threshold ? 'text-red-600' : ''}>
                      {formatBytes(usage.usedJSHeapSize)}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            {/* Alerta se memória alta */}
            {isHighMemory && (
              <div className="bg-red-50 border border-red-200 rounded p-2">
                <p className="text-xs text-red-700 flex items-center gap-1">
                  <AlertTriangle className="w-3 h-3" />
                  Uso de memória alto! Considere:
                </p>
                <ul className="text-xs text-red-600 mt-1 ml-4 list-disc">
                  <li>Forçar Garbage Collection</li>
                  <li>Fechar abas desnecessárias</li>
                  <li>Recarregar a página</li>
                </ul>
              </div>
            )}

            {/* Dicas de otimização */}
            <div className="text-xs text-muted-foreground border-t pt-2">
              <p><strong>Dicas:</strong></p>
              <ul className="list-disc ml-4 space-y-1">
                <li>Abra Dev Tools → Memory para análise detalhada</li>
                <li>Verifique vazamentos com heap snapshots</li>
                <li>Monitore event listeners não removidos</li>
              </ul>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

// Hook para usar programaticamente
export function useMemoryMonitor() {
  const [currentMemory, setCurrentMemory] = useState<MemoryUsage | null>(null);
  
  useEffect(() => {
    if (!('memory' in performance)) return;
    
    const updateMemory = () => {
      const memory = (performance as any).memory;
      setCurrentMemory({
        usedJSHeapSize: memory.usedJSHeapSize,
        totalJSHeapSize: memory.totalJSHeapSize,
        jsHeapSizeLimit: memory.jsHeapSizeLimit,
        timestamp: Date.now(),
      });
    };
    
    updateMemory();
    // ❌ DESABILITADO: setInterval que estava causando re-renders a cada 10s
    // const interval = setInterval(updateMemory, 10000);
    
    return () => {
      // clearInterval(interval);
    };
  }, []);
  
  return {
    currentMemory,
    isHighMemory: currentMemory ? currentMemory.usedJSHeapSize / (1024 * 1024) > 500 : false,
    formatBytes: (bytes: number) => {
      if (bytes === 0) return '0 B';
      const k = 1024;
      const sizes = ['B', 'KB', 'MB', 'GB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }
  };
}