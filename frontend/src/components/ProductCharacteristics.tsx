import { useState } from "react";
import { Plus, Edit, Trash2, Loader2, ChevronDown, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import { useProductCharacteristics, useCreateProductCharacteristic, useUpdateProductCharacteristic, useDeleteProductCharacteristic, useCharacteristicItems, useCreateCharacteristicItem, useUpdateCharacteristicItem, useDeleteCharacteristicItem } from "@/lib/api/hooks";
import { ProductCharacteristicCreateRequest, ProductCharacteristicUpdateRequest, CharacteristicItemCreateRequest, CharacteristicItemUpdateRequest } from "@/lib/api/types";
import { toast } from "sonner";
import CharacteristicItemsManager from "./CharacteristicItemsManager";

interface ProductCharacteristicsProps {
  productId: string;
}

interface EditingCharacteristic {
  id: string;
  title: string;
  is_required: boolean;
  is_multiple_choice: boolean;
}

interface CharacteristicItem {
  id?: string;
  name: string;
  price: string;
}

export default function ProductCharacteristics({ productId }: ProductCharacteristicsProps) {
  const [editingCharacteristic, setEditingCharacteristic] = useState<EditingCharacteristic | null>(null);
  const [newCharacteristic, setNewCharacteristic] = useState({
    title: "",
    is_required: false,
    is_multiple_choice: false,
  });
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [expandedCharacteristic, setExpandedCharacteristic] = useState<string | null>(null);
  
  // Estados para gerenciar itens nos modais
  const [newCharacteristicItems, setNewCharacteristicItems] = useState<CharacteristicItem[]>([]);
  const [editCharacteristicItems, setEditCharacteristicItems] = useState<CharacteristicItem[]>([]);
  const [editingItem, setEditingItem] = useState<{ index: number; item: CharacteristicItem } | null>(null);

  const { data: characteristics = [], isLoading, refetch } = useProductCharacteristics(productId);
  const createCharacteristicMutation = useCreateProductCharacteristic();
  const updateCharacteristicMutation = useUpdateProductCharacteristic();
  const deleteCharacteristicMutation = useDeleteProductCharacteristic();
  const createItemMutation = useCreateCharacteristicItem();
  const updateItemMutation = useUpdateCharacteristicItem();
  const deleteItemMutation = useDeleteCharacteristicItem();

  const handleCreateCharacteristic = async () => {
    if (!newCharacteristic.title.trim()) {
      toast.error("Título da característica é obrigatório");
      return;
    }

    try {
      const request: ProductCharacteristicCreateRequest = {
        title: newCharacteristic.title.trim(),
        is_required: newCharacteristic.is_required,
        is_multiple_choice: newCharacteristic.is_multiple_choice,
      };

      const createdChar = await createCharacteristicMutation.mutateAsync({
        productId,
        characteristic: request,
      });

      // Criar itens se existirem
      if (newCharacteristicItems.length > 0) {
        for (const item of newCharacteristicItems) {
          await createItemMutation.mutateAsync({
            characteristicId: createdChar.id,
            item: {
              name: item.name,
              price: item.price,
            }
          });
        }
      }

      setNewCharacteristic({ title: "", is_required: false, is_multiple_choice: false });
      setNewCharacteristicItems([]);
      setShowCreateDialog(false);
      refetch();
      toast.success("Característica criada com sucesso!");
    } catch (error) {
      console.error("Erro ao criar característica:", error);
      toast.error("Erro ao criar característica");
    }
  };

  const handleUpdateCharacteristic = async () => {
    if (!editingCharacteristic || !editingCharacteristic.title.trim()) {
      toast.error("Título da característica é obrigatório");
      return;
    }

    try {
      const request: ProductCharacteristicUpdateRequest = {
        title: editingCharacteristic.title.trim(),
        is_required: editingCharacteristic.is_required,
        is_multiple_choice: editingCharacteristic.is_multiple_choice,
      };

      await updateCharacteristicMutation.mutateAsync({
        productId,
        id: editingCharacteristic.id,
        characteristic: request,
      });

      // Atualizar itens se necessário
      // (Para simplicidade, vamos apenas atualizar a característica por enquanto)
      // Em uma implementação mais complexa, poderíamos fazer diff dos itens

      setEditingCharacteristic(null);
      setEditCharacteristicItems([]);
      refetch();
      toast.success("Característica atualizada com sucesso!");
    } catch (error) {
      console.error("Erro ao atualizar característica:", error);
      toast.error("Erro ao atualizar característica");
    }
  };

  const handleDeleteCharacteristic = async (id: string, title: string) => {
    if (!confirm(`Tem certeza que deseja excluir a característica "${title}"?`)) {
      return;
    }

    try {
      await deleteCharacteristicMutation.mutateAsync({ productId, id });
      refetch();
      toast.success("Característica excluída com sucesso!");
    } catch (error) {
      console.error("Erro ao excluir característica:", error);
      toast.error("Erro ao excluir característica");
    }
  };

  // Funções para gerenciar itens nos modais
  const addNewItem = () => {
    setNewCharacteristicItems([...newCharacteristicItems, { name: "", price: "0.00" }]);
  };

  const updateNewItem = (index: number, field: 'name' | 'price', value: string) => {
    const updated = [...newCharacteristicItems];
    updated[index][field] = value;
    setNewCharacteristicItems(updated);
  };

  const removeNewItem = (index: number) => {
    setNewCharacteristicItems(newCharacteristicItems.filter((_, i) => i !== index));
  };

  const startEditCharacteristic = (characteristic: any) => {
    setEditingCharacteristic({
      id: characteristic.id,
      title: characteristic.title,
      is_required: characteristic.is_required,
      is_multiple_choice: characteristic.is_multiple_choice
    });
    
    // Carregar itens existentes
    setEditCharacteristicItems(
      characteristic.items?.map((item: any) => ({
        id: item.id,
        name: item.name,
        price: item.price
      })) || []
    );
  };

  const formatPrice = (price: string) => {
    const numPrice = parseFloat(price || "0");
    return `R$ ${numPrice.toFixed(2).replace(".", ",")}`;
  };

  const renderItemsList = (items: CharacteristicItem[], updateItem: (index: number, field: 'name' | 'price', value: string) => void, removeItem: (index: number) => void, addItem: () => void) => (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label className="text-sm font-medium">Itens da Característica</Label>
        <Button type="button" size="sm" variant="outline" onClick={addItem}>
          <Plus className="w-3 h-3 mr-1" />
          Adicionar Item
        </Button>
      </div>
      
      {items.length > 0 && (
        <div className="space-y-2 max-h-48 overflow-y-auto">
          {items.map((item, index) => (
            <div key={index} className="flex gap-2 items-end p-2 border rounded">
              <div className="flex-1">
                <Label className="text-xs">Nome do Item</Label>
                <Input
                  value={item.name}
                  onChange={(e) => updateItem(index, 'name', e.target.value)}
                  placeholder="Ex: Pequena (25cm)"
                />
              </div>
              <div className="w-24">
                <Label className="text-xs">Preço</Label>
                <div className="relative">
                  <span className="absolute left-2 top-1/2 transform -translate-y-1/2 text-xs text-muted-foreground">R$</span>
                  <Input
                    type="number"
                    step="0.01"
                    min="0"
                    value={item.price}
                    onChange={(e) => updateItem(index, 'price', e.target.value)}
                    className="pl-8"
                  />
                </div>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => removeItem(index)}
              >
                <Trash2 className="w-3 h-3" />
              </Button>
            </div>
          ))}
        </div>
      )}
      
      {items.length === 0 && (
        <div className="text-center py-4 text-sm text-muted-foreground border-2 border-dashed rounded">
          Nenhum item adicionado. Clique em "Adicionar Item" para começar.
        </div>
      )}
    </div>
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="w-6 h-6 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Características do Produto</h3>
        <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="w-4 h-4 mr-2" />
              Adicionar Característica
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Nova Característica</DialogTitle>
            </DialogHeader>
            <div className="space-y-6">
              <div>
                <Label htmlFor="characteristic-title">Título da Característica</Label>
                <Input
                  id="characteristic-title"
                  value={newCharacteristic.title}
                  onChange={(e) => setNewCharacteristic(prev => ({
                    ...prev,
                    title: e.target.value
                  }))}
                  placeholder="Ex: Cor, Tamanho, Material..."
                />
              </div>
              
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="is-required"
                  checked={newCharacteristic.is_required}
                  onCheckedChange={(checked) => setNewCharacteristic(prev => ({
                    ...prev,
                    is_required: checked === true
                  }))}
                />
                <Label htmlFor="is-required">Característica obrigatória</Label>
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="is-multiple"
                  checked={newCharacteristic.is_multiple_choice}
                  onCheckedChange={(checked) => setNewCharacteristic(prev => ({
                    ...prev,
                    is_multiple_choice: checked === true
                  }))}
                />
                <Label htmlFor="is-multiple">Permite múltipla escolha</Label>
              </div>

              <Separator />
              
              {renderItemsList(
                newCharacteristicItems,
                updateNewItem,
                removeNewItem,
                addNewItem
              )}

              <div className="flex justify-end gap-2 pt-4">
                <Button variant="outline" onClick={() => {
                  setShowCreateDialog(false);
                  setNewCharacteristicItems([]);
                }}>
                  Cancelar
                </Button>
                <Button 
                  onClick={handleCreateCharacteristic}
                  disabled={createCharacteristicMutation.isPending}
                >
                  {createCharacteristicMutation.isPending && (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  )}
                  Criar
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {characteristics.length === 0 ? (
        <Card>
          <CardContent className="text-center py-8">
            <p className="text-muted-foreground mb-4">
              Nenhuma característica adicionada ainda
            </p>
            <Button onClick={() => setShowCreateDialog(true)}>
              <Plus className="w-4 h-4 mr-2" />
              Adicionar Primeira Característica
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {characteristics.map((characteristic) => (
            <Card key={characteristic.id}>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="text-base">{characteristic.title}</CardTitle>
                    <div className="flex gap-2 mt-1">
                      {characteristic.is_required && (
                        <span className="text-xs bg-red-100 text-red-800 px-2 py-1 rounded">
                          Obrigatório
                        </span>
                      )}
                      {characteristic.is_multiple_choice && (
                        <span className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded">
                          Múltipla escolha
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => startEditCharacteristic(characteristic)}
                    >
                      <Edit className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDeleteCharacteristic(characteristic.id, characteristic.title)}
                      disabled={deleteCharacteristicMutation.isPending}
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setExpandedCharacteristic(
                        expandedCharacteristic === characteristic.id ? null : characteristic.id
                      )}
                    >
                      {expandedCharacteristic === characteristic.id ? (
                        <ChevronDown className="w-4 h-4 mr-1" />
                      ) : (
                        <ChevronRight className="w-4 h-4 mr-1" />
                      )}
                      {expandedCharacteristic === characteristic.id ? "Ocultar" : "Gerenciar"} Itens
                    </Button>
                  </div>
                </div>
              </CardHeader>
              
              {expandedCharacteristic === characteristic.id && (
                <CardContent className="pt-0">
                  <CharacteristicItemsManager characteristicId={characteristic.id} />
                </CardContent>
              )}
              
              {characteristic.items && characteristic.items.length > 0 && expandedCharacteristic !== characteristic.id && (
                <CardContent className="pt-0">
                  <div className="text-sm text-muted-foreground mb-2">Itens:</div>
                  <div className="grid gap-2">
                    {characteristic.items.map((item) => (
                      <div key={item.id} className="flex justify-between items-center p-2 bg-muted rounded">
                        <span>{item.name}</span>
                        {item.price && (
                          <span className="text-sm font-medium">
                            R$ {parseFloat(item.price).toFixed(2)}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                </CardContent>
              )}
            </Card>
          ))}
        </div>
      )}

      {/* Edit Characteristic Dialog */}
      <Dialog open={!!editingCharacteristic} onOpenChange={(open) => {
        if (!open) {
          setEditingCharacteristic(null);
          setEditCharacteristicItems([]);
        }
      }}>
        <DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Editar Característica</DialogTitle>
          </DialogHeader>
          <div className="space-y-6">
            <div>
              <Label htmlFor="edit-characteristic-title">Título da Característica</Label>
              <Input
                id="edit-characteristic-title"
                value={editingCharacteristic?.title || ""}
                onChange={(e) => setEditingCharacteristic(prev => 
                  prev ? { ...prev, title: e.target.value } : null
                )}
              />
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="edit-is-required"
                checked={editingCharacteristic?.is_required || false}
                onCheckedChange={(checked) => setEditingCharacteristic(prev => 
                  prev ? { ...prev, is_required: checked === true } : null
                )}
              />
              <Label htmlFor="edit-is-required">Característica obrigatória</Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="edit-is-multiple"
                checked={editingCharacteristic?.is_multiple_choice || false}
                onCheckedChange={(checked) => setEditingCharacteristic(prev => 
                  prev ? { ...prev, is_multiple_choice: checked === true } : null
                )}
              />
              <Label htmlFor="edit-is-multiple">Permite múltipla escolha</Label>
            </div>

            <Separator />

            <div>
              <p className="text-sm text-muted-foreground mb-4">
                Para gerenciar os itens desta característica, utilize o botão "Gerenciar Itens" após salvar.
              </p>
            </div>

            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={() => {
                setEditingCharacteristic(null);
                setEditCharacteristicItems([]);
              }}>
                Cancelar
              </Button>
              <Button 
                onClick={handleUpdateCharacteristic}
                disabled={updateCharacteristicMutation.isPending}
              >
                {updateCharacteristicMutation.isPending && (
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
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