import { useState, useEffect } from "react";
import { Users, Plus, Search, Phone, MapPin, Mail, Loader2, ChevronLeft, ChevronRight, Edit, MessageCircle, ShoppingBag, Eye } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Skeleton } from "@/components/ui/skeleton";
import { TableSkeleton } from "@/components/ui/loading-skeletons";
import { ErrorState } from "@/components/ui/empty-state";
import { useCustomers, useOrdersByCustomer } from "@/lib/api/hooks";
import { useTenant } from "@/hooks/useTenant";
import { apiClient } from "@/lib/api/client";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";
import { useDebounce } from "../hooks/useDebounce";
import { toast } from "sonner";
import { AddressManager } from "@/components/AddressManager";
import { Customer, Order } from "@/lib/api/types";
import { PhoneNumberInput } from "@/components/ui/phone-input";
import { formatPhoneForDisplay, cleanPhoneForStorage, formatPhoneFromWhatsApp } from "@/lib/phone-utils";
import { useNavigate } from "react-router-dom";

interface CustomerFormData {
  name: string;
  email: string;
  phone: string;
  document: string;
  birth_date: string;
  gender: string;
  notes: string;
  is_active: boolean;
}

export default function Customers() {
  const navigate = useNavigate();
  const [searchTerm, setSearchTerm] = useState("");
  const [page, setPage] = useState(1);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isAddressModalOpen, setIsAddressModalOpen] = useState(false);
  const [isOrdersModalOpen, setIsOrdersModalOpen] = useState(false);
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null);
  const [formData, setFormData] = useState<CustomerFormData>({
    name: "",
    email: "",
    phone: "",
    document: "",
    birth_date: "",
    gender: "",
    notes: "",
    is_active: true
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const limit = 20;

  // Debounce search term to avoid too many API calls
  const debouncedSearchTerm = useDebounce(searchTerm, 500);

  const { data: customerResult, isLoading, error, refetch } = useCustomers({
    limit,
    offset: (page - 1) * limit,
    search: debouncedSearchTerm
  });

  // Hook para pedidos do cliente selecionado
  const { data: customerOrders } = useOrdersByCustomer(
    selectedCustomer?.id || '', 
    { limit: 10 }
  );

  const customers = customerResult?.data || [];
  const totalPages = customerResult?.total_pages || 1;
  const totalItems = customerResult?.total || 0;

  // Reset page when search changes
  useEffect(() => {
    setPage(1);
  }, [debouncedSearchTerm]);

  const handlePreviousPage = () => {
    setPage(prev => Math.max(prev - 1, 1));
  };

  const handleNextPage = () => {
    setPage(prev => Math.min(prev + 1, totalPages));
  };

  const resetForm = () => {
    setFormData({
      name: "",
      email: "",
      phone: "",
      document: "",
      birth_date: "",
      gender: "",
      notes: "",
      is_active: true
    });
  };

  const handleCreateCustomer = async () => {
    setIsSubmitting(true);
    try {
      const processedData = {
        ...formData,
        phone: cleanPhoneForStorage(formData.phone),
        birth_date: formData.birth_date && formData.birth_date.trim() !== '' ? formData.birth_date : null,
        document: formData.document && formData.document.trim() !== '' ? formData.document : null,
        email: formData.email && formData.email.trim() !== '' ? formData.email : null,
        gender: formData.gender && formData.gender.trim() !== '' ? formData.gender : null,
      };
      
      await apiClient.createCustomer(processedData);
      toast.success('Cliente criado com sucesso!');
      setIsCreateModalOpen(false);
      resetForm();
      refetch();
    } catch (error) {
      toast.error('Erro ao criar cliente');
      console.error('Error creating customer:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleEditCustomer = async () => {
    if (!selectedCustomer) return;
    
    setIsSubmitting(true);
    try {
      const processedData = {
        ...formData,
        phone: cleanPhoneForStorage(formData.phone),
        birth_date: formData.birth_date && formData.birth_date.trim() !== '' ? formData.birth_date : null,
        document: formData.document && formData.document.trim() !== '' ? formData.document : null,
        email: formData.email && formData.email.trim() !== '' ? formData.email : null,
        gender: formData.gender && formData.gender.trim() !== '' ? formData.gender : null,
      };
      
      await apiClient.updateCustomer(selectedCustomer.id, processedData);
      toast.success('Cliente atualizado com sucesso!');
      setIsEditModalOpen(false);
      resetForm();
      setSelectedCustomer(null);
      refetch();
    } catch (error) {
      toast.error('Erro ao atualizar cliente');
      console.error('Error updating customer:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  const openEditModal = (customer: Customer) => {
    setSelectedCustomer(customer);
    setFormData({
      name: customer.name || "",
      email: customer.email || "",
      phone: formatPhoneFromWhatsApp(customer.phone || ""),
      document: customer.document || "",
      birth_date: customer.birth_date || "",
      gender: customer.gender || "",
      notes: customer.notes || "",
      is_active: customer.is_active
    });
    setIsEditModalOpen(true);
  };

  const openCreateModal = () => {
    resetForm();
    setIsCreateModalOpen(true);
  };

  // Remove full-page loading and handle errors locally
  const hasError = error;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Users className="w-6 h-6 text-primary" />
            <h1 className="text-2xl font-bold">Clientes</h1>
          </div>
          <Badge variant="secondary" className="text-sm">
            {totalItems} {totalItems === 1 ? 'cliente' : 'clientes'}
          </Badge>
        </div>
        <Button onClick={openCreateModal}>
          <Plus className="w-4 h-4 mr-2" />
          Novo Cliente
        </Button>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Lista de Clientes</CardTitle>
              <CardDescription>
                Gerencie os clientes cadastrados no sistema
              </CardDescription>
            </div>
            <div className="flex items-center gap-4">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="Buscar por nome, telefone ou email..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10 w-80"
                />
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Cliente</TableHead>
                <TableHead>Contato</TableHead>
                <TableHead>Documento</TableHead>
                <TableHead>Data Nascimento</TableHead>
                <TableHead>Criado em</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-40">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {hasError ? (
                <TableRow>
                  <TableCell colSpan={7} className="p-0">
                    <ErrorState
                      message={error?.message}
                      onRetry={() => refetch()}
                    />
                  </TableCell>
                </TableRow>
              ) : isLoading ? (
                <TableSkeleton rows={5} columns={7} includeAvatar />
              ) : (
                customers.map((customer) => (
                <TableRow key={customer.id} className="hover:bg-accent/50">
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <Avatar className="w-10 h-10">
                        <AvatarFallback className="bg-primary text-primary-foreground text-sm">
                          {customer.name ? customer.name.split(' ').map(n => n[0]).join('').toUpperCase() : 'CL'}
                        </AvatarFallback>
                      </Avatar>
                      <div>
                        <div className="font-medium text-foreground">
                          {customer.name || 'Cliente sem nome'}
                        </div>
                        {customer.email && (
                          <div className="flex items-center gap-1 text-sm text-muted-foreground">
                            <Mail className="w-3 h-3" />
                            {customer.email}
                          </div>
                        )}
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1 text-sm">
                      <Phone className="w-4 h-4 text-muted-foreground" />
                      {formatPhoneForDisplay(customer.phone)}
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-muted-foreground">
                      {customer.document || 'Não informado'}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-muted-foreground">
                      {customer.birth_date ? format(new Date(customer.birth_date), 'dd/MM/yyyy', { locale: ptBR }) : 'Não informado'}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-muted-foreground">
                      {format(new Date(customer.created_at), 'dd/MM/yyyy', { locale: ptBR })}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Badge variant={customer.is_active ? 'default' : 'destructive'}>
                      {customer.is_active ? 'Ativo' : 'Bloqueado'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => navigate(`/customers/${customer.id}`)}
                        title="Ver detalhes"
                      >
                        <Eye className="w-4 h-4" />
                      </Button>
                      
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => {
                          navigate(`/whatsapp?customer_id=${customer.id}`);
                        }}
                        title="Ir para conversa"
                      >
                        <MessageCircle className="w-4 h-4" />
                      </Button>
                                           
                      
                      {/* <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => openEditModal(customer)}
                        title="Editar cliente"
                      >
                        <Edit className="w-4 h-4" />
                      </Button> */}
                      
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => {
                          setSelectedCustomer(customer);
                          setIsAddressModalOpen(true);
                        }}
                        title="Gerenciar endereços"
                      >
                        <MapPin className="w-4 h-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              )))}
            </TableBody>
          </Table>
          
          {customers.length === 0 && !isLoading && (
            <div className="text-center py-8">
              <p className="text-muted-foreground">
                {searchTerm ? 'Nenhum cliente encontrado para a busca' : 'Nenhum cliente cadastrado'}
              </p>
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-6">
              <div className="text-sm text-muted-foreground">
                Página {page} de {totalPages} • {totalItems} {totalItems === 1 ? 'item' : 'itens'} no total
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
                  Próxima
                  <ChevronRight className="w-4 h-4" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Customer Modal */}
      <Dialog open={isCreateModalOpen} onOpenChange={setIsCreateModalOpen}>
        <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Novo Cliente</DialogTitle>
            <DialogDescription>
              Preencha as informações do novo cliente.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="name">Nome *</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Nome completo"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="phone">Telefone *</Label>
                <PhoneNumberInput
                  value={formData.phone}
                  onChange={(value) => setFormData({ ...formData, phone: value || "" })}
                  placeholder="Digite o número do telefone"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  placeholder="email@exemplo.com"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="document">Documento</Label>
                <Input
                  id="document"
                  value={formData.document}
                  onChange={(e) => setFormData({ ...formData, document: e.target.value })}
                  placeholder="CPF/CNPJ"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="birth_date">Data de Nascimento</Label>
                <Input
                  id="birth_date"
                  type="date"
                  value={formData.birth_date}
                  onChange={(e) => setFormData({ ...formData, birth_date: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="gender">Gênero</Label>
                <select
                  id="gender"
                  value={formData.gender}
                  onChange={(e) => setFormData({ ...formData, gender: e.target.value })}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="">Selecionar</option>
                  <option value="M">Masculino</option>
                  <option value="F">Feminino</option>
                  <option value="O">Outro</option>
                </select>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="notes">Observações</Label>
              <textarea
                id="notes"
                value={formData.notes}
                onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                placeholder="Observações sobre o cliente..."
                className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
            <div className="flex items-center space-x-2">
              <Switch
                id="is_active"
                checked={formData.is_active}
                onCheckedChange={(checked) => setFormData({ ...formData, is_active: checked })}
              />
              <Label htmlFor="is_active">Cliente ativo</Label>
            </div>
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setIsCreateModalOpen(false)}>
              Cancelar
            </Button>
            <Button onClick={handleCreateCustomer} disabled={isSubmitting}>
              {isSubmitting ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Criando...
                </>
              ) : (
                'Criar Cliente'
              )}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Edit Customer Modal */}
      <Dialog open={isEditModalOpen} onOpenChange={setIsEditModalOpen}>
        <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Editar Cliente</DialogTitle>
            <DialogDescription>
              Atualize as informações do cliente.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="edit_name">Nome *</Label>
                <Input
                  id="edit_name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Nome completo"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit_phone">Telefone *</Label>
                <PhoneNumberInput
                  value={formData.phone}
                  onChange={(value) => setFormData({ ...formData, phone: value || "" })}
                  placeholder="Digite o número do telefone"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="edit_email">Email</Label>
                <Input
                  id="edit_email"
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  placeholder="email@exemplo.com"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit_document">Documento</Label>
                <Input
                  id="edit_document"
                  value={formData.document}
                  onChange={(e) => setFormData({ ...formData, document: e.target.value })}
                  placeholder="CPF/CNPJ"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="edit_birth_date">Data de Nascimento</Label>
                <Input
                  id="edit_birth_date"
                  type="date"
                  value={formData.birth_date}
                  onChange={(e) => setFormData({ ...formData, birth_date: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit_gender">Gênero</Label>
                <select
                  id="edit_gender"
                  value={formData.gender}
                  onChange={(e) => setFormData({ ...formData, gender: e.target.value })}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="">Selecionar</option>
                  <option value="M">Masculino</option>
                  <option value="F">Feminino</option>
                  <option value="O">Outro</option>
                </select>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit_notes">Observações</Label>
              <textarea
                id="edit_notes"
                value={formData.notes}
                onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                placeholder="Observações sobre o cliente..."
                className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
            <div className="flex items-center space-x-2">
              <Switch
                id="edit_is_active"
                checked={formData.is_active}
                onCheckedChange={(checked) => setFormData({ ...formData, is_active: checked })}
              />
              <Label htmlFor="edit_is_active">
                Cliente {formData.is_active ? 'ativo' : 'bloqueado'}
              </Label>
            </div>
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setIsEditModalOpen(false)}>
              Cancelar
            </Button>
            <Button onClick={handleEditCustomer} disabled={isSubmitting}>
              {isSubmitting ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Salvando...
                </>
              ) : (
                'Salvar Alterações'
              )}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Orders Modal */}
      <Dialog open={isOrdersModalOpen} onOpenChange={setIsOrdersModalOpen}>
        <DialogContent className="sm:max-w-[800px] max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              Pedidos de {selectedCustomer?.name || 'Cliente'}
            </DialogTitle>
            <DialogDescription>
              Lista de pedidos realizados pelo cliente
            </DialogDescription>
          </DialogHeader>
          <div className="max-h-96 overflow-y-auto">
            {customerOrders?.data && customerOrders.data.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Pedido</TableHead>
                    <TableHead>Data</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Total</TableHead>
                    <TableHead>Ações</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {customerOrders.data.map((order: any) => (
                    <TableRow key={order.id}>
                      <TableCell>
                        <div className="font-medium">#{order.order_number || order.id.substring(0, 8)}</div>
                      </TableCell>
                      <TableCell>
                        {format(new Date(order.created_at), 'dd/MM/yyyy', { locale: ptBR })}
                      </TableCell>
                      <TableCell>
                        <Badge variant={order.status === 'completed' ? 'default' : 'secondary'}>
                          {order.status === 'completed' ? 'Concluído' : 
                           order.status === 'processing' ? 'Processando' :
                           order.status === 'pending' ? 'Pendente' : order.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        R$ {parseFloat(order.total_amount || '0').toFixed(2).replace('.', ',')}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            navigate(`/sales/orders/${order.id}`);
                            setIsOrdersModalOpen(false);
                          }}
                        >
                          Ver Detalhes
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <div className="text-center py-8">
                <ShoppingBag className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p className="text-muted-foreground">
                  Este cliente ainda não possui pedidos
                </p>
              </div>
            )}
          </div>
          <div className="flex justify-end">
            <Button variant="outline" onClick={() => setIsOrdersModalOpen(false)}>
              Fechar
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Address Manager Modal */}
      {selectedCustomer && (
        <AddressManager 
          customer={selectedCustomer} 
          isOpen={isAddressModalOpen} 
          onClose={() => setIsAddressModalOpen(false)} 
        />
      )}
    </div>
  );
}
