import { useState } from "react";
import { Plus, Edit, Trash2, MapPin, Star, Loader2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { 
  useAddressesByCustomer, 
  useCreateAddress, 
  useUpdateAddress, 
  useDeleteAddress,
  useSetDefaultAddress 
} from "@/lib/api/hooks";
import { Address, CreateAddressRequest, UpdateAddressRequest, Customer } from "@/lib/api/types";
import { toast } from "sonner";

interface AddressManagerProps {
  customer: Customer;
  isOpen: boolean;
  onClose: () => void;
}

interface AddressFormData {
  label: string;
  street: string;
  number: string;
  complement: string;
  neighborhood: string;
  city: string;
  state: string;
  zip_code: string;
  country: string;
  is_default: boolean;
}

export function AddressManager({ customer, isOpen, onClose }: AddressManagerProps) {
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [selectedAddress, setSelectedAddress] = useState<Address | null>(null);
  const [formData, setFormData] = useState<AddressFormData>({
    label: "",
    street: "",
    number: "",
    complement: "",
    neighborhood: "",
    city: "",
    state: "",
    zip_code: "",
    country: "BR",
    is_default: false
  });

  const { data: addressData, isLoading } = useAddressesByCustomer(customer.id);
  const createAddressMutation = useCreateAddress();
  const updateAddressMutation = useUpdateAddress();
  const deleteAddressMutation = useDeleteAddress();
  const setDefaultAddressMutation = useSetDefaultAddress();

  const addresses = addressData?.addresses || [];

  const resetForm = () => {
    setFormData({
      label: "",
      street: "",
      number: "",
      complement: "",
      neighborhood: "",
      city: "",
      state: "",
      zip_code: "",
      country: "BR",
      is_default: false
    });
  };

  const handleCreateAddress = async () => {
    try {
      const addressData: CreateAddressRequest = {
        customer_id: customer.id,
        ...formData
      };

      await createAddressMutation.mutateAsync(addressData);
      toast.success("Endereço criado com sucesso!");
      setIsCreateModalOpen(false);
      resetForm();
    } catch (error) {
      toast.error("Erro ao criar endereço");
    }
  };

  const handleUpdateAddress = async () => {
    if (!selectedAddress) return;

    try {
      const updateData: UpdateAddressRequest = { ...formData };
      
      await updateAddressMutation.mutateAsync({
        id: selectedAddress.id,
        address: updateData
      });
      
      toast.success("Endereço atualizado com sucesso!");
      setIsEditModalOpen(false);
      setSelectedAddress(null);
      resetForm();
    } catch (error) {
      toast.error("Erro ao atualizar endereço");
    }
  };

  const handleDeleteAddress = async (address: Address) => {
    if (!confirm("Tem certeza que deseja excluir este endereço?")) return;

    try {
      await deleteAddressMutation.mutateAsync({
        id: address.id,
        customerId: customer.id
      });
      toast.success("Endereço excluído com sucesso!");
    } catch (error) {
      toast.error("Erro ao excluir endereço");
    }
  };

  const handleSetDefault = async (address: Address) => {
    try {
      await setDefaultAddressMutation.mutateAsync({
        id: address.id,
        customerId: customer.id
      });
      toast.success("Endereço definido como padrão!");
    } catch (error) {
      toast.error("Erro ao definir endereço como padrão");
    }
  };

  const openEditModal = (address: Address) => {
    setSelectedAddress(address);
    setFormData({
      label: address.label || "",
      street: address.street,
      number: address.number || "",
      complement: address.complement || "",
      neighborhood: address.neighborhood || "",
      city: address.city,
      state: address.state,
      zip_code: address.zip_code,
      country: address.country || "BR",
      is_default: address.is_default
    });
    setIsEditModalOpen(true);
  };

  const openCreateModal = () => {
    resetForm();
    setIsCreateModalOpen(true);
  };

  return (
    <>
      <Dialog open={isOpen} onOpenChange={onClose}>
        <DialogContent className="sm:max-w-[800px] max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <MapPin className="w-5 h-5" />
              Endereços de {customer.name}
            </DialogTitle>
            <DialogDescription>
              Gerencie os endereços do cliente
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-medium">Endereços Cadastrados</h3>
              <Button onClick={openCreateModal}>
                <Plus className="w-4 h-4 mr-2" />
                Novo Endereço
              </Button>
            </div>

            {isLoading ? (
              <div className="flex justify-center py-8">
                <Loader2 className="w-6 h-6 animate-spin" />
              </div>
            ) : addresses.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                <MapPin className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p>Nenhum endereço cadastrado</p>
                <Button className="mt-4" onClick={openCreateModal}>
                  Adicionar primeiro endereço
                </Button>
              </div>
            ) : (
              <div className="grid gap-4">
                {addresses.map((address) => (
                  <Card key={address.id} className={address.is_default ? "ring-2 ring-primary" : ""}>
                    <CardContent className="p-4">
                      <div className="flex justify-between items-start">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-2">
                            {address.label && (
                              <Badge variant="outline">{address.label}</Badge>
                            )}
                            {address.is_default && (
                              <Badge className="bg-yellow-100 text-yellow-800 hover:bg-yellow-100">
                                <Star className="w-3 h-3 mr-1" />
                                Padrão
                              </Badge>
                            )}
                          </div>
                          <div className="text-sm space-y-1">
                            <p className="font-medium">
                              {address.street}, {address.number}
                              {address.complement && ` - ${address.complement}`}
                            </p>
                            <p className="text-muted-foreground">
                              {address.neighborhood && `${address.neighborhood}, `}
                              {address.city} - {address.state}
                            </p>
                            <p className="text-muted-foreground">
                              CEP: {address.zip_code}
                            </p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          {!address.is_default && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleSetDefault(address)}
                              disabled={setDefaultAddressMutation.isPending}
                            >
                              <Star className="w-4 h-4" />
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => openEditModal(address)}
                          >
                            <Edit className="w-4 h-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDeleteAddress(address)}
                            disabled={deleteAddressMutation.isPending}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Create Address Modal */}
      <Dialog open={isCreateModalOpen} onOpenChange={setIsCreateModalOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Novo Endereço</DialogTitle>
            <DialogDescription>
              Adicione um novo endereço para {customer.name}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="label">Rótulo (opcional)</Label>
              <Input
                id="label"
                value={formData.label}
                onChange={(e) => setFormData({ ...formData, label: e.target.value })}
                placeholder="Casa, Trabalho, etc."
              />
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div className="col-span-2 space-y-2">
                <Label htmlFor="street">Rua *</Label>
                <Input
                  id="street"
                  value={formData.street}
                  onChange={(e) => setFormData({ ...formData, street: e.target.value })}
                  placeholder="Nome da rua"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="number">Número</Label>
                <Input
                  id="number"
                  value={formData.number}
                  onChange={(e) => setFormData({ ...formData, number: e.target.value })}
                  placeholder="123"
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="complement">Complemento</Label>
              <Input
                id="complement"
                value={formData.complement}
                onChange={(e) => setFormData({ ...formData, complement: e.target.value })}
                placeholder="Apto 101, Bloco A, etc."
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="neighborhood">Bairro</Label>
                <Input
                  id="neighborhood"
                  value={formData.neighborhood}
                  onChange={(e) => setFormData({ ...formData, neighborhood: e.target.value })}
                  placeholder="Nome do bairro"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="zip_code">CEP *</Label>
                <Input
                  id="zip_code"
                  value={formData.zip_code}
                  onChange={(e) => setFormData({ ...formData, zip_code: e.target.value })}
                  placeholder="00000-000"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="city">Cidade *</Label>
                <Input
                  id="city"
                  value={formData.city}
                  onChange={(e) => setFormData({ ...formData, city: e.target.value })}
                  placeholder="Nome da cidade"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="state">Estado *</Label>
                <Input
                  id="state"
                  value={formData.state}
                  onChange={(e) => setFormData({ ...formData, state: e.target.value })}
                  placeholder="SP"
                />
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="is_default"
                checked={formData.is_default}
                onChange={(e) => setFormData({ ...formData, is_default: e.target.checked })}
                className="h-4 w-4"
              />
              <Label htmlFor="is_default">Definir como endereço padrão</Label>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setIsCreateModalOpen(false)}>
              Cancelar
            </Button>
            <Button 
              onClick={handleCreateAddress} 
              disabled={createAddressMutation.isPending || !formData.street || !formData.city || !formData.state || !formData.zip_code}
            >
              {createAddressMutation.isPending && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              Criar Endereço
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Edit Address Modal */}
      <Dialog open={isEditModalOpen} onOpenChange={setIsEditModalOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Editar Endereço</DialogTitle>
            <DialogDescription>
              Edite o endereço de {customer.name}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="edit-label">Rótulo (opcional)</Label>
              <Input
                id="edit-label"
                value={formData.label}
                onChange={(e) => setFormData({ ...formData, label: e.target.value })}
                placeholder="Casa, Trabalho, etc."
              />
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div className="col-span-2 space-y-2">
                <Label htmlFor="edit-street">Rua *</Label>
                <Input
                  id="edit-street"
                  value={formData.street}
                  onChange={(e) => setFormData({ ...formData, street: e.target.value })}
                  placeholder="Nome da rua"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-number">Número</Label>
                <Input
                  id="edit-number"
                  value={formData.number}
                  onChange={(e) => setFormData({ ...formData, number: e.target.value })}
                  placeholder="123"
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-complement">Complemento</Label>
              <Input
                id="edit-complement"
                value={formData.complement}
                onChange={(e) => setFormData({ ...formData, complement: e.target.value })}
                placeholder="Apto 101, Bloco A, etc."
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="edit-neighborhood">Bairro</Label>
                <Input
                  id="edit-neighborhood"
                  value={formData.neighborhood}
                  onChange={(e) => setFormData({ ...formData, neighborhood: e.target.value })}
                  placeholder="Nome do bairro"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-zip_code">CEP *</Label>
                <Input
                  id="edit-zip_code"
                  value={formData.zip_code}
                  onChange={(e) => setFormData({ ...formData, zip_code: e.target.value })}
                  placeholder="00000-000"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="edit-city">Cidade *</Label>
                <Input
                  id="edit-city"
                  value={formData.city}
                  onChange={(e) => setFormData({ ...formData, city: e.target.value })}
                  placeholder="Nome da cidade"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-state">Estado *</Label>
                <Input
                  id="edit-state"
                  value={formData.state}
                  onChange={(e) => setFormData({ ...formData, state: e.target.value })}
                  placeholder="SP"
                />
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="edit-is_default"
                checked={formData.is_default}
                onChange={(e) => setFormData({ ...formData, is_default: e.target.checked })}
                className="h-4 w-4"
              />
              <Label htmlFor="edit-is_default">Definir como endereço padrão</Label>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setIsEditModalOpen(false)}>
              Cancelar
            </Button>
            <Button 
              onClick={handleUpdateAddress} 
              disabled={updateAddressMutation.isPending || !formData.street || !formData.city || !formData.state || !formData.zip_code}
            >
              {updateAddressMutation.isPending && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              Atualizar Endereço
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
