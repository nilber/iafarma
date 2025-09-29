import { useState } from "react";
import { Building2, Plus, Search, Filter, Loader2, Edit, MoreHorizontal, Trash2, Zap, Download, Key, MessageSquare } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ChevronLeft, ChevronRight } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useTenants, useDeleteTenant, useExportTenantProducts, useTenantCredits, useTenantStats, useTenantsStats } from "@/lib/api/hooks";
import { API_BASE_URL } from "@/lib/api/client";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";
import { CreateTenantDialog } from "@/components/admin/CreateTenantDialog";
import { EditTenantDialog } from "@/components/admin/EditTenantDialog";
import { TenantCreditManagementDialog } from "@/components/admin/TenantCreditManagementDialog";
import { Tenant } from "@/lib/api/types";
import { toast } from "@/hooks/use-toast";

// Componente para exibir conversas com used/total
function ConversationCell({ tenant }: { tenant: Tenant }) {
  const { data: stats } = useTenantStats(tenant.id);
  const maxConversations = tenant.plan_info?.max_conversations || 0;
  const usedConversations = stats?.active_conversations || 0;
  
  return (
    <span className="font-medium">
      {usedConversations}/{maxConversations}
    </span>
  );
}

// Componente para exibir mensagens do mês atual com used/total
function MessageCell({ tenant }: { tenant: Tenant }) {
  const { data: stats } = useTenantStats(tenant.id);
  const maxMessages = tenant.plan_info?.max_messages_per_month || 0;
  const usedMessages = stats?.messages_current_month || 0;
  
  return (
    <span className="font-medium">
      {usedMessages.toLocaleString()}/{maxMessages.toLocaleString()}
    </span>
  );
}

// Componente para exibir o botão de gerenciar créditos com quantidade
function CreditManageButton({ tenant, onClick }: { tenant: Tenant; onClick: () => void }) {
  const { data: credits } = useTenantCredits(tenant.id);
  
  return (
    <Button
      variant="outline"
      size="sm"
      onClick={onClick}
      className="flex items-center gap-1"
    >
      <Zap className="w-3 h-3 text-yellow-500" />
      Gerenciar {credits ? `(${credits.remaining_credits})` : ''}
    </Button>
  );
}

