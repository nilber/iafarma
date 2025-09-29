import { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { Search, User, ShoppingCart, Loader2, Command } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { useNavigate } from 'react-router-dom';
import { useCustomers, useOrders } from '@/lib/api/hooks';
import { Customer, OrderWithCustomer } from '@/lib/api/types';
import { cn } from '@/lib/utils';
// Debug hooks removidos

// Custom hook para debounce
function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}

interface SearchResult {
  type: 'customer' | 'order';
  id: string;
  title: string;
  subtitle: string;
  url: string;
  data: Customer | OrderWithCustomer;
}

interface GlobalSearchProps {
  className?: string;
}

export function GlobalSearch({ className }: GlobalSearchProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  
  const modalInputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();

  // Debounce do searchTerm para evitar chamadas excessivas
  const debouncedSearchTerm = useDebounce(searchTerm, 300);
  
  // Memoizar se deve fazer busca para evitar re-renders
  const shouldSearch = useMemo(() => 
    debouncedSearchTerm.length >= 2, 
    [debouncedSearchTerm]
  );

  // Fetch data with debounced search term
  const { data: customersResult, isLoading: customersLoading } = useCustomers({ 
    search: shouldSearch ? debouncedSearchTerm : '',
    limit: 10 
  });
  
  const { data: ordersResult, isLoading: ordersLoading } = useOrders({ 
    search: shouldSearch ? debouncedSearchTerm : '',
    limit: 10 
  });

  const isLoading = customersLoading || ordersLoading;

  // Memoizar results para evitar re-renders desnecessários
  const results = useMemo(() => {
    if (!shouldSearch) {
      return [];
    }

    const newResults: SearchResult[] = [];

    // Process customers
    if (customersResult?.data) {
      customersResult.data.forEach((customer) => {
        newResults.push({
          type: 'customer',
          id: customer.id,
          title: customer.name || 'Cliente sem nome',
          subtitle: customer.phone || customer.email || 'Sem contato',
          url: `/customers`,
          data: customer
        });
      });
    }

    // Process orders
    if (ordersResult?.data) {
      ordersResult.data.forEach((order) => {
        newResults.push({
          type: 'order',
          id: order.id,
          title: order.order_number || `Pedido #${order.id.slice(0, 8)}`,
          subtitle: order.customer_name || order.customer_phone || 'Cliente não identificado',
          url: `/sales/orders/${order.id}`,
          data: order
        });
      });
    }

    return newResults;
  }, [customersResult, ordersResult, shouldSearch]);

  // Auto focus no modal quando abrir
  useEffect(() => {
    if (isModalOpen && modalInputRef.current) {
      modalInputRef.current.focus();
    }
  }, [isModalOpen]);

  // Keyboard shortcut para abrir modal (Ctrl+K / Cmd+K)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        setIsModalOpen(true);
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  // Debug hooks removidos para eliminar re-renders

  // Memoizar handlers para evitar re-renders
  const handleResultClick = useCallback((result: SearchResult) => {
    if (result.type === 'customer') {
      navigate(`/customers/${result.id}`);
    } else {
      navigate(result.url);
    }
    
    setSearchTerm('');
    setIsModalOpen(false);
    setSelectedIndex(-1);
  }, [navigate]);

  // Handle keyboard navigation no modal
  const handleModalKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (results.length === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex(prev => prev < results.length - 1 ? prev + 1 : 0);
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex(prev => prev > 0 ? prev - 1 : results.length - 1);
        break;
      case 'Enter':
        e.preventDefault();
        if (selectedIndex >= 0 && results[selectedIndex]) {
          handleResultClick(results[selectedIndex]);
        }
        break;
    }
  }, [results.length, selectedIndex, handleResultClick]);

  // Reset ao fechar modal
  const handleModalClose = useCallback(() => {
    setIsModalOpen(false);
    setSearchTerm('');
    setSelectedIndex(-1);
  }, []);

  // Memoizar funções auxiliares
  const getResultIcon = useCallback((type: 'customer' | 'order') => {
    switch (type) {
      case 'customer':
        return <User className="w-4 h-4 text-blue-500" />;
      case 'order':
        return <ShoppingCart className="w-4 h-4 text-green-500" />;
    }
  }, []);

  // Get badge for result type
  const getResultBadge = useCallback((type: 'customer' | 'order') => {
    switch (type) {
      case 'customer':
        return <Badge variant="secondary" className="text-xs">Cliente</Badge>;
      case 'order':
        return <Badge variant="outline" className="text-xs">Pedido</Badge>;
    }
  }, []);

  return (
    <>
      {/* Search Trigger Button */}
      <Button
        variant="outline"
        className={cn("relative max-w-md justify-start text-muted-foreground", className)}
        onClick={() => setIsModalOpen(true)}
      >
        <Search className="w-4 h-4 mr-2" />
        <span>Buscar clientes, pedidos...</span>
        <div className="ml-auto flex items-center gap-1">
          <kbd className="pointer-events-none h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium opacity-100 flex">
            <Command className="w-3 h-3" />
            K
          </kbd>
        </div>
      </Button>

      {/* Search Modal */}
      <Dialog open={isModalOpen} onOpenChange={handleModalClose}>
        <DialogContent className="sm:max-w-2xl p-0">
          <DialogHeader className="p-6 pb-2">
            <DialogTitle className="text-left">Buscar</DialogTitle>
          </DialogHeader>
          
          <div className="px-6">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input 
                ref={modalInputRef}
                placeholder="Digite para buscar clientes, pedidos..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                onKeyDown={handleModalKeyDown}
                className="pl-10 text-base"
                autoComplete="off"
              />
              {isLoading && (
                <Loader2 className="absolute right-3 top-1/2 transform -translate-y-1/2 w-4 h-4 animate-spin text-muted-foreground" />
              )}
            </div>
          </div>

          {/* Results */}
          <div className="max-h-96 overflow-y-auto">
            {searchTerm.length >= 2 && (
              <div className="p-6 pt-4">
                {results.length > 0 && (
                  <div className="mb-2">
                    <p className="text-xs text-muted-foreground px-2">
                      {results.length} resultado{results.length > 1 ? 's' : ''} encontrado{results.length > 1 ? 's' : ''}
                    </p>
                  </div>
                )}

                {results.length > 0 ? (
                  <div className="space-y-1">
                    {results.map((result, index) => (
                      <div
                        key={`${result.type}-${result.id}`}
                        className={cn(
                          "flex items-center gap-3 p-3 rounded-md cursor-pointer transition-colors",
                          index === selectedIndex ? "bg-accent" : "hover:bg-accent/50"
                        )}
                        onClick={() => handleResultClick(result)}
                      >
                        <div className="flex-shrink-0">
                          {getResultIcon(result.type)}
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <h4 className="text-sm font-medium text-foreground truncate">
                              {result.title}
                            </h4>
                            {getResultBadge(result.type)}
                          </div>
                          <p className="text-xs text-muted-foreground truncate">
                            {result.subtitle}
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  !isLoading && (
                    <div className="text-center py-8">
                      <p className="text-sm text-muted-foreground">
                        Nenhum resultado encontrado para "{searchTerm}"
                      </p>
                    </div>
                  )
                )}
              </div>
            )}

            {searchTerm.length < 2 && (
              <div className="p-6 pt-4">
                <div className="text-center py-8">
                  <Search className="w-8 h-8 text-muted-foreground mx-auto mb-2" />
                  <p className="text-sm text-muted-foreground">
                    Digite pelo menos 2 caracteres para buscar
                  </p>
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
