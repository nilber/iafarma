import { useState, useEffect } from "react";
import { Users, Plus, Search, Phone, MapPin, Mail, Loader2, ChevronLeft, ChevronRight, Edit, Eye, MessageCircle, ShoppingBag, ToggleLeft, ToggleRight } from "lucide-react";
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
import { useCustomers, useOrdersByCustomer } from "@/lib/api/hooks";
import { PhoneNumberInput } from "@/components/ui/phone-input";
import { formatPhoneForDisplay, cleanPhoneForStorage, formatPhoneFromWhatsApp } from "@/lib/phone-utils";
import { apiClient } from "@/lib/api/client";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";
import { useDebounce } from "../hooks/useDebounce";
import { toast } from "sonner";
import { AddressManager } from "@/components/AddressManager";
import { Customer } from "@/lib/api/types";

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
  const [searchTerm, setSearchTerm] = useState("");
  const [page, setPage] = useState(1);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isAddressModalOpen, setIsAddressModalOpen] = useState(false);
  const [isOrdersModalOpen, setIsOrdersModalOpen] = useState(false);
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null);
  const [formData, setFormData] = useState({
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
      // Remove tenant_id from formData as it will be automatically added by the backend
      // Convert empty strings to null for date fields
      const { ...customerData } = formData;
      
      // Process data to handle empty strings
      const processedData = {
        ...customerData,
        phone: cleanPhoneForStorage(customerData.phone), // Limpa o telefone para salvar
        birth_date: customerData.birth_date && customerData.birth_date.trim() !== '' ? customerData.birth_date : null,
        document: customerData.document && customerData.document.trim() !== '' ? customerData.document : null,
        email: customerData.email && customerData.email.trim() !== '' ? customerData.email : null,
        gender: customerData.gender && customerData.gender.trim() !== '' ? customerData.gender : null,
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
      // Process data to handle empty strings
      const processedData = {
        ...formData,
        phone: cleanPhoneForStorage(formData.phone), // Limpa o telefone para salvar
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
      phone: formatPhoneFromWhatsApp(customer.phone || ""), // Formata telefone para edição
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

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-96">
        <Loader2 className="w-8 h-8 animate-spin" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <p className="text-muted-foreground">Erro ao carregar clientes</p>
          <p className="text-sm text-destructive">{error.message}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Clientes</h1>
          <p className="text-muted-foreground">Gerencie sua base de clientes</p>
        </div>
        <Button 
          className="bg-gradient-primary"
          onClick={() => {
            resetForm();
            setIsCreateModalOpen(true);
          }}
        >
          <Plus className="w-4 h-4 mr-2" />
          Novo Cliente
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total de Clientes</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">{totalItems}</div>
            <p className="text-sm text-success">Total no sistema</p>
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Clientes Ativos</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">
              {customers.filter(c => c.is_active).length}
            </div>
            <p className="text-sm text-muted-foreground">
              {customers.length > 0 ? Math.round((customers.filter(c => c.is_active).length / customers.length) * 100) : 0}% do total
            </p>
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Com Email</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-primary">
              {customers.filter(c => c.email).length}
            </div>
            <p className="text-sm text-success">
              {customers.length > 0 ? Math.round((customers.filter(c => c.email).length / customers.length) * 100) : 0}% do total
            </p>
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Com Documento</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">
              {customers.filter(c => c.document).length}
            </div>
            <p className="text-sm text-success">
              {customers.length > 0 ? Math.round((customers.filter(c => c.document).length / customers.length) * 100) : 0}% do total
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Customer List */}
      <Card className="border-0 shadow-custom-md">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Users className="w-5 h-5 text-primary" />
              Lista de Clientes ({customers.length} de {totalItems})
            </CardTitle>
            <div className="flex items-center gap-3">
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
                <TableHead className="w-24">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {customers.map((customer) => (
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
                      {/* Botão para conversa */}
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => {
                          // Navegar para conversa com o cliente
                          const whatsappNumber = customer.phone.replace('@c.us', '');
                          window.open(`/conversations?phone=${whatsappNumber}`, '_blank');
                        }}
                        title="Ir para conversa"
                      >
                        <MessageCircle className="w-4 h-4" />
                      </Button>
                      
                      {/* Botão para pedidos */}
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => {
                          setSelectedCustomer(customer);
                          setIsOrdersModalOpen(true);
                        }}
                        title="Ver pedidos"
                      >
                        <ShoppingBag className="w-4 h-4" />
                      </Button>
                      
                      {/* Botão para editar */}
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => openEditModal(customer)}
                        title="Editar cliente"
                      >
                        <Edit className="w-4 h-4" />
                        Editar
                      </Button>
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => {
                          setSelectedCustomer(customer);
                          setIsAddressModalOpen(true);
                        }}
                      >
                        <MapPin className="w-4 h-4" />
                        Endereços
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
                        <Edit className="w-4 h-4" />
                        Editar
                      </Button>
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => {
                          setSelectedCustomer(customer);
                          setIsAddressModalOpen(true);
                        }}
                      >
                        <MapPin className="w-4 h-4" />
                        Endereços
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
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
            <div className="flex items-center justify-between mt-6">
              <div className="text-sm text-muted-foreground">
                Página {page} de {totalPages} - {totalItems} clientes no total
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
                <Input
                  id="phone"
                  value={formData.phone}
                  onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                  placeholder="(11) 99999-9999"
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
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="">Selecione</option>
                  <option value="masculino">Masculino</option>
                  <option value="feminino">Feminino</option>
                  <option value="outro">Outro</option>
                </select>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="notes">Observações</Label>
              <textarea
                id="notes"
                value={formData.notes}
                onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                placeholder="Observações sobre o cliente"
                className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setIsCreateModalOpen(false)}>
              Cancelar
            </Button>
            <Button onClick={handleCreateCustomer} disabled={isSubmitting}>
              {isSubmitting && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              Criar Cliente
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
              Edite as informações do cliente.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="edit-name">Nome *</Label>
                <Input
                  id="edit-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Nome completo"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-phone">Telefone *</Label>
                <Input
                  id="edit-phone"
                  value={formData.phone}
                  onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                  placeholder="(11) 99999-9999"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="edit-email">Email</Label>
                <Input
                  id="edit-email"
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  placeholder="email@exemplo.com"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-document">Documento</Label>
                <Input
                  id="edit-document"
                  value={formData.document}
                  onChange={(e) => setFormData({ ...formData, document: e.target.value })}
                  placeholder="CPF/CNPJ"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="edit-birth_date">Data de Nascimento</Label>
                <Input
                  id="edit-birth_date"
                  type="date"
                  value={formData.birth_date}
                  onChange={(e) => setFormData({ ...formData, birth_date: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-gender">Gênero</Label>
                <select
                  id="edit-gender"
                  value={formData.gender}
                  onChange={(e) => setFormData({ ...formData, gender: e.target.value })}
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="">Selecione</option>
                  <option value="masculino">Masculino</option>
                  <option value="feminino">Feminino</option>
                  <option value="outro">Outro</option>
                </select>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-notes">Observações</Label>
              <textarea
                id="edit-notes"
                value={formData.notes}
                onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                placeholder="Observações sobre o cliente"
                className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setIsEditModalOpen(false)}>
              Cancelar
            </Button>
            <Button onClick={handleEditCustomer} disabled={isSubmitting}>
              {isSubmitting && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              Salvar Alterações
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Address Manager Modal */}
      {selectedCustomer && (
        <AddressManager
          customer={selectedCustomer}
          isOpen={isAddressModalOpen}
          onClose={() => {
            setIsAddressModalOpen(false);
            setSelectedCustomer(null);
          }}
        />
      )}
    </div>
  );
}