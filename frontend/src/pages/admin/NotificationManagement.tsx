import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { Tenant, NotificationLog, NotificationStats } from '@/lib/api/types';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Checkbox } from '@/components/ui/checkbox';
import { toast } from 'sonner';
import { 
  Mail, 
  RefreshCw, 
  Search, 
  Filter,
  Send,
  TrendingUp,
  AlertTriangle,
  Calendar,
  Users
} from 'lucide-react';
import { format } from 'date-fns';
import { ptBR } from 'date-fns/locale';

export default function NotificationManagement() {
  const [selectedNotifications, setSelectedNotifications] = useState<string[]>([]);
  const [filters, setFilters] = useState({
    tenant_id: '',
    type: '',
    status: '',
    page: 1,
    limit: 20,
  });
  const [showResendDialog, setShowResendDialog] = useState(false);
  const [showTriggerDialog, setShowTriggerDialog] = useState(false);
  const [selectedTenants, setSelectedTenants] = useState<string[]>([]);
  const [triggerType, setTriggerType] = useState<'daily_sales_report' | 'low_stock_alert'>('daily_sales_report');
  const [forceResend, setForceResend] = useState(false);

  const queryClient = useQueryClient();

  // Fetch notifications
  const { data: notificationsData, isLoading: loadingNotifications, error: notificationsError } = useQuery({
    queryKey: ['admin-notifications', filters],
    queryFn: async () => {
      console.log('Fetching notifications with filters:', filters);
      const result = await apiClient.getNotifications({
        limit: 50,
        offset: 0,
        ...filters
      });
      console.log('Notifications result:', result);
      return result;
    },
  });

  // Fetch notification stats
  const { data: stats } = useQuery({
    queryKey: ['admin-notification-stats', filters.tenant_id],
    queryFn: async () => {
      return await apiClient.getNotificationStats({
        tenant_id: filters.tenant_id || undefined
      });
    },
  });

  // Fetch tenants
  const { data: tenantsData } = useQuery({
    queryKey: ['admin-tenants'],
    queryFn: async () => {
      return await apiClient.getTenants({ limit: 100, offset: 0 });
    },
  });

  // Resend notification mutation
  const resendMutation = useMutation({
    mutationFn: async ({ notificationId, tenantIds, forceResend }: {
      notificationId: string;
      tenantIds: string[];
      forceResend: boolean;
    }) => {
      return await apiClient.resendNotifications({
        notification_id: notificationId,
        tenant_ids: tenantIds.length > 0 ? tenantIds : [],
        force_resend: forceResend,
      });
    },
    onSuccess: (data) => {
      toast.success(data.message || 'Notificação reenviada com sucesso');
      
      queryClient.invalidateQueries({ queryKey: ['admin-notifications'] });
      queryClient.invalidateQueries({ queryKey: ['admin-notification-stats'] });
      setShowResendDialog(false);
      setSelectedTenants([]);
      setSelectedNotifications([]);
    },
    onError: () => {
      toast.error('Erro ao reenviar notificação');
    },
  });

  // Trigger notification mutation
  const triggerMutation = useMutation({
    mutationFn: async ({ type, tenantIds, forceRun }: {
      type: string;
      tenantIds: string[];
      forceRun: boolean;
    }) => {
      return await apiClient.triggerNotifications({
        type,
        tenant_ids: tenantIds,
        force_run: forceRun,
      });
    },
    onSuccess: (data) => {
      toast.success(data.message || 'Notificação disparada com sucesso');
      
      queryClient.invalidateQueries({ queryKey: ['admin-notifications'] });
      queryClient.invalidateQueries({ queryKey: ['admin-notification-stats'] });
      setShowTriggerDialog(false);
      setSelectedTenants([]);
    },
    onError: () => {
      toast.error('Erro ao disparar notificação');
    },
  });

  const getStatusBadge = (status: string) => {
    const variants: Record<string, 'default' | 'secondary' | 'destructive'> = {
      sent: 'default',
      failed: 'destructive',
      pending: 'secondary',
    };
    
    return (
      <Badge variant={variants[status] || 'secondary'}>
        {status === 'sent' ? 'Enviado' : status === 'failed' ? 'Falha' : 'Pendente'}
      </Badge>
    );
  };

  const getTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      daily_sales_report: 'Relatório Diário',
      low_stock_alert: 'Alerta de Estoque',
      daily_sales_report_resend: 'Relatório Diário (Reenvio)',
      low_stock_alert_resend: 'Alerta de Estoque (Reenvio)',
    };
    
    return labels[type] || type;
  };

  const handleSelectNotification = (id: string) => {
    setSelectedNotifications(prev => 
      prev.includes(id) 
        ? prev.filter(n => n !== id)
        : [...prev, id]
    );
  };

  const handleSelectAllNotifications = () => {
    if (selectedNotifications.length === notificationsData?.data.length) {
      setSelectedNotifications([]);
    } else {
      setSelectedNotifications(notificationsData?.data.map((n: NotificationLog) => n.id) || []);
    }
  };

  const handleTenantToggle = (tenantId: string) => {
    setSelectedTenants(prev =>
      prev.includes(tenantId)
        ? prev.filter(id => id !== tenantId)
        : [...prev, tenantId]
    );
  };

  const handleResendSelected = () => {
    if (selectedNotifications.length === 0) {
      toast.error('Selecione pelo menos uma notificação para reenviar.');
      return;
    }
    setShowResendDialog(true);
  };

  const handleResendNotification = async () => {
    if (selectedNotifications.length === 0) return;
    
    try {
      // Resend each selected notification individually 
      // Each notification already has its tenant_id, so we don't need to specify tenant_ids
      for (const notificationId of selectedNotifications) {
        await resendMutation.mutateAsync({
          notificationId,
          tenantIds: [], // Empty array since notification already has tenant info
          forceResend: forceResend
        });
      }
      
      toast.success(`${selectedNotifications.length} notificação(ões) reenviada(s) com sucesso!`);
      setShowResendDialog(false);
      setSelectedNotifications([]);
    } catch (error) {
      console.error('Erro ao reenviar notificações:', error);
      toast.error('Erro ao reenviar notificações. Tente novamente.');
    }
  };

  const confirmTrigger = () => {
    if (selectedTenants.length === 0) {
      toast.error('Selecione pelo menos um tenant');
      return;
    }
    
    triggerMutation.mutate({
      type: triggerType,
      tenantIds: selectedTenants,
      forceRun: forceResend,
    });
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-foreground">Gerenciamento de Notificações</h1>
        <p className="text-muted-foreground">
          Monitore e gerencie todas as notificações do sistema
        </p>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Enviadas</CardTitle>
            <Mail className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.total || 0}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Enviadas</CardTitle>
            <Calendar className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.sent || 0}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Falhas</CardTitle>
            <AlertTriangle className="h-4 w-4 text-destructive" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-destructive">{stats?.failed || 0}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Taxa de Sucesso</CardTitle>
            <TrendingUp className="h-4 w-4 text-green-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {stats?.total ? Math.round(((stats.total - (stats.failed || 0)) / stats.total) * 100) : 100}%
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters and Actions */}
      <Card>
        <CardHeader>
          <CardTitle>Filtros e Ações</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Select value={filters.tenant_id || "all"} onValueChange={(value) => setFilters(prev => ({ ...prev, tenant_id: value === "all" ? "" : value, page: 1 }))}>
                <SelectTrigger>
                  <SelectValue placeholder="Filtrar por tenant" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Todos os tenants</SelectItem>
                  {tenantsData?.data?.map((tenant: Tenant) => (
                    <SelectItem key={tenant.id} value={tenant.id}>
                      {tenant.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="flex-1 min-w-[200px]">
              <Select value={filters.type || "all"} onValueChange={(value) => setFilters(prev => ({ ...prev, type: value === "all" ? "" : value, page: 1 }))}>
                <SelectTrigger>
                  <SelectValue placeholder="Filtrar por tipo" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Todos os tipos</SelectItem>
                  <SelectItem value="daily_sales_report">Relatório Diário</SelectItem>
                  <SelectItem value="low_stock_alert">Alerta de Estoque</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex-1 min-w-[200px]">
              <Select value={filters.status || "all"} onValueChange={(value) => setFilters(prev => ({ ...prev, status: value === "all" ? "" : value, page: 1 }))}>
                <SelectTrigger>
                  <SelectValue placeholder="Filtrar por status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Todos os status</SelectItem>
                  <SelectItem value="sent">Enviado</SelectItem>
                  <SelectItem value="failed">Falha</SelectItem>
                  <SelectItem value="pending">Pendente</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="flex gap-2">
            <Button
              onClick={handleResendSelected}
              disabled={selectedNotifications.length === 0}
              variant="outline"
            >
              <RefreshCw className="h-4 w-4 mr-2" />
              Reenviar Selecionadas
            </Button>
            
            <Button
              onClick={() => setShowTriggerDialog(true)}
              variant="outline"
            >
              <Send className="h-4 w-4 mr-2" />
              Disparar Notificação
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Notifications Table */}
      <Card>
        <CardHeader>
          <CardTitle>Histórico de Notificações</CardTitle>
          <CardDescription>
            Lista de todas as notificações enviadas
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loadingNotifications ? (
            <div className="space-y-2">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-12 bg-muted rounded animate-pulse" />
              ))}
            </div>
          ) : notificationsError ? (
            <div className="text-red-500 p-4">
              Erro ao carregar notificações: {notificationsError.message}
            </div>
          ) : !notificationsData?.data?.length ? (
            <div className="text-center p-8 text-muted-foreground">
              Nenhuma notificação encontrada
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12">
                    <Checkbox
                      checked={selectedNotifications.length === notificationsData?.data?.length && notificationsData?.data?.length > 0}
                      onCheckedChange={handleSelectAllNotifications}
                    />
                  </TableHead>
                  <TableHead>Tenant</TableHead>
                  <TableHead>Tipo</TableHead>
                  <TableHead>Destinatário</TableHead>
                  <TableHead>Assunto</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Enviado em</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {notificationsData?.data?.map((notification: NotificationLog) => (
                  <TableRow key={notification.id}>
                    <TableCell>
                      <Checkbox
                        checked={selectedNotifications.includes(notification.id)}
                        onCheckedChange={() => handleSelectNotification(notification.id)}
                      />
                    </TableCell>
                    <TableCell className="font-medium">
                      {notification.tenant_name}
                    </TableCell>
                    <TableCell>
                      {getTypeLabel(notification.type)}
                    </TableCell>
                    <TableCell>{notification.recipient}</TableCell>
                    <TableCell className="max-w-[200px] truncate">
                      {notification.subject}
                    </TableCell>
                    <TableCell>
                      {getStatusBadge(notification.status)}
                    </TableCell>
                    <TableCell>
                      {notification.sent_at 
                        ? format(new Date(notification.sent_at), 'dd/MM/yyyy HH:mm', { locale: ptBR })
                        : '-'
                      }
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Resend Dialog */}
      <Dialog open={showResendDialog} onOpenChange={setShowResendDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Reenviar Notificações Selecionadas</DialogTitle>
            <DialogDescription>
              {selectedNotifications.length === 1 
                ? 'Confirme o reenvio de 1 notificação selecionada.'
                : `Confirme o reenvio de ${selectedNotifications.length} notificações selecionadas.`
              }
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            {/* Mostrar notificações selecionadas com mais detalhes */}
            <div className="space-y-2">
              <h4 className="text-sm font-medium">Notificações que serão reenviadas:</h4>
              <div className="max-h-[200px] overflow-y-auto border rounded p-3 bg-gray-50 space-y-2">
                {selectedNotifications.map(id => {
                  const notification = notificationsData?.data.find((n: NotificationLog) => n.id === id);
                  return notification ? (
                    <div key={id} className="bg-white p-2 rounded border">
                      <div className="flex justify-between items-start">
                        <div className="flex-1">
                          <span className="font-medium text-sm">{getTypeLabel(notification.type)}</span>
                          <Badge 
                            variant={notification.status === 'sent' ? 'default' : 
                                   notification.status === 'failed' ? 'destructive' : 'secondary'}
                            className="ml-2 text-xs"
                          >
                            {notification.status}
                          </Badge>
                        </div>
                      </div>
                      <div className="text-xs text-gray-600 mt-1">
                        <div><strong>Para:</strong> {notification.recipient}</div>
                        <div><strong>Tenant:</strong> {notification.tenant_name}</div>
                        <div><strong>Assunto:</strong> {notification.subject}</div>
                        <div><strong>Enviado em:</strong> {format(new Date(notification.sent_at), 'dd/MM/yyyy HH:mm')}</div>
                      </div>
                    </div>
                  ) : null;
                })}
              </div>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="force-resend"
                checked={forceResend}
                onCheckedChange={(checked) => setForceResend(checked as boolean)}
              />
              <label htmlFor="force-resend" className="text-sm">
                Forçar reenvio (ignorar limitações de "já enviado hoje")
              </label>
            </div>

            <div className="flex gap-2">
              <Button
                onClick={handleResendNotification}
                disabled={resendMutation.isPending}
                className="flex-1"
              >
                {resendMutation.isPending ? (
                  <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4 mr-2" />
                )}
                Confirmar Reenvio
              </Button>
              <Button
                variant="outline"
                onClick={() => setShowResendDialog(false)}
                className="flex-1"
              >
                Cancelar
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Trigger Dialog */}
      <Dialog open={showTriggerDialog} onOpenChange={setShowTriggerDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Disparar Notificação</DialogTitle>
            <DialogDescription>
              Dispare uma notificação manualmente para os tenants selecionados.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium">Tipo de Notificação:</label>
              <Select value={triggerType} onValueChange={(value: any) => setTriggerType(value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="daily_sales_report">Relatório Diário de Vendas</SelectItem>
                  <SelectItem value="low_stock_alert">Alerta de Estoque Baixo</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="force-run"
                checked={forceResend}
                onCheckedChange={(checked) => setForceResend(checked as boolean)}
              />
              <label htmlFor="force-run" className="text-sm">
                Forçar execução (ignorar "já enviado hoje")
              </label>
            </div>

            <div className="space-y-2">
              <h4 className="text-sm font-medium">Selecionar Tenants:</h4>
              <div className="max-h-[200px] overflow-y-auto space-y-1">
                {tenantsData?.data?.map((tenant: Tenant) => (
                  <div key={tenant.id} className="flex items-center space-x-2">
                    <Checkbox
                      id={`trigger-tenant-${tenant.id}`}
                      checked={selectedTenants.includes(tenant.id)}
                      onCheckedChange={() => handleTenantToggle(tenant.id)}
                    />
                    <label htmlFor={`trigger-tenant-${tenant.id}`} className="text-sm">
                      {tenant.name}
                    </label>
                  </div>
                ))}
              </div>
            </div>

            <div className="flex gap-2">
              <Button
                onClick={confirmTrigger}
                disabled={triggerMutation.isPending || selectedTenants.length === 0}
                className="flex-1"
              >
                {triggerMutation.isPending ? (
                  <Send className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <Send className="h-4 w-4 mr-2" />
                )}
                Disparar
              </Button>
              <Button
                variant="outline"
                onClick={() => setShowTriggerDialog(false)}
                className="flex-1"
              >
                Cancelar
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