export default function Tenants() {
  const navigate = useNavigate();
  const [searchTerm, setSearchTerm] = useState("");
  const [page, setPage] = useState(1);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingTenant, setDeletingTenant] = useState<Tenant | null>(null);
  const [creditDialogOpen, setCreditDialogOpen] = useState(false);
  const [creditManagementTenant, setCreditManagementTenant] = useState<Tenant | null>(null);
  const [resetPasswordDialogOpen, setResetPasswordDialogOpen] = useState(false);
  const [resetPasswordTenant, setResetPasswordTenant] = useState<Tenant | null>(null);
  const [isResettingPassword, setIsResettingPassword] = useState(false);
  const limit = 12;

  const { data: tenantResult, isLoading, error } = useTenants({
    limit,
    offset: (page - 1) * limit
  });

  const { data: tenantsStats, isLoading: isLoadingStats } = useTenantsStats();

  const deleteTenanMutation = useDeleteTenant();
  const exportProductsMutation = useExportTenantProducts();

  // Extract tenants from paginated result
  const tenants = tenantResult?.data || [];

  // Filter tenants based on search term
  const filteredTenants = tenants.filter(tenant =>
    tenant.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    tenant.domain?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return <Badge variant="default" className="bg-green-500 hover:bg-green-600">Ativo</Badge>;
      case 'inactive':
        return <Badge variant="destructive">Inativo</Badge>;
      case 'suspended':
        return <Badge variant="secondary">Suspenso</Badge>;
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  };

  const getPlanBadge = (tenant: Tenant) => {
    // Usar o nome do plano da tabela plans se disponível, senão usar o campo plan antigo
    const planName = tenant.plan_info?.name || tenant.plan || 'Sem Plano';
    
    // Cores baseadas no nome do plano
    const getColorClass = (name: string) => {
      const lowercaseName = name.toLowerCase();
      if (lowercaseName.includes('gratuito') || lowercaseName.includes('free')) {
        return "bg-gray-500 hover:bg-gray-600";
      } else if (lowercaseName.includes('básico') || lowercaseName.includes('basic')) {
        return "bg-blue-500 hover:bg-blue-600";
      } else if (lowercaseName.includes('essencial') || lowercaseName.includes('premium')) {
        return "bg-purple-500 hover:bg-purple-600";
      } else if (lowercaseName.includes('pro') || lowercaseName.includes('enterprise')) {
        return "bg-green-500 hover:bg-green-600";
      } else {
        return "bg-orange-500 hover:bg-orange-600";
      }
    };

    return (
      <Badge variant="default" className={getColorClass(planName)}>
        {planName}
      </Badge>
    );
  };

  const getBusinessTypeBadge = (businessType: string | undefined) => {
    switch (businessType) {
      case 'sales':
        return <Badge variant="default" className="bg-green-500 hover:bg-green-600">Vendas</Badge>;    
      default:
        return <Badge variant="outline">Vendas</Badge>; // Default to sales
    }
  };

  const handleEditTenant = (tenant: Tenant) => {
    setEditingTenant(tenant);
    setEditDialogOpen(true);
  };

  const handleDeleteTenant = (tenant: Tenant) => {
    setDeletingTenant(tenant);
    setDeleteDialogOpen(true);
  };

  const handleManageCredits = (tenant: Tenant) => {
    setCreditManagementTenant(tenant);
    setCreditDialogOpen(true);
  };

  const handleExportProducts = async (tenant: Tenant) => {
    try {
      const blob = await exportProductsMutation.mutateAsync(tenant.id);
      
      // Criar URL para download
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      
      // Gerar nome do arquivo com timestamp
      const timestamp = new Date().toISOString().split('T')[0];
      link.download = `produtos_${tenant.name.replace(/[^a-zA-Z0-9]/g, '_')}_${timestamp}.csv`;
      
      // Fazer download
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      
      // Limpar URL
      window.URL.revokeObjectURL(url);
      
      toast({
        title: "Sucesso",
        description: `Produtos de ${tenant.name} exportados com sucesso!`,
      });
    } catch (error: any) {
      toast({
        title: "Erro",
        description: error.message || "Erro ao exportar produtos",
        variant: "destructive",
      });
    }
  };

  const confirmDeleteTenant = async () => {
    if (!deletingTenant) return;
    
    try {
      await deleteTenanMutation.mutateAsync(deletingTenant.id);
      toast({
        title: "Sucesso",
        description: "Empresa excluída com sucesso!",
      });
      setDeleteDialogOpen(false);
      setDeletingTenant(null);
    } catch (error: any) {
      toast({
        title: "Erro",
        description: error.message || "Erro ao excluir empresa",
        variant: "destructive",
      });
    }
  };

  const handleResetPassword = (tenant: Tenant) => {
    setResetPasswordTenant(tenant);
    setResetPasswordDialogOpen(true);
  };

  const confirmResetPassword = async () => {
    if (!resetPasswordTenant) return;
    
    setIsResettingPassword(true);
    try {
      // Verificar se há token válido
      const token = localStorage.getItem('access_token');
      if (!token) {
        throw new Error('Token de acesso não encontrado. Faça login novamente.');
      }

      // Primeiro, buscar o admin do tenant
      const adminResponse = await fetch(`${API_BASE_URL}/admin/tenants/${resetPasswordTenant.id}/users`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });
      
      if (!adminResponse.ok) {
        if (adminResponse.status === 401) {
          throw new Error('Token expirado. Faça login novamente.');
        }
        throw new Error(`Erro ao buscar administrador do tenant (${adminResponse.status})`);
      }
      
      const adminData = await adminResponse.json();
      const tenantAdmin = adminData.users?.find((user: any) => user.role === 'tenant_admin');
      
      if (!tenantAdmin) {
        throw new Error('Administrador do tenant não encontrado');
      }

      // Gerar nova senha de 8 caracteres (números e letras)
      const generatePassword = () => {
        const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        let password = '';
        for (let i = 0; i < 8; i++) {
          password += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return password;
      };

      const newPassword = generatePassword();

      // Atualizar a senha do admin
      const updateResponse = await fetch(`${API_BASE_URL}/admin/tenant-admin/${tenantAdmin.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          name: tenantAdmin.name,
          email: tenantAdmin.email,
          phone: tenantAdmin.phone || '',
          password: newPassword,
        }),
      });

      if (!updateResponse.ok) {
        throw new Error('Erro ao atualizar senha do administrador');
      }

      // Enviar email com as novas credenciais
      const emailResponse = await fetch(`${API_BASE_URL}/admin/send-tenant-credentials`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          tenant_id: resetPasswordTenant.id,
          tenant_name: resetPasswordTenant.name,
          admin_name: tenantAdmin.name,
          admin_email: tenantAdmin.email,
          admin_password: newPassword,
        }),
      });

      if (!emailResponse.ok) {
        throw new Error('Senha atualizada, mas erro ao enviar email');
      }

      toast({
        title: "Sucesso",
        description: `Nova senha gerada e enviada para ${tenantAdmin.email}`,
      });
      
      setResetPasswordDialogOpen(false);
      setResetPasswordTenant(null);
    } catch (error: any) {
      toast({
        title: "Erro",
        description: error.message || "Erro ao resetar senha",
        variant: "destructive",
      });
    } finally {
      setIsResettingPassword(false);
    }
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
          <p className="text-muted-foreground">Erro ao carregar empresas</p>
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
          <h1 className="text-3xl font-bold text-foreground">Empresas</h1>
          <p className="text-muted-foreground">Gerencie empresas do sistema</p>
        </div>
        <Button className="bg-gradient-primary" onClick={() => setCreateDialogOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Nova Empresa
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total de Empresas</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">
              {isLoadingStats ? (
                <div className="animate-pulse bg-muted h-8 w-16 rounded"></div>
              ) : (
                tenantsStats?.total_tenants || 0
              )}
            </div>
            <p className="text-sm text-success">Total cadastradas</p>
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Ativas</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {isLoadingStats ? (
                <div className="animate-pulse bg-muted h-8 w-16 rounded"></div>
              ) : (
                tenantsStats?.active_tenants || 0
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              {tenantsStats && tenantsStats.total_tenants > 0 
                ? Math.round((tenantsStats.active_tenants / tenantsStats.total_tenants) * 100) 
                : 0}% do total
            </p>
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Planos Pagos</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-blue-600">
              {isLoadingStats ? (
                <div className="animate-pulse bg-muted h-8 w-16 rounded"></div>
              ) : (
                tenantsStats?.paid_plan_tenants || 0
              )}
            </div>
            <p className="text-sm text-blue-600">
              {tenantsStats && tenantsStats.total_tenants > 0 
                ? Math.round((tenantsStats.paid_plan_tenants / tenantsStats.total_tenants) * 100) 
                : 0}% do total
            </p>
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Conversas Totais</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">
              {isLoadingStats ? (
                <div className="animate-pulse bg-muted h-8 w-16 rounded"></div>
              ) : (
                tenantsStats?.total_conversations || 0
              )}
            </div>
            <p className="text-sm text-success">Capacidade total de conversas</p>
          </CardContent>
        </Card>
      </div>

      {/* Filters and Search */}
      <Card className="border-0 shadow-custom-md">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Building2 className="w-5 h-5 text-primary" />
              Lista de Empresas ({filteredTenants.length})
            </CardTitle>
            <div className="flex items-center gap-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input 
                  placeholder="Buscar empresas..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10 w-80"
                />
              </div>
              <Button variant="outline" size="sm">
                <Filter className="w-4 h-4 mr-2" />
                Filtros
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Empresa</TableHead>
                <TableHead>Domínio</TableHead>
                <TableHead>Tipo</TableHead>
                <TableHead>Plano</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Conversas</TableHead>
                <TableHead>Mensagens/Mês</TableHead>
                <TableHead>Créditos IA</TableHead>
                <TableHead>Criado em</TableHead>
                <TableHead className="w-24">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredTenants.map((tenant) => (
                <TableRow key={tenant.id} className="hover:bg-accent/50">
                  <TableCell>
                    <div>
                      <div className="font-medium text-foreground">{tenant.name}</div>
                      <div className="text-sm text-muted-foreground">
                        {tenant.admin_email || 'Email não configurado'}
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    {tenant.domain ? (
                      <code className="text-sm bg-muted px-2 py-1 rounded">{tenant.domain}</code>
                    ) : (
                      <span className="text-sm text-muted-foreground">Não configurado</span>
                    )}
                  </TableCell>
                  <TableCell>
                    {getBusinessTypeBadge(tenant.business_type)}
                  </TableCell>
                  <TableCell>
                    {getPlanBadge(tenant)}
                  </TableCell>
                  <TableCell>
                    {getStatusBadge(tenant.status || 'inactive')}
                  </TableCell>
                  <TableCell>
                    <ConversationCell tenant={tenant} />
                  </TableCell>
                  <TableCell>
                    <MessageCell tenant={tenant} />
                  </TableCell>
                  <TableCell>
                    <CreditManageButton 
                      tenant={tenant}
                      onClick={() => handleManageCredits(tenant)}
                    />
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-muted-foreground">
                      {format(new Date(tenant.created_at), 'dd/MM/yyyy', { locale: ptBR })}
                    </span>
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm">
                          <MoreHorizontal className="w-4 h-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent>
                        <DropdownMenuItem onClick={() => handleEditTenant(tenant)}>
                          <Edit className="w-4 h-4 mr-2" />
                          Editar
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleManageCredits(tenant)}>
                          <Zap className="w-4 h-4 mr-2 text-yellow-500" />
                          Créditos IA
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => navigate(`/admin/channels?tenant=${tenant.id}`)}>
                          <MessageSquare className="w-4 h-4 mr-2 text-blue-500" />
                          Administrar Canais
                        </DropdownMenuItem>
                        <DropdownMenuItem 
                          onClick={() => handleExportProducts(tenant)}
                          disabled={exportProductsMutation.isPending}
                        >
                          <Download className="w-4 h-4 mr-2 text-green-500" />
                          {exportProductsMutation.isPending ? 'Exportando...' : 'Exportar Produtos'}
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleResetPassword(tenant)}>
                          <Key className="w-4 h-4 mr-2 text-orange-500" />
                          Resetar Senha Admin
                        </DropdownMenuItem>
                        <DropdownMenuItem 
                          onClick={() => handleDeleteTenant(tenant)}
                          className="text-destructive focus:text-destructive"
                        >
                          <Trash2 className="w-4 h-4 mr-2" />
                          Excluir
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          
          {filteredTenants.length === 0 && !isLoading && (
            <div className="text-center py-8">
              <p className="text-muted-foreground">
                {searchTerm ? 'Nenhuma empresa encontrada para a busca' : 'Nenhuma empresa cadastrada'}
              </p>
            </div>
          )}
        </CardContent>
        
        {/* Paginação */}
        {tenantResult && tenantResult.total_pages > 1 && (
          <div className="flex items-center justify-between px-6 py-4 border-t">
            <div className="text-sm text-muted-foreground">
              Página {tenantResult.page} de {tenantResult.total_pages} - {tenantResult.total} empresas no total
            </div>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page === 1}
              >
                <ChevronLeft className="w-4 h-4" />
                Anterior
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(Math.min(tenantResult.total_pages, page + 1))}
                disabled={page === tenantResult.total_pages}
              >
                Próximo
                <ChevronRight className="w-4 h-4" />
              </Button>
            </div>
          </div>
        )}
      </Card>

      {/* Dialogs */}
      <CreateTenantDialog 
        open={createDialogOpen} 
        onOpenChange={setCreateDialogOpen} 
      />
      
      <EditTenantDialog 
        open={editDialogOpen} 
        onOpenChange={setEditDialogOpen}
        tenant={editingTenant}
      />

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Confirmar Exclusão</AlertDialogTitle>
            <AlertDialogDescription>
              Tem certeza que deseja excluir a empresa "{deletingTenant?.name}"? 
              Esta ação não pode ser desfeita e irá remover todos os dados associados 
              à empresa, incluindo usuários, produtos, pedidos e conversas.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancelar</AlertDialogCancel>
            <AlertDialogAction 
              onClick={confirmDeleteTenant}
              className="bg-destructive hover:bg-destructive/90"
              disabled={deleteTenanMutation.isPending}
            >
              {deleteTenanMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Excluindo...
                </>
              ) : (
                "Excluir"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Credit Management Dialog */}
      <TenantCreditManagementDialog
        tenant={creditManagementTenant}
        open={creditDialogOpen}
        onOpenChange={setCreditDialogOpen}
      />

      {/* Reset Password Confirmation Dialog */}
      <AlertDialog open={resetPasswordDialogOpen} onOpenChange={setResetPasswordDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Resetar Senha do Administrador</AlertDialogTitle>
            <AlertDialogDescription>
              Tem certeza que deseja resetar a senha do administrador da empresa "{resetPasswordTenant?.name}"?
              <br /><br />
              <strong>Uma nova senha de 8 caracteres será gerada automaticamente e enviada por email para o administrador.</strong>
              <br /><br />
              Esta ação irá:
              <ul className="list-disc list-inside mt-2 space-y-1">
                <li>Gerar uma nova senha aleatória</li>
                <li>Atualizar as credenciais do administrador</li>
                <li>Enviar as novas credenciais por email</li>
              </ul>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isResettingPassword}>Cancelar</AlertDialogCancel>
            <AlertDialogAction 
              onClick={confirmResetPassword}
              className="bg-orange-500 hover:bg-orange-600"
              disabled={isResettingPassword}
            >
              {isResettingPassword ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Resetando...
                </>
              ) : (
                <>
                  <Key className="w-4 h-4 mr-2" />
                  Resetar Senha
                </>
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
