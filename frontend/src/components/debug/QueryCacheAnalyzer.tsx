import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Database, Trash2, Eye, EyeOff, RefreshCw } from 'lucide-react';

interface QueryCacheItem {
  queryKey: string;
  dataSize: number;
  state: string;
  lastUpdated: string;
  isFetching: boolean;
  refetchInterval?: number;
}

interface QueryCacheAnalyzerProps {
  enabled?: boolean;
  position?: 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right';
}

export function QueryCacheAnalyzer({ 
  enabled = true, 
  position = 'top-left'
}: QueryCacheAnalyzerProps) {
  const [isVisible, setIsVisible] = useState(false);
  const [cacheItems, setCacheItems] = useState<QueryCacheItem[]>([]);
  const [totalCacheSize, setTotalCacheSize] = useState(0);
  const queryClient = useQueryClient();

  const calculateSize = (obj: any): number => {
    try {
      return new Blob([JSON.stringify(obj)]).size;
    } catch {
      return 0;
    }
  };

  const analyzeCacheData = () => {
    const cache = queryClient.getQueryCache();
    const queries = cache.getAll();
    
    let totalSize = 0;
    const items: QueryCacheItem[] = [];

    queries.forEach((query) => {
      const dataSize = calculateSize(query.state.data);
      totalSize += dataSize;
      
      items.push({
        queryKey: JSON.stringify(query.queryKey),
        dataSize,
        state: query.state.status,
        lastUpdated: query.state.dataUpdatedAt ? new Date(query.state.dataUpdatedAt).toLocaleTimeString() : 'Nunca',
        isFetching: query.state.fetchStatus === 'fetching',
        refetchInterval: (query.options as any)?.refetchInterval,
      });
    });

    // Ordenar por tamanho (maior primeiro)
    items.sort((a, b) => b.dataSize - a.dataSize);
    
    setCacheItems(items);
    setTotalCacheSize(totalSize);
  };

  const clearCache = () => {
    queryClient.clear();
    analyzeCacheData();
    console.log('🗑️ Cache do React Query limpo');
  };

  const clearSpecificQuery = (queryKey: string) => {
    const parsedKey = JSON.parse(queryKey);
    queryClient.removeQueries({ queryKey: parsedKey });
    analyzeCacheData();
    console.log('🗑️ Query removida:', queryKey);
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  useEffect(() => {
    if (enabled && isVisible) {
      analyzeCacheData();
      // ❌ DESABILITADO: setInterval que estava causando re-renders a cada 5s
      // const interval = setInterval(analyzeCacheData, 5000);
      // return () => clearInterval(interval);
    }
  }, [enabled, isVisible]);

  const positionClasses = {
    'top-left': 'top-4 left-4',
    'top-right': 'top-4 right-4',  
    'bottom-left': 'bottom-4 left-4',
    'bottom-right': 'bottom-4 right-4',
  };

  if (!enabled) return null;

  const isHighCache = totalCacheSize > 50 * 1024 * 1024; // 50MB
  const queryCount = cacheItems.length;
  const activePolling = cacheItems.filter(item => item.refetchInterval).length;

  return (
    <div className={`fixed ${positionClasses[position]} z-[9996] max-w-lg`}>
      {/* Botão toggle */}
      <Button
        variant="outline"
        size="sm"
        onClick={() => setIsVisible(!isVisible)}
        className={`mb-2 ${isHighCache ? 'bg-orange-500 text-white border-orange-500 hover:bg-orange-600' : 'bg-blue-500 text-white border-blue-500 hover:bg-blue-600'} shadow-lg`}
        title="Analisador de Cache React Query"
      >
        <Database className="w-4 h-4 mr-1" />
        {isVisible ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
        <Badge variant={isHighCache ? "destructive" : "secondary"} className="ml-1 text-xs">
          {formatBytes(totalCacheSize)}
        </Badge>
        {activePolling > 0 && (
          <Badge variant="outline" className="ml-1 text-xs">
            {activePolling} polling
          </Badge>
        )}
      </Button>

      {/* Painel de análise */}
      {isVisible && (
        <Card className="w-96 max-h-96 shadow-xl border-blue-200 bg-white/95 backdrop-blur-sm">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm flex items-center gap-2">
                <Database className="w-4 h-4 text-blue-500" />
                Cache React Query
                {isHighCache && (
                  <Badge variant="destructive" className="text-xs">
                    ⚠️ Alto
                  </Badge>
                )}
              </CardTitle>
              <div className="flex gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={analyzeCacheData}
                  title="Atualizar análise"
                >
                  <RefreshCw className="w-4 h-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={clearCache}
                  title="Limpar todo o cache"
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-3 space-y-3">
            {/* Resumo */}
            <div className="grid grid-cols-3 gap-2 text-sm">
              <div className="text-center">
                <div className="font-semibold">{queryCount}</div>
                <div className="text-xs text-muted-foreground">Queries</div>
              </div>
              <div className="text-center">
                <div className="font-semibold">{formatBytes(totalCacheSize)}</div>
                <div className="text-xs text-muted-foreground">Cache</div>
              </div>
              <div className="text-center">
                <div className="font-semibold text-orange-600">{activePolling}</div>
                <div className="text-xs text-muted-foreground">Polling</div>
              </div>
            </div>

            {/* Lista de queries maiores */}
            <div className="border-t pt-2">
              <div className="text-xs font-semibold text-gray-600 mb-2">
                Maiores queries no cache:
              </div>
              <div className="space-y-1 max-h-48 overflow-y-auto">
                {cacheItems.slice(0, 10).map((item, index) => (
                  <div key={index} className="flex items-center justify-between text-xs p-2 rounded bg-gray-50 hover:bg-gray-100">
                    <div className="flex-1 min-w-0">
                      <div className="font-mono text-xs truncate" title={item.queryKey}>
                        {item.queryKey.length > 40 ? `${item.queryKey.substring(0, 40)}...` : item.queryKey}
                      </div>
                      <div className="flex items-center gap-2 mt-1">
                        <Badge variant={item.state === 'success' ? 'default' : 'destructive'} className="text-xs">
                          {item.state}
                        </Badge>
                        {item.isFetching && (
                          <Badge variant="outline" className="text-xs">
                            Fetching
                          </Badge>
                        )}
                        {item.refetchInterval && (
                          <Badge variant="secondary" className="text-xs">
                            {item.refetchInterval / 1000}s polling
                          </Badge>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-2 ml-2">
                      <span className={`font-mono ${item.dataSize > 1024 * 1024 ? 'text-red-600' : 'text-gray-600'}`}>
                        {formatBytes(item.dataSize)}
                      </span>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => clearSpecificQuery(item.queryKey)}
                        className="h-6 w-6 p-0"
                        title="Remover esta query"
                      >
                        <Trash2 className="w-3 h-3" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Alertas e dicas */}
            {(isHighCache || activePolling > 5) && (
              <div className="bg-orange-50 border border-orange-200 rounded p-2">
                <p className="text-xs text-orange-700 font-semibold mb-1">
                  ⚠️ Possíveis problemas de memória:
                </p>
                <ul className="text-xs text-orange-600 space-y-1">
                  {isHighCache && <li>• Cache muito grande ({formatBytes(totalCacheSize)})</li>}
                  {activePolling > 5 && <li>• Muitas queries com polling ativo ({activePolling})</li>}
                  <li>• Considere aumentar staleTime nas queries</li>
                  <li>• Revise refetchInterval desnecessários</li>
                </ul>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}