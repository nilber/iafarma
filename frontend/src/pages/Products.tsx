import { useState, useEffect, useCallback, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Package, Plus, Search, Filter, Loader2, Edit, ChevronLeft, ChevronRight, Trash2, Upload, Image, ChevronDown, Clock } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { TableSkeleton } from "@/components/ui/loading-skeletons";
import { ErrorState } from "@/components/ui/empty-state";
import { useProducts, useProductsAdmin, useDeleteProduct, useCategories, useProductStats } from "@/lib/api/hooks";
import { Product } from "@/lib/api/types";
import ProductForm from "@/components/ProductForm";
import { ProductImportDialog } from "@/components/ProductImportDialog";
import { ProductAsyncImportDialog } from "@/components/ProductAsyncImportDialog";
import { ProductImageImportDialog } from "@/components/ProductImageImportDialog";
import { ImportJobsDialog } from "@/components/ImportJobsDialog";
import { ProductFiltersSheet, ProductFilters } from "@/components/ProductFiltersSheet";
import { StockStatusBadge } from "@/components/StockStatusBadge";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";
import { toast } from "sonner";

export default function Products() {
  const navigate = useNavigate();
  const [searchTerm, setSearchTerm] = useState("");
  const [debouncedSearchTerm, setDebouncedSearchTerm] = useState("");
  const [page, setPage] = useState(1);
  const [isImportDialogOpen, setIsImportDialogOpen] = useState(false);
  const [isAsyncImportDialogOpen, setIsAsyncImportDialogOpen] = useState(false);
  const [isImageImportDialogOpen, setIsImageImportDialogOpen] = useState(false);
  const [isJobsDialogOpen, setIsJobsDialogOpen] = useState(false);
  const [filtersSheetOpen, setFiltersSheetOpen] = useState(false);
  const [activeFilters, setActiveFilters] = useState<ProductFilters>({});
  const [deletingProduct, setDeletingProduct] = useState<Product | undefined>();
  const [isSearching, setIsSearching] = useState(false);
  const limit = 20;

  // Improved debounce with search loading state
  useEffect(() => {
    if (searchTerm !== debouncedSearchTerm) {
      setIsSearching(true);
    }
    
    const timer = setTimeout(() => {
      setDebouncedSearchTerm(searchTerm);
      setPage(1); // Reset to first page when search changes
      setIsSearching(false);
    }, 800); // Increased to 800ms to reduce API calls

    return () => clearTimeout(timer);
  }, [searchTerm, debouncedSearchTerm]);

  // Reset page when filters change
  useEffect(() => {
    setPage(1);
  }, [activeFilters]);

  // Convert filters to API params
  const getApiParams = useMemo(() => {
    const params: any = {
      limit,
      offset: (page - 1) * limit,
      search: debouncedSearchTerm.trim() || undefined,
    };

    // Add product filters
    if (activeFilters.category) {
      params.category_id = activeFilters.category;
    }
    if (activeFilters.minPrice !== undefined && activeFilters.minPrice > 0) {
      params.min_price = activeFilters.minPrice;
    }
    if (activeFilters.maxPrice !== undefined && activeFilters.maxPrice > 0) {
      params.max_price = activeFilters.maxPrice;
    }
    if (activeFilters.hasPromotion) {
      params.has_promotion = true;
    }
    if (activeFilters.hasSku) {
      params.has_sku = true;
    }
    if (activeFilters.hasStock) {
      params.has_stock = true;
    }
    if (activeFilters.outOfStock) {
      params.out_of_stock = true;
    }

    return params;
  }, [page, limit, debouncedSearchTerm, activeFilters]);

  const { data: productResult, isLoading, error, isFetching } = useProductsAdmin(getApiParams);

  const { data: categories } = useCategories();
  const { data: productStats, isLoading: statsLoading } = useProductStats();

  const deleteProduct = useDeleteProduct();

  const getCategoryName = useCallback((categoryId?: string) => {
    if (!categoryId || !categories) return 'Não categorizado';
    const category = categories.find(cat => cat.id === categoryId);
    return category?.name || 'Categoria não encontrada';
  }, [categories]);

  const products = productResult?.data || [];
  const totalPages = productResult?.total_pages || 1;

  // Memoized calculations to avoid unnecessary re-renders
  const filteredProducts = useMemo(() => products, [products]);
  
  // Use server-side statistics instead of client-side calculations
  const stats = useMemo(() => {
    if (productStats) {
      return {
        total: productStats.total,
        withSku: productStats.with_sku,
        withPromotion: productStats.with_promotion,
        averagePrice: productStats.average_price
      };
    }
    
    // Fallback to current page calculation if stats not loaded
    const totalValue = products.reduce((sum, product) => {
      return sum + parseFloat(product.price);
    }, 0);
    
    return {
      total: products.length,
      withSku: products.filter(p => p.sku).length,
      withPromotion: products.filter(p => p.sale_price).length,
      averagePrice: products.length > 0 ? totalValue / products.length : 0
    };
  }, [products, productStats]);

  // Optimized handlers with useCallback
  const handleNewProduct = useCallback(() => {
    navigate('/products/new');
  }, [navigate]);

  const handleEditProduct = useCallback((product: Product) => {
    navigate(`/products/${product.id}`);
  }, [navigate]);

  const handleDeleteProduct = useCallback(async () => {
    if (!deletingProduct) return;

    try {
      await deleteProduct.mutateAsync(deletingProduct.id);
      toast.success("Produto excluído com sucesso!");
      setDeletingProduct(undefined);
    } catch (error) {
      toast.error("Erro ao excluir produto");
    }
  }, [deletingProduct, deleteProduct]);

  // Optimized pagination handlers
  const handlePreviousPage = useCallback(() => {
    setPage(p => Math.max(1, p - 1));
  }, []);

  const handleNextPage = useCallback(() => {
    setPage(p => Math.min(totalPages, p + 1));
  }, [totalPages]);

  // Remove full-page loading and handle errors locally
  const hasError = error;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Produtos</h1>
          <p className="text-muted-foreground">Gerencie seu catálogo de produtos</p>
        </div>
        <div className="flex gap-2">
          <div className="flex gap-1">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" className="rounded-r-none">
                  <Upload className="w-4 h-4 mr-2" />
                  Importar CSV
                  <ChevronDown className="w-4 h-4 ml-2" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                <DropdownMenuItem onClick={() => setIsImportDialogOpen(true)}>
                  <Upload className="w-4 h-4 mr-2" />
                  Importação Rápida (Síncrona)
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setIsAsyncImportDialogOpen(true)}>
                  <Clock className="w-4 h-4 mr-2" />
                  Importação em Lote (Assíncrona)
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setIsJobsDialogOpen(true)}>
                  <Clock className="w-4 h-4 mr-2" />
                  Gerenciar Jobs de Importação
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <Button 
              variant="outline" 
              onClick={() => setIsImageImportDialogOpen(true)}
              className="rounded-l-none border-l-0 bg-muted hover:bg-accent"
            >
              <Image className="w-4 h-4 mr-2" />
              Importar com IA
            </Button>
          </div>
          <Button className="bg-gradient-primary" onClick={handleNewProduct}>
            <Plus className="w-4 h-4 mr-2" />
            Novo Produto
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total de Produtos</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">
              {statsLoading ? (
                <Skeleton className="h-7 w-16" />
              ) : (
                stats.total
              )}
            </div>
            <p className="text-sm text-success">
              {productStats ? 'Dados completos da base' : 'Dados da página atual'}
            </p>
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Com SKU</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">
              {statsLoading ? (
                <Skeleton className="h-7 w-16" />
              ) : (
                stats.withSku
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              {stats.total > 0 ? Math.round((stats.withSku / stats.total) * 100) : 0}% do total
            </p>
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Com Promoção</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-warning">
              {statsLoading ? (
                <Skeleton className="h-7 w-16" />
              ) : (
                stats.withPromotion
              )}
            </div>
            <p className="text-sm text-warning">
              {stats.total > 0 ? Math.round((stats.withPromotion / stats.total) * 100) : 0}% do total
            </p>
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Valor Médio</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">
              {statsLoading ? (
                <Skeleton className="h-7 w-16" />
              ) : (
                `R$ ${stats.averagePrice.toFixed(2)}`
              )}
            </div>
            <p className="text-sm text-success">
              {productStats ? 'Base de dados completa' : 'Página atual'}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Filters and Search */}
      <Card className="border-0 shadow-custom-md">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Package className="w-5 h-5 text-primary" />
              Lista de Produtos ({filteredProducts.length})
            </CardTitle>
            <div className="flex items-center gap-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                {(isSearching || isFetching) && (
                  <Loader2 className="absolute right-3 top-1/2 transform -translate-y-1/2 w-4 h-4 animate-spin text-primary" />
                )}
                <Input 
                  placeholder="Buscar produtos, SKU ou descrição..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className={`pl-10 ${(isSearching || isFetching) ? 'pr-10' : ''} w-80 transition-all duration-200`}
                />
              </div>
              <Button variant="outline" size="sm" onClick={() => setFiltersSheetOpen(true)}>
                <Filter className="w-4 h-4 mr-2" />
                Filtros
                {Object.keys(activeFilters).filter(key => activeFilters[key as keyof ProductFilters] !== undefined && activeFilters[key as keyof ProductFilters] !== '').length > 0 && (
                  <span className="ml-2 bg-primary text-primary-foreground text-xs rounded-full w-5 h-5 flex items-center justify-center">
                    {Object.keys(activeFilters).filter(key => activeFilters[key as keyof ProductFilters] !== undefined && activeFilters[key as keyof ProductFilters] !== '').length}
                  </span>
                )}
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Produto</TableHead>
                <TableHead>SKU</TableHead>
                <TableHead>Categoria</TableHead>
                <TableHead>Preço</TableHead>
                <TableHead>Preço Promoção</TableHead>
                <TableHead>Estoque</TableHead>
                <TableHead>Criado em</TableHead>
                <TableHead className="w-24">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {hasError ? (
                <TableRow>
                  <TableCell colSpan={8} className="p-0">
                    <ErrorState
                      message={error?.message}
                      onRetry={() => window.location.reload()}
                    />
                  </TableCell>
                </TableRow>
              ) : (isLoading || (isSearching && !products.length)) ? (
                <TableSkeleton rows={5} columns={8} />
              ) : (
                filteredProducts.map((product) => {
                  const stock = product.stock_quantity || 0;
                  const threshold = product.low_stock_threshold || 5;
                  const isOutOfStock = stock === 0;
                  const isLowStock = stock > 0 && stock <= threshold;

                  return (
                    <TableRow 
                      key={product.id} 
                      className={`hover:bg-accent/50 ${
                        isOutOfStock 
                          ? 'bg-red-50 border-l-4 border-red-500' 
                          : isLowStock 
                          ? 'bg-amber-50 border-l-4 border-amber-500' 
                          : ''
                      }`}
                    >
                    <TableCell>
                      <div>
                        <div className="font-medium text-foreground">{product.name}</div>
                        {product.description && (
                          <div className="text-sm text-muted-foreground line-clamp-1">
                            {product.description}
                          </div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      {product.sku ? (
                        <code className="text-sm bg-muted px-2 py-1 rounded">{product.sku}</code>
                      ) : (
                        <span className="text-sm text-muted-foreground">Não informado</span>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {getCategoryName(product.category_id)}
                    </TableCell>
                    <TableCell className="font-medium">
                      R$ {parseFloat(product.price).toFixed(2).replace('.', ',')}
                    </TableCell>
                    <TableCell>
                      {product.sale_price ? (
                        <span className="font-medium text-success">
                          R$ {parseFloat(product.sale_price).toFixed(2).replace('.', ',')}
                        </span>
                      ) : (
                        <span className="text-sm text-muted-foreground">-</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <StockStatusBadge 
                        stockQuantity={product.stock_quantity} 
                        lowStockThreshold={product.low_stock_threshold} 
                      />
                    </TableCell>
                    <TableCell>
                      <span className="text-sm text-muted-foreground">
                        {format(new Date(product.created_at), 'dd/MM/yyyy', { locale: ptBR })}
                      </span>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Button 
                          variant="ghost" 
                          size="sm"
                          onClick={() => handleEditProduct(product)}
                        >
                          <Edit className="w-4 h-4 mr-2" />
                          Editar
                        </Button>
                        <Button 
                          variant="ghost" 
                          size="sm"
                          onClick={() => setDeletingProduct(product)}
                        >
                          <Trash2 className="w-4 h-4 mr-2 text-destructive" />
                          Excluir
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
                })
              )}
            </TableBody>
          </Table>
          
          {filteredProducts.length === 0 && !isLoading && !isSearching && (
            <div className="text-center py-8">
              <p className="text-muted-foreground">
                {searchTerm ? `Nenhum produto encontrado para "${searchTerm}"` : 'Nenhum produto cadastrado'}
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Pagination */}
      {totalPages > 1 && (
        <Card className="border-0 shadow-custom-md">
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div className="text-sm text-muted-foreground">
                Página {page} de {totalPages} • Total: {productResult?.total || 0} produtos
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handlePreviousPage}
                  disabled={page === 1}
                >
                  <ChevronLeft className="w-4 h-4" />
                  Anterior
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleNextPage}
                  disabled={page === totalPages}
                >
                  Próximo
                  <ChevronRight className="w-4 h-4 ml-1" />
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Alert Dialog for Delete Confirmation */}
      <AlertDialog open={!!deletingProduct} onOpenChange={() => setDeletingProduct(undefined)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Confirmar exclusão</AlertDialogTitle>
            <AlertDialogDescription>
              Tem certeza que deseja excluir o produto "{deletingProduct?.name}"? 
              Esta ação não pode ser desfeita.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancelar</AlertDialogCancel>
            <AlertDialogAction 
              onClick={handleDeleteProduct}
              disabled={deleteProduct.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteProduct.isPending && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              Excluir Produto
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Import Dialog */}
      <ProductImportDialog 
        open={isImportDialogOpen}
        onOpenChange={setIsImportDialogOpen}
      />

      {/* Async Import Dialog */}
      <ProductAsyncImportDialog 
        open={isAsyncImportDialogOpen}
        onOpenChange={setIsAsyncImportDialogOpen}
      />

      {/* Image Import Dialog */}
      <ProductImageImportDialog 
        open={isImageImportDialogOpen}
        onOpenChange={setIsImageImportDialogOpen}
      />

      {/* Import Jobs Dialog */}
      <ImportJobsDialog 
        open={isJobsDialogOpen}
        onOpenChange={setIsJobsDialogOpen}
      />

      {/* Filters Sheet */}
      <ProductFiltersSheet
        open={filtersSheetOpen}
        onOpenChange={setFiltersSheetOpen}
        filters={activeFilters}
        onFiltersChange={setActiveFilters}
      />
    </div>
  );
}