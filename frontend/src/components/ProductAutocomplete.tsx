import React, { useState, useEffect, useRef } from 'react';
import { Search, Loader2 } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { useProducts } from '@/lib/api/hooks';
import { cn } from '@/lib/utils';

interface Product {
  id: string;
  name: string;
  price: string;
  description?: string;
  stock_quantity?: number;
}

interface ProductAutocompleteProps {
  value: string;
  onChange: (value: string) => void;
  onProductSelect: (product: Product) => void;
  placeholder?: string;
  className?: string;
}

export function ProductAutocomplete({
  value,
  onChange,
  onProductSelect,
  placeholder = "Digite para buscar produtos...",
  className
}: ProductAutocompleteProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState(value || '');
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Usar o hook de produtos com busca
  const { data: productsData, isLoading } = useProducts({ 
    search: searchTerm,
    limit: 50 
  });

  const products = productsData?.data || [];

  // Debounce para a busca
  useEffect(() => {
    const timer = setTimeout(() => {
      onChange(searchTerm);
    }, 300);

    return () => clearTimeout(timer);
  }, [searchTerm, onChange]);

  // Fechar dropdown quando clicar fora
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current && 
        !dropdownRef.current.contains(event.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setSearchTerm(newValue);
    setIsOpen(true);
    setSelectedProduct(null);
  };

  const handleProductClick = (product: Product) => {
    setSelectedProduct(product);
    setSearchTerm(product.name);
    setIsOpen(false);
    onProductSelect(product);
  };

  const handleInputFocus = () => {
    setIsOpen(true);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      setIsOpen(false);
    }
  };

  return (
    <div className={cn("relative w-full", className)}>
      <div className="relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
        <Input
          ref={inputRef}
          type="text"
          value={searchTerm}
          onChange={handleInputChange}
          onFocus={handleInputFocus}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          className="pl-10 pr-10"
          autoComplete="off"
        />
        {isLoading && (
          <Loader2 className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4 animate-spin" />
        )}
      </div>

      {isOpen && (
        <Card 
          ref={dropdownRef}
          className="absolute z-50 w-full mt-1 max-h-96 overflow-y-auto shadow-lg border"
        >
          <CardContent className="p-0">
            {isLoading ? (
              <div className="p-4 text-center text-gray-500">
                <Loader2 className="h-4 w-4 animate-spin mx-auto mb-2" />
                Buscando produtos...
              </div>
            ) : products.length > 0 ? (
              <div className="divide-y">
                {products.map((product) => (
                  <Button
                    key={product.id}
                    variant="ghost"
                    className="w-full justify-start p-4 h-auto hover:bg-gray-50 rounded-none"
                    onClick={() => handleProductClick(product)}
                  >
                    <div className="flex flex-col items-start w-full">
                      <div className="font-medium text-left">{product.name}</div>
                      <div className="flex justify-between items-center w-full mt-1">
                        <span className="text-sm text-gray-500">
                          {product.description && product.description.length > 50
                            ? `${product.description.substring(0, 50)}...`
                            : product.description || 'Sem descrição'
                          }
                        </span>
                        <div className="flex items-center gap-2">
                          {product.stock_quantity !== undefined && (
                            <span className="text-xs bg-gray-100 px-2 py-1 rounded">
                              Estoque: {product.stock_quantity}
                            </span>
                          )}
                          <span className="font-semibold text-green-600">
                            R$ {parseFloat(product.price).toFixed(2)}
                          </span>
                        </div>
                      </div>
                    </div>
                  </Button>
                ))}
              </div>
            ) : searchTerm.length > 0 ? (
              <div className="p-4 text-center text-gray-500">
                Nenhum produto encontrado para "{searchTerm}"
              </div>
            ) : (
              <div className="p-4 text-center text-gray-500">
                Digite para buscar produtos
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
