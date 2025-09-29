import { useState, useEffect } from "react";
import { Plus, Trash2, Users, Phone, Loader2, Edit2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { PhoneNumberInput } from "@/components/ui/phone-input";
import { toast } from "@/hooks/use-toast";
import { useAlerts, useCreateAlert, useUpdateAlert, useDeleteAlert } from "@/lib/api/hooks";
import { Channel, Alert as AlertType, CreateAlertRequest } from "@/lib/api/types";

interface ChannelAlertsDialogProps {
  channel: Channel;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

interface PhoneNumber {
  id: string;
  number: string;
}

interface AlertForm {
  name: string;
  trigger_on: 'order_created' | 'human_support_request';
  phones: PhoneNumber[];
}

const ChannelAlertsDialog = ({ channel, isOpen, onOpenChange }: ChannelAlertsDialogProps) => {
  const [isCreateMode, setIsCreateMode] = useState(false);
  const [editingAlert, setEditingAlert] = useState<AlertType | null>(null);
  const [alertForm, setAlertForm] = useState<AlertForm>({
    name: '',
    trigger_on: 'order_created',
    phones: []
  });
  const [newPhone, setNewPhone] = useState('');

  // API Hooks
  const { data: alertsData, isLoading: alertsLoading } = useAlerts({ 
    channel_id: channel.id,
    limit: 50 // Get more alerts for this specific channel
  });
  const createAlertMutation = useCreateAlert();
  const updateAlertMutation = useUpdateAlert();
  const deleteAlertMutation = useDeleteAlert();

  const alerts = alertsData?.data || [];

  // Reset form when dialog opens/closes
  useEffect(() => {
    if (!isOpen) {
      setIsCreateMode(false);
      setEditingAlert(null);
      resetForm();
    }
  }, [isOpen]);

  const resetForm = () => {
    setAlertForm({
      name: '',
      trigger_on: 'order_created',
      phones: []
    });
    setNewPhone('');
  };

  const handleStartCreate = () => {
    resetForm();
    setIsCreateMode(true);
    setEditingAlert(null);
  };

  const handleStartEdit = (alert: AlertType) => {
    setAlertForm({
      name: alert.name,
      trigger_on: alert.trigger_on as 'order_created' | 'human_support_request',
      phones: alert.phones ? alert.phones.split(',').map((phone, index) => ({
        id: `phone-${index}`,
        number: phone.trim()
      })) : []
    });
    setEditingAlert(alert);
    setIsCreateMode(false);
  };

  const handleCancelEdit = () => {
    setIsCreateMode(false);
    setEditingAlert(null);
    resetForm();
  };

  // Format phone number for display (more user-friendly)
  const formatPhoneForDisplay = (phoneNumber: string) => {
    // Remove @c.us and format for display
    const cleanNumber = phoneNumber.replace('@c.us', '');
    
    // If it's a Brazilian number (starts with 55)
    if (cleanNumber.startsWith('55') && cleanNumber.length >= 12) {
      const withoutCountry = cleanNumber.slice(2); // Remove 55
      const ddd = withoutCountry.slice(0, 2);
      const number = withoutCountry.slice(2);
      
      // Format as +55 (11) 99999-9999
      if (number.length === 9) {
        return `+55 (${ddd}) ${number.slice(0, 5)}-${number.slice(5)}`;
      } else if (number.length === 8) {
        return `+55 (${ddd}) ${number.slice(0, 4)}-${number.slice(4)}`;
      }
    }
    
    // Fallback to original format
    return phoneNumber;
  };

  const addPhone = () => {
    if (!newPhone.trim()) return;
    
    // Check for duplicates
    if (alertForm.phones.some(p => p.number === newPhone.trim())) {
      toast({
        title: "Erro", 
        description: "Este telefone já foi adicionado",
        variant: "destructive"
      });
      return;
    }

    // Basic validation - check if it looks like a phone number
    if (newPhone.length < 10) {
      toast({
        title: "Erro",
        description: "Número de telefone muito curto",
        variant: "destructive"
      });
      return;
    }

    setAlertForm(prev => ({
      ...prev,
      phones: [...prev.phones, {
        id: `phone-${Date.now()}`,
        number: newPhone.trim()
      }]
    }));
    setNewPhone('');
  };

  const removePhone = (phoneId: string) => {
    setAlertForm(prev => ({
      ...prev,
      phones: prev.phones.filter(p => p.id !== phoneId)
    }));
  };

  const handleCreateAlert = async () => {
    if (!alertForm.name.trim()) {
      toast({
        title: "Erro",
        description: "Nome do alerta é obrigatório",
        variant: "destructive"
      });
      return;
    }

    if (alertForm.phones.length === 0) {
      toast({
        title: "Erro",
        description: "Adicione pelo menos um telefone para receber os alertas",
        variant: "destructive"
      });
      return;
    }

    try {
      const createRequest: CreateAlertRequest = {
        name: alertForm.name,
        channel_id: channel.id,
        group_name: alertForm.name, // Use apenas o nome do alerta
        trigger_on: alertForm.trigger_on,
        phones: alertForm.phones.map(p => p.number).join(','),
        is_active: true
      };

      await createAlertMutation.mutateAsync(createRequest);

      toast({
        title: "Sucesso",
        description: "Alerta criado com sucesso! O grupo do WhatsApp será criado automaticamente."
      });

      resetForm();
      setIsCreateMode(false);
    } catch (error) {
      toast({
        title: "Erro",
        description: "Erro ao criar alerta",
        variant: "destructive"
      });
    }
  };

  const handleUpdateAlert = async () => {
    if (!editingAlert) return;

    try {
      await updateAlertMutation.mutateAsync({
        id: editingAlert.id,
        alert: {
          name: alertForm.name,
          group_name: alertForm.name, // Use apenas o nome do alerta
          trigger_on: alertForm.trigger_on,
          phones: alertForm.phones.map(p => p.number).join(','),
          is_active: true
        }
      });

      toast({
        title: "Sucesso",
        description: "Alerta atualizado com sucesso!"
      });

      setEditingAlert(null);
      resetForm();
    } catch (error) {
      toast({
        title: "Erro",
        description: "Erro ao atualizar alerta",
        variant: "destructive"
      });
    }
  };

  const handleDeleteAlert = async (alertId: string) => {
    try {
      await deleteAlertMutation.mutateAsync(alertId);
      toast({
        title: "Sucesso",
        description: "Alerta removido com sucesso!"
      });
    } catch (error) {
      toast({
        title: "Erro",
        description: "Erro ao remover alerta",
        variant: "destructive"
      });
    }
  };

  const getTriggerLabel = (trigger: string) => {
    switch (trigger) {
      case 'order_created':
        return 'Pedido Criado';
      case 'human_support_request':
        return 'Solicitação de Suporte Humano';
      default:
        return trigger;
    }
  };

  const getTriggerColor = (trigger: string) => {
    switch (trigger) {
      case 'order_created':
        return 'bg-blue-100 text-blue-800';
      case 'human_support_request':
        return 'bg-yellow-100 text-yellow-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Phone className="w-5 h-5" />
            Alertas do Canal: {channel.name}
          </DialogTitle>
          <DialogDescription>
            Configure alertas automáticos para notificações via WhatsApp sobre pedidos neste canal.
            Os alertas são enviados para grupos do ZapPlus criados automaticamente.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Canal Info */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium">Informações do Canal</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Nome:</span>
                <span className="font-medium">{channel.name}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Sessão:</span>
                <span className="font-mono text-xs bg-muted px-2 py-1 rounded">{channel.session}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Status:</span>
                <Badge variant={channel.status === 'connected' ? 'default' : 'destructive'}>
                  {channel.status || 'disconnected'}
                </Badge>
              </div>
            </CardContent>
          </Card>

          {/* Create/Edit Form */}
          {(isCreateMode || editingAlert) && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center justify-between">
                  <span>{editingAlert ? 'Editar Alerta' : 'Criar Novo Alerta'}</span>
                  <Button variant="ghost" size="sm" onClick={handleCancelEdit}>
                    <X className="w-4 h-4" />
                  </Button>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="alertName">Nome do Alerta</Label>
                    <Input
                      id="alertName"
                      value={alertForm.name}
                      onChange={(e) => setAlertForm(prev => ({ ...prev, name: e.target.value }))}
                      placeholder="Ex: Notificações de Pedidos"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="triggerType">Tipo de Gatilho</Label>
                    <Select
                      value={alertForm.trigger_on}
                      onValueChange={(value) => setAlertForm(prev => ({ ...prev, trigger_on: value as any }))}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="order_created">Pedido Criado</SelectItem>
                        <SelectItem value="human_support_request">Solicitação de Suporte Humano</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <Separator />

                {/* Phones Management */}
                <div className="space-y-3">
                  <Label>Telefones para Receber Alertas</Label>
                  <Alert>
                    <AlertDescription>
                      Digite o número de telefone com código do país. O formato para WhatsApp será aplicado automaticamente.
                    </AlertDescription>
                  </Alert>
                  
                  <div className="flex gap-2">
                    <PhoneNumberInput
                      value={newPhone}
                      onChange={(value) => setNewPhone(value || '')}
                      placeholder="Digite o número do telefone"
                      className="flex-1"
                    />
                    <Button onClick={addPhone} disabled={!newPhone.trim()}>
                      <Plus className="w-4 h-4" />
                      Adicionar
                    </Button>
                  </div>

                  {/* Phone List */}
                  {alertForm.phones.length > 0 && (
                    <div className="space-y-2">
                      <Label className="text-sm text-muted-foreground">
                        Telefones adicionados ({alertForm.phones.length}):
                      </Label>
                      <div className="space-y-2 max-h-32 overflow-y-auto">
                        {alertForm.phones.map((phone) => (
                          <div key={phone.id} className="flex items-center justify-between bg-muted p-2 rounded">
                            <span className="text-sm">{formatPhoneForDisplay(phone.number)}</span>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => removePhone(phone.id)}
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>

                <div className="flex justify-end gap-2 pt-4">
                  <Button variant="outline" onClick={handleCancelEdit}>
                    Cancelar
                  </Button>
                  <Button 
                    onClick={editingAlert ? handleUpdateAlert : handleCreateAlert}
                    disabled={createAlertMutation.isPending || updateAlertMutation.isPending}
                  >
                    {(createAlertMutation.isPending || updateAlertMutation.isPending) && (
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    )}
                    {editingAlert ? 'Salvar Alterações' : 'Criar Alerta'}
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Alerts List */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span>Alertas Configurados ({alerts.length})</span>
                {!isCreateMode && !editingAlert && (
                  <Button onClick={handleStartCreate}>
                    <Plus className="w-4 h-4 mr-2" />
                    Novo Alerta
                  </Button>
                )}
              </CardTitle>
              <CardDescription>
                Gerencie os alertas automáticos configurados para este canal
              </CardDescription>
            </CardHeader>
            <CardContent>
              {alertsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="w-6 h-6 animate-spin mr-2" />
                  Carregando alertas...
                </div>
              ) : alerts.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  <Users className="w-12 h-12 mx-auto mb-4 opacity-50" />
                  <p>Nenhum alerta configurado</p>
                  <p className="text-sm">Crie um alerta para receber notificações automáticas</p>
                </div>
              ) : (
                <div className="space-y-3">
                  {alerts.map((alert) => (
                    <div key={alert.id} className="border rounded-lg p-4 space-y-3">
                      <div className="flex items-center justify-between">
                        <div className="space-y-1">
                          <h4 className="font-medium">{alert.name}</h4>
                          <div className="flex items-center gap-2">
                            <Badge className={getTriggerColor(alert.trigger_on)}>
                              {getTriggerLabel(alert.trigger_on)}
                            </Badge>
                            <Badge variant={alert.is_active ? 'default' : 'secondary'}>
                              {alert.is_active ? 'Ativo' : 'Inativo'}
                            </Badge>
                            {alert.group_name && (
                              <Badge variant="outline">
                                Grupo: {alert.group_name}
                              </Badge>
                            )}
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleStartEdit(alert)}
                          >
                            <Edit2 className="w-4 h-4" />
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleDeleteAlert(alert.id)}
                            disabled={deleteAlertMutation.isPending}
                          >
                            {deleteAlertMutation.isPending ? (
                              <Loader2 className="w-4 h-4 animate-spin" />
                            ) : (
                              <Trash2 className="w-4 h-4" />
                            )}
                          </Button>
                        </div>
                      </div>
                      
                      {alert.phones && (
                        <div className="space-y-2">
                          <Label className="text-sm text-muted-foreground">
                            Telefones ({alert.phones.split(',').length}):
                          </Label>
                          <div className="flex flex-wrap gap-1">
                            {alert.phones.split(',').map((phone, index) => (
                              <Badge key={index} variant="secondary" className="text-xs">
                                {formatPhoneForDisplay(phone.trim())}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ChannelAlertsDialog;