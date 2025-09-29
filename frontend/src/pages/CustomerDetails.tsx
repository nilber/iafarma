import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeft, User, Phone, Mail, FileText, Calendar, Edit, Save, X, MapPin, ShoppingBag, MessageCircle, Plus } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useCustomer, useUpdateCustomer, useAddressesByCustomer, useOrdersByCustomer } from "@/lib/api/hooks";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";
import { formatPhoneForDisplay, cleanPhoneForStorage } from "@/lib/phone-utils";
import { formatCurrency, formatDateTime } from "@/lib/utils";
import { toast } from "sonner";
import { Customer } from "@/lib/api/types";
import { AddressManager } from "@/components/AddressManager";

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

export default function CustomerDetails() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [isEditing, setIsEditing] = useState(false);
  const [isAddressModalOpen, setIsAddressModalOpen] = useState(false);
  const [formData, setFormData] = useState<CustomerFormData>({
    name: '',
    email: '',
    phone: '',
    document: '',
    birth_date: '',
    gender: '',
    notes: '',
    is_active: true,
  });

  // Fetch customer data
  const { data: customer, isLoading, error, refetch } = useCustomer(id!);
  const { data: addresses } = useAddressesByCustomer(id!);
  const { data: ordersResult } = useOrdersByCustomer(id!, { limit: 10 });
  
  const updateCustomerMutation = useUpdateCustomer();

  // Initialize form data when customer is loaded
  useEffect(() => {
    if (customer) {
      setFormData({
        name: customer.name || '',
        email: customer.email || '',
        phone: customer.phone || '',
        document: customer.document || '',
        birth_date: customer.birth_date || '',
        gender: customer.gender || '',
        notes: customer.notes || '',
        is_active: customer.is_active,
      });
    }
  }, [customer]);

  const handleSave = async () => {
    if (!customer) return;
    
    try {
      const cleanedData = {
        ...formData,
        phone: cleanPhoneForStorage(formData.phone),
      };

      await updateCustomerMutation.mutateAsync({
        id: customer.id,
        customer: cleanedData,
      });

      toast.success("Cliente atualizado com sucesso!");
      setIsEditing(false);
      refetch();
    } catch (error) {
      console.error('Error updating customer:', error);
      toast.error("Erro ao atualizar cliente");
    }
  };

  const handleCancel = () => {
    if (customer) {
      setFormData({
        name: customer.name || '',
        email: customer.email || '',
        phone: customer.phone || '',
        document: customer.document || '',
        birth_date: customer.birth_date || '',
        gender: customer.gender || '',
        notes: customer.notes || '',
        is_active: customer.is_active,
      });
    }
    setIsEditing(false);
  };

  const getInitials = (name?: string) => {
    if (!name) return 'CL';
    return name
      .split(' ')
      .map(word => word.charAt(0))
      .slice(0, 2)
      .join('')
      .toUpperCase();
  };

  const getStatusBadge = (isActive: boolean) => {
    return (
      <Badge variant={isActive ? "default" : "secondary"} className={isActive ? "bg-green-500" : ""}>
        {isActive ? "Ativo" : "Inativo"}
      </Badge>
    );
  };

  const getOrderStatusBadge = (status?: string) => {
    switch (status) {
      case 'completed':
        return <Badge variant="default" className="bg-green-500">Concluído</Badge>;
      case 'pending':
        return <Badge variant="secondary">Pendente</Badge>;
      case 'cancelled':
        return <Badge variant="destructive">Cancelado</Badge>;
      default:
        return <Badge variant="outline">{status || 'Desconhecido'}</Badge>;
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Carregando dados do cliente...</p>
        </div>
      </div>
    );
  }

  if (error || !customer) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <User className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="text-lg font-semibold mb-2">Cliente não encontrado</h3>
          <p className="text-muted-foreground mb-4">
            Não foi possível encontrar os dados deste cliente.
          </p>
          <Button onClick={() => navigate(-1)}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="outline" onClick={() => navigate(-1)}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar
          </Button>
          <div>
            <h1 className="text-2xl font-bold">{customer.name || 'Cliente sem nome'}</h1>
            <p className="text-muted-foreground">
              Cadastrado em {format(new Date(customer.created_at), "dd/MM/yyyy", { locale: ptBR })}
            </p>
          </div>
        </div>
        
        <div className="flex items-center gap-3">
          {getStatusBadge(customer.is_active)}
          {isEditing ? (
            <div className="flex gap-2">
              <Button variant="outline" onClick={handleCancel}>
                <X className="w-4 h-4 mr-2" />
                Cancelar
              </Button>
              <Button onClick={handleSave} disabled={updateCustomerMutation.isPending}>
                <Save className="w-4 h-4 mr-2" />
                Salvar
              </Button>
            </div>
          ) : (
            <Button onClick={() => setIsEditing(true)}>
              <Edit className="w-4 h-4 mr-2" />
              Editar
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Customer Info */}
        <div className="lg:col-span-2 space-y-6">
          {/* Basic Information */}
          <Card className="border-0 shadow-custom-md">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <User className="w-5 h-5 text-primary" />
                Informações Básicas
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-4 mb-6">
                <Avatar className="w-16 h-16">
                  <AvatarFallback className="text-lg font-semibold bg-primary/10 text-primary">
                    {getInitials(customer.name)}
                  </AvatarFallback>
                </Avatar>
                <div>
                  <h3 className="text-lg font-semibold">{customer.name || 'Nome não informado'}</h3>
                  <p className="text-sm text-muted-foreground">ID: {customer.id}</p>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="name">Nome</Label>
                  {isEditing ? (
                    <Input
                      id="name"
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      placeholder="Nome do cliente"
                    />
                  ) : (
                    <p className="text-sm py-2">{customer.name || 'Não informado'}</p>
                  )}
                </div>

                <div>
                  <Label htmlFor="email">Email</Label>
                  {isEditing ? (
                    <Input
                      id="email"
                      type="email"
                      value={formData.email}
                      onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                      placeholder="email@exemplo.com"
                    />
                  ) : (
                    <p className="text-sm py-2">{customer.email || 'Não informado'}</p>
                  )}
                </div>

                <div>
                  <Label htmlFor="phone">Telefone</Label>
                  {isEditing ? (
                    <Input
                      id="phone"
                      value={formData.phone}
                      onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                      placeholder="(11) 99999-9999"
                    />
                  ) : (
                    <p className="text-sm py-2">{formatPhoneForDisplay(customer.phone) || 'Não informado'}</p>
                  )}
                </div>

                <div>
                  <Label htmlFor="document">CPF/CNPJ</Label>
                  {isEditing ? (
                    <Input
                      id="document"
                      value={formData.document}
                      onChange={(e) => setFormData({ ...formData, document: e.target.value })}
                      placeholder="000.000.000-00"
                    />
                  ) : (
                    <p className="text-sm py-2">{customer.document || 'Não informado'}</p>
                  )}
                </div>

                <div>
                  <Label htmlFor="birth_date">Data de Nascimento</Label>
                  {isEditing ? (
                    <Input
                      id="birth_date"
                      type="date"
                      value={formData.birth_date}
                      onChange={(e) => setFormData({ ...formData, birth_date: e.target.value })}
                    />
                  ) : (
                    <p className="text-sm py-2">
                      {customer.birth_date 
                        ? format(new Date(customer.birth_date), "dd/MM/yyyy", { locale: ptBR })
                        : 'Não informado'
                      }
                    </p>
                  )}
                </div>

                <div>
                  <Label htmlFor="gender">Gênero</Label>
                  {isEditing ? (
                    <select
                      id="gender"
                      value={formData.gender}
                      onChange={(e) => setFormData({ ...formData, gender: e.target.value })}
                      className="w-full px-3 py-2 border border-input rounded-md bg-background"
                    >
                      <option value="">Selecione</option>
                      <option value="male">Masculino</option>
                      <option value="female">Feminino</option>
                      <option value="other">Outro</option>
                    </select>
                  ) : (
                    <p className="text-sm py-2">
                      {customer.gender === 'male' ? 'Masculino' : 
                       customer.gender === 'female' ? 'Feminino' :
                       customer.gender === 'other' ? 'Outro' : 'Não informado'}
                    </p>
                  )}
                </div>
              </div>

              <div>
                <Label htmlFor="notes">Observações</Label>
                {isEditing ? (
                  <Textarea
                    id="notes"
                    value={formData.notes}
                    onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                    placeholder="Observações sobre o cliente..."
                    rows={3}
                  />
                ) : (
                  <p className="text-sm py-2">{customer.notes || 'Nenhuma observação'}</p>
                )}
              </div>

              {isEditing && (
                <div className="flex items-center space-x-2">
                  <Switch
                    id="is_active"
                    checked={formData.is_active}
                    onCheckedChange={(checked) => setFormData({ ...formData, is_active: checked })}
                  />
                  <Label htmlFor="is_active">Cliente ativo</Label>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Recent Orders */}
          <Card className="border-0 shadow-custom-md">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShoppingBag className="w-5 h-5 text-primary" />
                Pedidos Recentes
              </CardTitle>
              <CardDescription>
                Últimos {ordersResult?.data?.length || 0} pedidos do cliente
              </CardDescription>
            </CardHeader>
            <CardContent>
              {ordersResult?.data && ordersResult.data.length > 0 ? (
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
                    {ordersResult.data.map((order) => (
                      <TableRow key={order.id}>
                        <TableCell>
                          <div className="font-medium">{order.order_number || `#${order.id.slice(0, 8)}`}</div>
                        </TableCell>
                        <TableCell>
                          {formatDateTime(order.created_at)}
                        </TableCell>
                        <TableCell>
                          {getOrderStatusBadge(order.status)}
                        </TableCell>
                        <TableCell>
                          <span className="font-medium">
                            {formatCurrency(parseFloat(order.total_amount || '0'))}
                          </span>
                        </TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => navigate(`/sales/orders/${order.id}`)}
                          >
                            Ver Detalhes
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <ShoppingBag className="w-12 h-12 mx-auto mb-4 opacity-50" />
                  <p>Nenhum pedido encontrado</p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Quick Actions */}
          <Card className="border-0 shadow-custom-md">
            <CardHeader>
              <CardTitle>Ações Rápidas</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <Button 
                variant="outline" 
                className="w-full justify-start"
                onClick={() => navigate(`/whatsapp/conversations?customer=${customer.id}`)}
              >
                <MessageCircle className="w-4 h-4 mr-2" />
                Ver Conversas
              </Button>
              <Button 
                variant="outline" 
                className="w-full justify-start"
                onClick={() => navigate(`/sales/orders?customer=${customer.id}`)}
              >
                <ShoppingBag className="w-4 h-4 mr-2" />
                Ver Todos os Pedidos
              </Button>
            </CardContent>
          </Card>

          {/* Addresses */}
          <Card className="border-0 shadow-custom-md">
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <MapPin className="w-5 h-5 text-primary" />
                  Endereços
                </CardTitle>
                <Button 
                  variant="outline" 
                  size="sm"
                  onClick={() => setIsAddressModalOpen(true)}
                >
                  <Plus className="w-4 h-4 mr-2" />
                  Gerenciar Endereços
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {addresses?.addresses && addresses.addresses.length > 0 ? (
                <div className="space-y-3">
                  {addresses.addresses.map((address) => (
                    <div key={address.id} className="p-3 border rounded-lg">
                      <div className="flex items-start justify-between">
                        <div>
                          <div className="flex items-center gap-2 mb-1">
                            <span className="font-medium text-sm">{address.label}</span>
                            {address.is_default && (
                              <Badge variant="secondary" className="text-xs">Principal</Badge>
                            )}
                          </div>
                          <p className="text-sm text-muted-foreground">
                            {address.street}, {address.number}
                            {address.complement && `, ${address.complement}`}
                          </p>
                          <p className="text-sm text-muted-foreground">
                            {address.neighborhood} - {address.city}/{address.state}
                          </p>
                          <p className="text-sm text-muted-foreground">
                            CEP: {address.zip_code}
                          </p>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-4 text-muted-foreground">
                  <MapPin className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">Nenhum endereço cadastrado</p>
                  <Button 
                    variant="outline" 
                    size="sm"
                    onClick={() => setIsAddressModalOpen(true)}
                    className="mt-2"
                  >
                    <Plus className="w-4 h-4 mr-2" />
                    Adicionar Primeiro Endereço
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Customer Stats */}
          <Card className="border-0 shadow-custom-md">
            <CardHeader>
              <CardTitle>Estatísticas</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Total de Pedidos</span>
                <span className="font-medium">{ordersResult?.total || 0}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Endereços Cadastrados</span>
                <span className="font-medium">{addresses?.addresses?.length || 0}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Membro desde</span>
                <span className="font-medium">
                  {format(new Date(customer.created_at), "MMM yyyy", { locale: ptBR })}
                </span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Address Manager Modal */}
      {customer && (
        <AddressManager 
          customer={customer} 
          isOpen={isAddressModalOpen} 
          onClose={() => setIsAddressModalOpen(false)} 
        />
      )}
    </div>
  );
}
