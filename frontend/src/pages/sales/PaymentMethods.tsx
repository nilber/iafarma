import { useState, useEffect } from 'react';
import { Plus, Search, Pencil, Trash2, Eye, EyeOff } from 'lucide-react';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useToast } from "@/hooks/use-toast";
import { apiClient } from "@/lib/api/client";
import { PaymentMethod } from "@/lib/api/types";

interface PaymentMethodFormData {
  name: string;
  is_active: boolean;
}

interface PaymentMethodFormProps {
  paymentMethod?: PaymentMethod;
  onSave: (data: PaymentMethodFormData) => Promise<void>;
  onCancel: () => void;
  isLoading: boolean;
}

function PaymentMethodForm({ paymentMethod, onSave, onCancel, isLoading }: PaymentMethodFormProps) {
  const [formData, setFormData] = useState<PaymentMethodFormData>({
    name: paymentMethod?.name || '',
    is_active: paymentMethod?.is_active ?? true,
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Nome é obrigatório';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) return;

    try {
      await onSave(formData);
    } catch (error) {
      console.error('Error saving payment method:', error);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <label htmlFor="name" className="text-sm font-medium">
          Nome *
        </label>
        <Input
          id="name"
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          placeholder="Digite o nome da forma de pagamento"
          className={errors.name ? 'border-red-500' : ''}
        />
        {errors.name && <p className="text-sm text-red-500">{errors.name}</p>}
      </div>

      <div className="flex items-center space-x-2">
        <input
          type="checkbox"
          id="is_active"
          checked={formData.is_active}
          onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
          className="h-4 w-4 text-primary focus:ring-primary border-gray-300 rounded"
        />
        <label htmlFor="is_active" className="text-sm font-medium">
          Ativo
        </label>
      </div>

      <div className="flex gap-2 pt-4">
        <Button type="submit" disabled={isLoading}>
          {isLoading ? 'Salvando...' : 'Salvar'}
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancelar
        </Button>
      </div>
    </form>
  );
}

export default function PaymentMethods() {
  const [paymentMethods, setPaymentMethods] = useState<PaymentMethod[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [showInactive, setShowInactive] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [editingPaymentMethod, setEditingPaymentMethod] = useState<PaymentMethod | undefined>();
  const [formLoading, setFormLoading] = useState(false);
  const { toast } = useToast();

  // Pagination
  const [currentPage, setCurrentPage] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const itemsPerPage = 10;

  const loadPaymentMethods = async () => {
    try {
      setLoading(true);
      
      const params = new URLSearchParams({
        page: currentPage.toString(),
        limit: itemsPerPage.toString(),
        ...(searchTerm && { search: searchTerm }),
        ...(showInactive && { show_inactive: 'true' }),
      });

      const response = await apiClient.getPaymentMethods({
        page: currentPage,
        limit: itemsPerPage,
        ...(searchTerm && { search: searchTerm }),
        ...(showInactive && { show_inactive: true }),
      });
      setPaymentMethods(response.data);
      setTotalItems(response.total);
    } catch (error) {
      console.error('Error loading payment methods:', error);
      toast({
        title: "Erro",
        description: "Não foi possível carregar as formas de pagamento",
        variant: "destructive",
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPaymentMethods();
  }, [currentPage, searchTerm, showInactive]);

  const handleCreateOrUpdate = async (data: PaymentMethodFormData) => {
    try {
      setFormLoading(true);

      if (editingPaymentMethod) {
        await apiClient.updatePaymentMethod(editingPaymentMethod.id, data);
        toast({
          title: "Sucesso",
          description: "Forma de pagamento atualizada com sucesso",
        });
      } else {
        await apiClient.createPaymentMethod(data);
        toast({
          title: "Sucesso",
          description: "Forma de pagamento criada com sucesso",
        });
      }

      setShowForm(false);
      setEditingPaymentMethod(undefined);
      await loadPaymentMethods();
    } catch (error) {
      console.error('Error saving payment method:', error);
      toast({
        title: "Erro",
        description: "Não foi possível salvar a forma de pagamento",
        variant: "destructive",
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDelete = async (paymentMethod: PaymentMethod) => {
    if (!window.confirm('Tem certeza que deseja excluir esta forma de pagamento?')) {
      return;
    }

    try {
      await apiClient.deletePaymentMethod(paymentMethod.id);
      toast({
        title: "Sucesso",
        description: "Forma de pagamento excluída com sucesso",
      });
      await loadPaymentMethods();
    } catch (error: any) {
      console.error('Error deleting payment method:', error);
      const message = error.response?.data?.message || "Não foi possível excluir a forma de pagamento";
      toast({
        title: "Erro",
        description: message,
        variant: "destructive",
      });
    }
  };

  const totalPages = Math.ceil(totalItems / itemsPerPage);

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Formas de Pagamento</h1>
          <p className="text-muted-foreground">
            Gerencie as formas de pagamento disponíveis no sistema
          </p>
        </div>
        <Button 
          onClick={() => {
            setEditingPaymentMethod(undefined);
            setShowForm(true);
          }}
        >
          <Plus className="mr-2 h-4 w-4" />
          Nova Forma de Pagamento
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Lista de Formas de Pagamento</CardTitle>
          <CardDescription>
            {totalItems} forma{totalItems !== 1 ? 's' : ''} de pagamento encontrada{totalItems !== 1 ? 's' : ''}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4 mb-6">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
                <Input
                  placeholder="Buscar por nome..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>
            <Button
              variant="outline"
              onClick={() => setShowInactive(!showInactive)}
            >
              {showInactive ? <EyeOff className="mr-2 h-4 w-4" /> : <Eye className="mr-2 h-4 w-4" />}
              {showInactive ? 'Ocultar Inativas' : 'Mostrar Inativas'}
            </Button>
          </div>

          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nome</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Criado em</TableHead>
                  <TableHead className="w-[100px]">Ações</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center py-8">
                      Carregando...
                    </TableCell>
                  </TableRow>
                ) : paymentMethods.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center py-8 text-muted-foreground">
                      Nenhuma forma de pagamento encontrada
                    </TableCell>
                  </TableRow>
                ) : (
                  paymentMethods.map((paymentMethod) => (
                    <TableRow key={paymentMethod.id}>
                      <TableCell className="font-medium">
                        {paymentMethod.name}
                      </TableCell>
                      <TableCell>
                        <Badge variant={paymentMethod.is_active ? "default" : "secondary"}>
                          {paymentMethod.is_active ? "Ativo" : "Inativo"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {new Date(paymentMethod.created_at).toLocaleDateString('pt-BR')}
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              setEditingPaymentMethod(paymentMethod);
                              setShowForm(true);
                            }}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(paymentMethod)}
                            className="text-destructive hover:text-destructive"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex justify-center gap-2 mt-6">
              <Button
                variant="outline"
                disabled={currentPage === 1}
                onClick={() => setCurrentPage(currentPage - 1)}
              >
                Anterior
              </Button>
              <span className="flex items-center px-4 py-2 text-sm">
                Página {currentPage} de {totalPages}
              </span>
              <Button
                variant="outline"
                disabled={currentPage === totalPages}
                onClick={() => setCurrentPage(currentPage + 1)}
              >
                Próxima
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Form Dialog */}
      <Dialog open={showForm} onOpenChange={setShowForm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingPaymentMethod ? 'Editar' : 'Nova'} Forma de Pagamento
            </DialogTitle>
            <DialogDescription>
              {editingPaymentMethod 
                ? 'Edite as informações da forma de pagamento'
                : 'Preencha as informações da nova forma de pagamento'
              }
            </DialogDescription>
          </DialogHeader>
          <PaymentMethodForm
            paymentMethod={editingPaymentMethod}
            onSave={handleCreateOrUpdate}
            onCancel={() => {
              setShowForm(false);
              setEditingPaymentMethod(undefined);
            }}
            isLoading={formLoading}
          />
        </DialogContent>
      </Dialog>
    </div>
  );
}