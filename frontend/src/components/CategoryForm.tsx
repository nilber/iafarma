import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Loader2 } from "lucide-react";
import { Category, CategoryCreateRequest } from "@/lib/api/types";
import { useCreateCategory, useUpdateCategory } from "@/lib/api/hooks";
import { toast } from "sonner";

interface CategoryFormProps {
  category?: Category;
  onSuccess: () => void;
  onCancel: () => void;
}

type CategoryFormData = {
  name: string;
  description: string;
  sort_order: number;
  is_active: boolean;
};

export default function CategoryForm({ category, onSuccess, onCancel }: CategoryFormProps) {
  const [formData, setFormData] = useState<CategoryFormData>({
    name: "",
    description: "",
    sort_order: 1,
    is_active: true,
  });

  const createCategoryMutation = useCreateCategory();
  const updateCategoryMutation = useUpdateCategory();

  const isEditing = !!category;
  const isLoading = createCategoryMutation.isPending || updateCategoryMutation.isPending;

  useEffect(() => {
    if (category) {
      setFormData({
        name: category.name,
        description: category.description || "",
        sort_order: category.sort_order,
        is_active: category.is_active,
      });
    }
  }, [category]);

  const handleChange = (field: keyof CategoryFormData, value: string | number | boolean) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      if (isEditing) {
        await updateCategoryMutation.mutateAsync({
          id: category.id,
          data: {
            name: formData.name,
            description: formData.description,
            sort_order: formData.sort_order,
            is_active: formData.is_active,
          }
        });
        toast.success("Categoria atualizada com sucesso!");
      } else {
        const categoryData: CategoryCreateRequest = {
          name: formData.name,
          description: formData.description,
          sort_order: formData.sort_order,
        };
        await createCategoryMutation.mutateAsync(categoryData);
        toast.success("Categoria criada com sucesso!");
      }
      onSuccess();
    } catch (error: any) {
      toast.error(error.message || "Erro ao salvar categoria");
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="grid gap-4">
        {/* Nome */}
        <div className="space-y-2">
          <Label htmlFor="name">Nome *</Label>
          <Input
            id="name"
            value={formData.name}
            onChange={(e) => handleChange("name", e.target.value)}
            placeholder="Nome da categoria"
            required
          />
        </div>

        {/* Descrição */}
        <div className="space-y-2">
          <Label htmlFor="description">Descrição</Label>
          <Textarea
            id="description"
            value={formData.description}
            onChange={(e) => handleChange("description", e.target.value)}
            placeholder="Descrição da categoria (opcional)"
            rows={3}
          />
        </div>

        {/* Ordem */}
        <div className="space-y-2">
          <Label htmlFor="sort_order">Ordem de Exibição *</Label>
          <Input
            id="sort_order"
            type="number"
            min="1"
            value={formData.sort_order}
            onChange={(e) => handleChange("sort_order", parseInt(e.target.value) || 1)}
            placeholder="1"
            required
          />
          <p className="text-sm text-muted-foreground">
            Ordem em que a categoria aparecerá nas listas (1 = primeira)
          </p>
        </div>

        {/* Status Ativo - Apenas para edição */}
        {isEditing && (
          <div className="flex items-center space-x-2">
            <Switch
              id="is_active"
              checked={formData.is_active}
              onCheckedChange={(checked) => handleChange("is_active", checked)}
            />
            <Label htmlFor="is_active">Categoria ativa</Label>
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex justify-end space-x-2">
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
          disabled={isLoading}
        >
          Cancelar
        </Button>
        <Button type="submit" disabled={isLoading}>
          {isLoading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              {isEditing ? "Atualizando..." : "Criando..."}
            </>
          ) : (
            isEditing ? "Atualizar Categoria" : "Criar Categoria"
          )}
        </Button>
      </div>
    </form>
  );
}
