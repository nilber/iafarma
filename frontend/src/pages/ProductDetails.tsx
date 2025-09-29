import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeft, Package, Save, Loader2, Settings, Images } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useProduct } from "@/lib/api/hooks";
import ProductForm from "@/components/ProductForm";
import ProductCharacteristics from "@/components/ProductCharacteristics";
import ProductImages from "@/components/ProductImages";
import { Product } from "@/lib/api/types";
import { toast } from "sonner";

export default function ProductDetails() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isNew = id === "new";

  // Only fetch product if we have an ID and it's not "new"
  const shouldFetchProduct = !isNew && !!id;
  const { data: product, isLoading, error } = useProduct(shouldFetchProduct ? id! : "skip");

  const handleSuccess = () => {
    toast.success(isNew ? "Produto criado com sucesso!" : "Produto atualizado com sucesso!");
    navigate("/products");
  };

  const handleCancel = () => {
    navigate("/products");
  };

  if (!isNew && isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => navigate("/products")}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar para Produtos
          </Button>
        </div>
        
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center min-h-[400px]">
              <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!isNew && error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => navigate("/products")}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar para Produtos
          </Button>
        </div>
        
        <Card>
          <CardContent className="p-6">
            <div className="text-center py-8">
              <Package className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2">Produto não encontrado</h3>
              <p className="text-muted-foreground mb-4">
                O produto solicitado não foi encontrado ou você não tem permissão para visualizá-lo.
              </p>
              <Button onClick={() => navigate("/products")}>
                Voltar para Produtos
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => navigate("/products")}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar para Produtos
          </Button>
          
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {isNew ? "Novo Produto" : "Editar Produto"}
            </h1>
            <p className="text-muted-foreground">
              {isNew 
                ? "Adicione um novo produto ao seu catálogo"
                : `Atualize as informações do produto${product?.name ? `: ${product.name}` : ""}`
              }
            </p>
          </div>
        </div>
      </div>

      {/* Product Form with Tabs */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Package className="w-5 h-5" />
            {isNew ? "Novo Produto" : "Gerenciar Produto"}
          </CardTitle>
          <CardDescription>
            {isNew 
              ? "Adicione um novo produto com suas características e imagens"
              : "Atualize as informações do produto, características e imagens"
            }
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="basic" className="w-full">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="basic" className="flex items-center gap-2">
                <Package className="w-4 h-4" />
                Informações Básicas
              </TabsTrigger>
              <TabsTrigger value="characteristics" disabled={isNew} className="flex items-center gap-2">
                <Settings className="w-4 h-4" />
                Características
              </TabsTrigger>
              <TabsTrigger value="images" disabled={isNew} className="flex items-center gap-2">
                <Images className="w-4 h-4" />
                Imagens
              </TabsTrigger>
            </TabsList>
            
            <TabsContent value="basic" className="mt-6">
              <ProductForm
                product={isNew ? undefined : product}
                onSuccess={handleSuccess}
                onCancel={handleCancel}
              />
            </TabsContent>
            
            <TabsContent value="characteristics" className="mt-6">
              {!isNew && product && (
                <ProductCharacteristics productId={product.id} />
              )}
            </TabsContent>
            
            <TabsContent value="images" className="mt-6">
              {!isNew && product && (
                <ProductImages productId={product.id} />
              )}
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  );
}
