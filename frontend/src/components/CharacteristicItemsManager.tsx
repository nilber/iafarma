import { useState } from "react";
import { Plus, Edit, Trash2, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { useCharacteristicItems, useCreateCharacteristicItem, useUpdateCharacteristicItem, useDeleteCharacteristicItem } from "@/lib/api/hooks";
import { CharacteristicItemCreateRequest, CharacteristicItemUpdateRequest } from "@/lib/api/types";
import { toast } from "sonner";

interface CharacteristicItemsManagerProps {
  characteristicId: string;
}

interface EditingItem {
  id: string;
  name: string;
  price: string;
}

export default function CharacteristicItemsManager({ characteristicId }: CharacteristicItemsManagerProps) {
  const [editingItem, setEditingItem] = useState<EditingItem | null>(null);
  const [newItem, setNewItem] = useState({
    name: "",
    price: "0.00",
  });
  const [showCreateDialog, setShowCreateDialog] = useState(false);

  const { data: items = [], isLoading, refetch } = useCharacteristicItems(characteristicId);
  const createItemMutation = useCreateCharacteristicItem();
  const updateItemMutation = useUpdateCharacteristicItem();
  const deleteItemMutation = useDeleteCharacteristicItem();

  const handleCreateItem = async () => {
    if (!newItem.name.trim()) {
      toast.error("Nome do item é obrigatório");
      return;
    }

    try {
      const request: CharacteristicItemCreateRequest = {
        name: newItem.name.trim(),
        price: newItem.price,
      };

      await createItemMutation.mutateAsync({
        characteristicId,
        item: request,
      });

      setNewItem({ name: "", price: "0.00" });
      setShowCreateDialog(false);
      refetch();
      toast.success("Item criado com sucesso!");
    } catch (error) {
      console.error("Erro ao criar item:", error);
      toast.error("Erro ao criar item");
    }
  };

  const handleUpdateItem = async () => {
    if (!editingItem || !editingItem.name.trim()) {
      toast.error("Nome do item é obrigatório");
      return;
    }

    try {
      const request: CharacteristicItemUpdateRequest = {
        name: editingItem.name.trim(),
        price: editingItem.price,
      };

      await updateItemMutation.mutateAsync({
        characteristicId,
        id: editingItem.id,
        item: request,
      });

      setEditingItem(null);
      refetch();
      toast.success("Item atualizado com sucesso!");
    } catch (error) {
      console.error("Erro ao atualizar item:", error);
      toast.error("Erro ao atualizar item");
    }
  };

  const handleDeleteItem = async (id: string, name: string) => {
    if (!confirm(`Tem certeza que deseja excluir o item "${name}"?`)) {
      return;
    }

    try {
      await deleteItemMutation.mutateAsync({ characteristicId, id });
      refetch();
      toast.success("Item excluído com sucesso!");
    } catch (error) {
      console.error("Erro ao excluir item:", error);
      toast.error("Erro ao excluir item");
    }
  };

  const formatPrice = (price: string) => {
    const numPrice = parseFloat(price || "0");
    return `R$ ${numPrice.toFixed(2).replace(".", ",")}`;
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="w-4 h-4 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium text-muted-foreground">
          Itens da Característica ({items.length})
        </h4>
        <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
          <DialogTrigger asChild>
            <Button size="sm" variant="outline">
              <Plus className="w-3 h-3 mr-1" />
              Adicionar Item
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Novo Item</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div>
                <Label htmlFor="item-name">Nome do Item</Label>
                <Input
                  id="item-name"
                  value={newItem.name}
                  onChange={(e) => setNewItem(prev => ({
                    ...prev,
                    name: e.target.value
                  }))}
                  placeholder="Ex: Pequena (25cm), Média (30cm)..."
                />
              </div>
              
              <div>
                <Label htmlFor="item-price">Preço Adicional</Label>
                <div className="relative">
                  <span className="absolute left-3 top-1/2 transform -translate-y-1/2 text-sm text-muted-foreground">
                    R$
                  </span>
                  <Input
                    id="item-price"
                    type="number"
                    step="0.01"
                    min="0"
                    value={newItem.price}
                    onChange={(e) => setNewItem(prev => ({
                      ...prev,
                      price: e.target.value
                    }))}
                    className="pl-10"
                    placeholder="0.00"
                  />
                </div>
              </div>

              <div className="flex justify-end gap-2">
                <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
                  Cancelar
                </Button>
                <Button 
                  onClick={handleCreateItem}
                  disabled={createItemMutation.isPending}
                >
                  {createItemMutation.isPending && (
                    <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                  )}
                  Criar
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {items.length === 0 ? (
        <Card className="border-dashed">
          <CardContent className="text-center py-6">
            <p className="text-sm text-muted-foreground mb-3">
              Nenhum item adicionado ainda
            </p>
            <Button size="sm" onClick={() => setShowCreateDialog(true)}>
              <Plus className="w-3 h-3 mr-1" />
              Adicionar Primeiro Item
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {items.map((item, index) => (
            <div
              key={item.id}
              className="flex items-center justify-between p-3 bg-muted/50 rounded-lg"
            >
              <div className="flex-1">
                <div className="flex items-center justify-between">
                  <span className="font-medium">{item.name}</span>
                  <span className="text-sm font-medium text-green-600">
                    {formatPrice(item.price)}
                  </span>
                </div>
              </div>
              
              <div className="flex items-center gap-1 ml-3">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setEditingItem({
                    id: item.id,
                    name: item.name,
                    price: item.price
                  })}
                >
                  <Edit className="w-3 h-3" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDeleteItem(item.id, item.name)}
                  disabled={deleteItemMutation.isPending}
                >
                  <Trash2 className="w-3 h-3" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Edit Item Dialog */}
      <Dialog open={!!editingItem} onOpenChange={(open) => !open && setEditingItem(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Editar Item</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="edit-item-name">Nome do Item</Label>
              <Input
                id="edit-item-name"
                value={editingItem?.name || ""}
                onChange={(e) => setEditingItem(prev => 
                  prev ? { ...prev, name: e.target.value } : null
                )}
              />
            </div>

            <div>
              <Label htmlFor="edit-item-price">Preço Adicional</Label>
              <div className="relative">
                <span className="absolute left-3 top-1/2 transform -translate-y-1/2 text-sm text-muted-foreground">
                  R$
                </span>
                <Input
                  id="edit-item-price"
                  type="number"
                  step="0.01"
                  min="0"
                  value={editingItem?.price || "0.00"}
                  onChange={(e) => setEditingItem(prev => 
                    prev ? { ...prev, price: e.target.value } : null
                  )}
                  className="pl-10"
                />
              </div>
            </div>

            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setEditingItem(null)}>
                Cancelar
              </Button>
              <Button 
                onClick={handleUpdateItem}
                disabled={updateItemMutation.isPending}
              >
                {updateItemMutation.isPending && (
                  <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                )}
                Salvar
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}