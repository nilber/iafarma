import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { CurrencyInput } from "@/components/ui/currency-input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Loader2 } from "lucide-react";
import { Product, ProductCreateRequest } from "@/lib/api/types";
import { useCreateProduct, useUpdateProduct, useCategories } from "@/lib/api/hooks";
import { toast } from "sonner";
import { AIProductGenerator } from "./AIProductGenerator";

interface ProductFormProps {
  product?: Product;
  onSuccess: () => void;
  onCancel: () => void;
}

type ProductFormData = {
  name: string;
  description: string;
  price: string;
  sale_price: string;
  sku: string;
  brand: string;
  weight: string;
  dimensions: string;
  barcode: string;
  tags: string;
  stock_quantity: number;
  low_stock_threshold: number;
  category_id: string;
};

export default function ProductForm({ product, onSuccess, onCancel }: ProductFormProps) {
  const [formData, setFormData] = useState<ProductFormData>({
    name: "",
    description: "",
    price: "",
    sale_price: "",
    sku: "",
    brand: "",
    weight: "",
    dimensions: "",
    barcode: "",
    tags: "",
    stock_quantity: 0,
    low_stock_threshold: 5,
    category_id: "",
  });

  const createProduct = useCreateProduct();
  const updateProduct = useUpdateProduct();
  const { data: categories = [], isLoading: categoriesLoading } = useCategories();

  const isEditing = !!product;
  const isLoading = createProduct.isPending || updateProduct.isPending;

  useEffect(() => {
    if (product) {
      setFormData({
        name: product.name || "",
        description: product.description || "",
        price: product.price || "",
        sale_price: product.sale_price || "",
        sku: product.sku || "",
        brand: product.brand || "",
        weight: product.weight || "",
        dimensions: product.dimensions || "",
        barcode: product.barcode || "",
        tags: product.tags || "",
        stock_quantity: product.stock_quantity || 0,
        low_stock_threshold: product.low_stock_threshold || 5,
        category_id: product.category_id || "",
      });
    }
  }, [product]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.name.trim()) {
      toast.error("O nome do produto é obrigatório");
      return;
    }

    if (!formData.price.trim()) {
      toast.error("O preço do produto é obrigatório");
      return;
    }

    try {
      // Preparar dados para envio - garantir que category_id seja null se vazio
      const submitData = {
        ...formData,
        category_id: formData.category_id === "" ? null : formData.category_id
      };

      if (isEditing && product) {
        await updateProduct.mutateAsync({
          id: product.id,
          product: submitData,
        });
        toast.success("Produto atualizado com sucesso!");
      } else {
        await createProduct.mutateAsync(submitData as ProductCreateRequest);
        toast.success("Produto criado com sucesso!");
      }
      onSuccess();
    } catch (error) {
      // Extract error message from API response
      let errorMessage = isEditing ? "Erro ao atualizar produto" : "Erro ao criar produto";
      
      if (error instanceof Error && error.message) {
        errorMessage = error.message;
      }
      
      toast.error(errorMessage);
    }
  };

  const formatWeight = (value: string): string => {
    // Remove tudo exceto números, vírgula e ponto
    let cleaned = value.replace(/[^0-9,.]/g, '');
    
    // Substitui vírgula por ponto
    cleaned = cleaned.replace(',', '.');
    
    // Remove pontos extras (manter apenas o primeiro)
    const parts = cleaned.split('.');
    if (parts.length > 2) {
      cleaned = parts[0] + '.' + parts.slice(1).join('');
    }
    
    // Limita a 3 casas decimais
    if (parts.length === 2 && parts[1].length > 3) {
      cleaned = parts[0] + '.' + parts[1].substring(0, 3);
    }
    
    return cleaned;
  };

  const handleChange = (field: keyof ProductFormData, value: string) => {
    if (field === 'stock_quantity' || field === 'low_stock_threshold') {
      setFormData(prev => ({ ...prev, [field]: parseInt(value) || 0 }));
    } else if (field === 'weight') {
      setFormData(prev => ({ ...prev, [field]: formatWeight(value) }));
    } else {
      setFormData(prev => ({ ...prev, [field]: value }));
    }
  };

  const handleAIGeneration = (aiData: {
    name: string;
    description: string;
    price: string;
    sku?: string;
    brand?: string;
    tags?: string;
  }) => {
    setFormData(prev => ({
      ...prev,
      name: aiData.name,
      description: aiData.description,
      price: aiData.price,
      sku: aiData.sku || prev.sku,
      brand: aiData.brand || prev.brand,
      tags: aiData.tags || prev.tags,
    }));
  };

  return (
    <div className="space-y-6">
      {/* AI Product Generator - Only show for new products */}
      {!isEditing && <AIProductGenerator onProductGenerated={handleAIGeneration} />}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="name">Nome do Produto *</Label>
          <Input
            id="name"
            value={formData.name}
            onChange={(e) => handleChange("name", e.target.value)}
            placeholder="Nome do produto"
            required
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="description">Descrição</Label>
          <Textarea
            id="description"
            value={formData.description}
            onChange={(e) => handleChange("description", e.target.value)}
            placeholder="Descrição do produto"
            rows={3}
          />
        </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="price">Preço *</Label>
          <CurrencyInput
            value={formData.price}
            onChange={(value) => handleChange("price", value)}
            placeholder="R$ 0,00"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="sale_price">Preço Promocional</Label>
          <CurrencyInput
            value={formData.sale_price}
            onChange={(value) => handleChange("sale_price", value)}
            placeholder="R$ 0,00"
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="stock_quantity">Quantidade em Estoque</Label>
          <Input
            id="stock_quantity"
            type="number"
            min="0"
            value={formData.stock_quantity}
            onChange={(e) => handleChange("stock_quantity", e.target.value)}
            placeholder="0"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="low_stock_threshold">Alerta de Estoque Baixo</Label>
          <Input
            id="low_stock_threshold"
            type="number"
            min="1"
            value={formData.low_stock_threshold}
            onChange={(e) => handleChange("low_stock_threshold", e.target.value)}
            placeholder="5"
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="brand">Marca</Label>
          <Input
            id="brand"
            value={formData.brand}
            onChange={(e) => handleChange("brand", e.target.value)}
            placeholder="Marca do produto"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="category">Categoria</Label>
          <Select
            value={formData.category_id || "none"}
            onValueChange={(value) => handleChange("category_id", value === "none" ? "" : value)}
          >
            <SelectTrigger>
              <SelectValue placeholder="Selecione uma categoria (opcional)" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">Sem categoria</SelectItem>
              {categories
                .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name))
                .map((category) => (
                  <SelectItem key={category.id} value={category.id}>
                    {category.name}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="barcode">Código de Barras</Label>
          <Input
            id="barcode"
            value={formData.barcode}
            onChange={(e) => handleChange("barcode", e.target.value)}
            placeholder="Código de barras"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="sku">SKU</Label>
          <Input
            id="sku"
            value={formData.sku}
            onChange={(e) => handleChange("sku", e.target.value)}
            placeholder="Código interno do produto"
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="weight">Peso (kg)</Label>
          <Input
            id="weight"
            value={formData.weight}
            onChange={(e) => handleChange("weight", e.target.value)}
            placeholder="Ex: 1.500 (até 3 casas decimais)"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="dimensions">Dimensões</Label>
          <Input
            id="dimensions"
            value={formData.dimensions}
            onChange={(e) => handleChange("dimensions", e.target.value)}
            placeholder="Ex: 20x15x10cm"
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="tags">Tags (separadas por vírgula)</Label>
        <Input
          id="tags"
          value={formData.tags}
          onChange={(e) => handleChange("tags", e.target.value)}
          placeholder="tag1, tag2, tag3"
        />
      </div>

      <div className="flex justify-end gap-3 pt-4 border-t mt-6">
        <Button variant="outline" onClick={onCancel} type="button">
          Cancelar
        </Button>
        <Button type="submit" disabled={isLoading} className="bg-gradient-primary">
          {isLoading && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
          {isEditing ? "Atualizar Produto" : "Salvar Produto"}
        </Button>
      </div>
    </form>
    </div>
  );
}
