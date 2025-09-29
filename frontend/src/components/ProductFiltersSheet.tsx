import React, { useState, useEffect } from 'react';
import { Filter, X } from 'lucide-react';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
import { useCategories } from '@/lib/api/hooks';

export interface ProductFilters {
  category?: string;
  minPrice?: number;
  maxPrice?: number;
  hasPromotion?: boolean;
  hasSku?: boolean;
  hasStock?: boolean;
  outOfStock?: boolean;
}

export interface ProductFiltersSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  filters: ProductFilters;
  onFiltersChange: (filters: ProductFilters) => void;
}

export function ProductFiltersSheet({
  open,
  onOpenChange,
  filters,
  onFiltersChange
}: ProductFiltersSheetProps) {
  const [localFilters, setLocalFilters] = useState<ProductFilters>(filters);
  const { data: categories = [] } = useCategories();

  // Update local filters when props change
  useEffect(() => {
    setLocalFilters(filters);
  }, [filters]);

  const handleApplyFilters = () => {
    onFiltersChange(localFilters);
    onOpenChange(false);
  };

  const handleClearFilters = () => {
    const emptyFilters: ProductFilters = {};
    setLocalFilters(emptyFilters);
    onFiltersChange(emptyFilters);
    onOpenChange(false);
  };

  const handleFilterChange = (key: keyof ProductFilters, value: any) => {
    setLocalFilters(prev => ({
      ...prev,
      [key]: value
    }));
  };

  const removeFilter = (key: keyof ProductFilters) => {
    setLocalFilters(prev => {
      const newFilters = { ...prev };
      delete newFilters[key];
      return newFilters;
    });
  };

  // Get active filters for display
  const getActiveFilters = () => {
    const active: Array<{ key: keyof ProductFilters; label: string; value: any }> = [];
    
    if (localFilters.category) {
      const category = categories.find(c => c.id === localFilters.category);
      active.push({
        key: 'category',
        label: 'Categoria',
        value: category?.name || localFilters.category
      });
    }
    
    if (localFilters.minPrice !== undefined && localFilters.minPrice > 0) {
      active.push({
        key: 'minPrice',
        label: 'Preço mín.',
        value: `R$ ${localFilters.minPrice.toFixed(2)}`
      });
    }
    
    if (localFilters.maxPrice !== undefined && localFilters.maxPrice > 0) {
      active.push({
        key: 'maxPrice',
        label: 'Preço máx.',
        value: `R$ ${localFilters.maxPrice.toFixed(2)}`
      });
    }
    
    if (localFilters.hasPromotion) {
      active.push({
        key: 'hasPromotion',
        label: 'Com promoção',
        value: 'Sim'
      });
    }
    
    if (localFilters.hasSku) {
      active.push({
        key: 'hasSku',
        label: 'Com SKU',
        value: 'Sim'
      });
    }
    
    if (localFilters.hasStock) {
      active.push({
        key: 'hasStock',
        label: 'Com estoque',
        value: 'Sim'
      });
    }
    
    if (localFilters.outOfStock) {
      active.push({
        key: 'outOfStock',
        label: 'Sem estoque',
        value: 'Sim'
      });
    }
    
    return active;
  };

  const activeFilters = getActiveFilters();

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[400px] sm:w-[540px]">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <Filter className="w-5 h-5" />
            Filtros de Produtos
          </SheetTitle>
          <SheetDescription>
            Aplique filtros para refinar a lista de produtos
          </SheetDescription>
        </SheetHeader>

        <div className="mt-6 space-y-6">
          {/* Active Filters */}
          {activeFilters.length > 0 && (
            <div className="space-y-3">
              <Label className="text-sm font-medium">Filtros Ativos</Label>
              <div className="flex flex-wrap gap-2">
                {activeFilters.map(filter => (
                  <Badge
                    key={filter.key}
                    variant="secondary"
                    className="flex items-center gap-1 py-1 px-2"
                  >
                    <span className="text-xs">
                      {filter.label}: {filter.value}
                    </span>
                    <X
                      className="w-3 h-3 cursor-pointer hover:text-destructive"
                      onClick={() => removeFilter(filter.key)}
                    />
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {/* Category Filter */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="category">Categoria</Label>
              {localFilters.category && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleFilterChange('category', undefined)}
                  className="h-6 px-2 text-xs text-muted-foreground hover:text-foreground"
                >
                  Limpar
                </Button>
              )}
            </div>
            <Select 
              value={localFilters.category || undefined} 
              onValueChange={(value) => handleFilterChange('category', value || undefined)}
            >
              <SelectTrigger>
                <SelectValue placeholder="Selecione uma categoria" />
              </SelectTrigger>
              <SelectContent>
                {categories.map(category => (
                  <SelectItem key={category.id} value={category.id}>
                    {category.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Price Range */}
          <div className="space-y-3">
            <Label>Faixa de Preço</Label>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="minPrice" className="text-xs text-muted-foreground">
                  Preço Mínimo
                </Label>
                <Input
                  id="minPrice"
                  type="number"
                  min="0"
                  step="0.01"
                  placeholder="0,00"
                  value={localFilters.minPrice || ''}
                  onChange={(e) => {
                    const value = parseFloat(e.target.value);
                    handleFilterChange('minPrice', isNaN(value) ? undefined : value);
                  }}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="maxPrice" className="text-xs text-muted-foreground">
                  Preço Máximo
                </Label>
                <Input
                  id="maxPrice"
                  type="number"
                  min="0"
                  step="0.01"
                  placeholder="0,00"
                  value={localFilters.maxPrice || ''}
                  onChange={(e) => {
                    const value = parseFloat(e.target.value);
                    handleFilterChange('maxPrice', isNaN(value) ? undefined : value);
                  }}
                />
              </div>
            </div>
          </div>

          {/* Boolean Filters */}
          <div className="space-y-4">
            <Label>Características</Label>
            
            <div className="space-y-3">
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="hasPromotion"
                  checked={localFilters.hasPromotion || false}
                  onCheckedChange={(checked) => 
                    handleFilterChange('hasPromotion', checked ? true : undefined)
                  }
                />
                <Label htmlFor="hasPromotion" className="text-sm font-normal">
                  Produtos em promoção
                </Label>
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="hasSku"
                  checked={localFilters.hasSku || false}
                  onCheckedChange={(checked) => 
                    handleFilterChange('hasSku', checked ? true : undefined)
                  }
                />
                <Label htmlFor="hasSku" className="text-sm font-normal">
                  Produtos com SKU
                </Label>
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="hasStock"
                  checked={localFilters.hasStock || false}
                  onCheckedChange={(checked) => 
                    handleFilterChange('hasStock', checked ? true : undefined)
                  }
                />
                <Label htmlFor="hasStock" className="text-sm font-normal">
                  Produtos com estoque
                </Label>
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="outOfStock"
                  checked={localFilters.outOfStock || false}
                  onCheckedChange={(checked) => 
                    handleFilterChange('outOfStock', checked ? true : undefined)
                  }
                />
                <Label htmlFor="outOfStock" className="text-sm font-normal">
                  Produtos sem estoque
                </Label>
              </div>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-3 mt-8 pt-6 border-t">
          <Button
            variant="outline"
            onClick={handleClearFilters}
            className="flex-1"
          >
            Limpar Filtros
          </Button>
          <Button
            onClick={handleApplyFilters}
            className="flex-1"
          >
            Aplicar Filtros
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}
