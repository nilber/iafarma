import { useState, useRef, useEffect } from "react";
import { Plus, QrCode, Smartphone, CheckCircle, AlertCircle, RefreshCw, Loader2, Settings, Trash2, Phone, Bell, ArrowRightLeft, Power } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/hooks/use-toast";
import { useChannels, useCreateChannel, useUpdateChannel, useDeleteChannel, useWhatsAppQR, useWhatsAppSessionStatus, useMigrateConversations } from "@/lib/api/hooks";
import { Channel } from "@/lib/api/types";
import ChannelAlertsDialog from "@/components/alerts/ChannelAlertsDialog";

interface CreateChannelForm {
  name: string;
  type: string;
  session: string;
  config: string;
  webhook_url: string;
}

interface QRState {
  channelId: string | null;
  showQR: boolean;
  connectionStatus: 'disconnected' | 'connecting' | 'connected';
}

export default function WhatsAppChannels() {
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null);
  const [alertsDialogOpen, setAlertsDialogOpen] = useState(false);
  const [selectedChannelForAlerts, setSelectedChannelForAlerts] = useState<Channel | null>(null);
  const [isMigrateModalOpen, setIsMigrateModalOpen] = useState(false);
  const [sourceChannelForMigration, setSourceChannelForMigration] = useState<Channel | null>(null);
  const [destinationChannelId, setDestinationChannelId] = useState<string>('');
  const [isMigrating, setIsMigrating] = useState(false);
  const [qrState, setQrState] = useState<QRState>({
    channelId: null,
    showQR: false,
    connectionStatus: 'disconnected'
  });
  
  const [formData, setFormData] = useState<CreateChannelForm>({
    name: '',
    type: 'whatsapp',
    session: '',
    config: '',
    webhook_url: ''
  });

  // API Hooks
  const { data: channelsData, isLoading: channelsLoading, error: channelsError, refetch: refetchChannels } = useChannels();
  const createChannelMutation = useCreateChannel();
  const updateChannelMutation = useUpdateChannel();
  const deleteChannelMutation = useDeleteChannel();
  const migrateConversationsMutation = useMigrateConversations();
  
  // QR Code hooks - only when needed  
  const { data: qrImageUrl, isLoading: qrLoading, error: qrError, refetch: refetchQR } = useWhatsAppQR(qrState.channelId || undefined);
  
  // Use intelligent polling - poll faster when connecting
  const isConnecting = qrState.showQR && qrState.connectionStatus === 'connecting';
  const { data: sessionStatus, isLoading: sessionLoading } = useWhatsAppSessionStatus(
    qrState.channelId || undefined, 
    isConnecting
  );

  const channels = channelsData?.data || [];

  // Watch for connection status changes
  const previousStatus = useRef<string | undefined>();
  useEffect(() => {
    if (sessionStatus?.status && previousStatus.current !== sessionStatus.status) {
      console.log('WhatsApp status changed:', previousStatus.current, '->', sessionStatus.status);
      
      // If just connected, update connection status and refresh channels
      if (sessionStatus.status === 'WORKING' && previousStatus.current !== 'WORKING') {
        setQrState(prev => ({ ...prev, connectionStatus: 'connected' }));
        
        // Refresh channels list to show updated status
        setTimeout(() => {
          refetchChannels();
        }, 1000);
        
        toast({
          title: "WhatsApp Conectado!",
          description: `Conectado como: ${sessionStatus.me?.pushName || 'WhatsApp'}`,
          variant: "default"
        });
        
        // Auto-close modal after 3 seconds of successful connection
        setTimeout(() => {
          setQrState(prev => ({ ...prev, showQR: false }));
        }, 3000);
      }
      
      previousStatus.current = sessionStatus.status;
    }
  }, [sessionStatus, refetchChannels, toast]);

  const handleCreateChannel = async () => {
    if (!formData.name.trim()) {
      toast({
        title: "Erro",
        description: "Nome do canal é obrigatório",
        variant: "destructive"
      });
      return;
    }

    if (!formData.session.trim()) {
      toast({
        title: "Erro",
        description: "Session é obrigatório",
        variant: "destructive"
      });
      return;
    }

    try {
      await createChannelMutation.mutateAsync({
        name: formData.name,
        type: formData.type,
        session: formData.session,
        status: 'disconnected',
        is_active: true,
        config: formData.config || undefined,
        webhook_url: formData.webhook_url || undefined,
        // tenant_id is automatically set by backend from JWT
      });

      toast({
        title: "Sucesso",
        description: "Canal criado com sucesso!"
      });

      setIsCreateModalOpen(false);
      setFormData({
        name: '',
        type: 'whatsapp',
        session: '',
        config: '',
        webhook_url: ''
      });
    } catch (error) {
      // Extract error message from API response
      let errorMessage = "Erro ao criar canal";
      
      if (error instanceof Error) {
        // Check if it's an ApiError with specific message
        if ('message' in error && error.message) {
          errorMessage = error.message;
        }
      }
      
      toast({
        title: "Erro",
        description: errorMessage,
        variant: "destructive"
      });
    }
  };

  const handleGenerateQR = (channelId: string) => {
    console.log('Generating QR for channel:', channelId);
    
    setQrState({
      channelId,
      showQR: true,
      connectionStatus: 'connecting'
    });
    
    // Set a small delay to ensure state is updated before refetching
    setTimeout(() => {
      refetchQR();
    }, 100);
  };

  const handleUpdateChannelStatus = async (channelId: string, status: string) => {
    try {
      await updateChannelMutation.mutateAsync({
        id: channelId,
        channel: { status }
      });

      toast({
        title: "Sucesso",
        description: "Status do canal atualizado!"
      });
    } catch (error) {
      toast({
        title: "Erro",
        description: "Erro ao atualizar status do canal",
        variant: "destructive"
      });
    }
  };

  const handleEditChannel = (channel: Channel) => {
    setEditingChannel(channel);
    setFormData({
      name: channel.name,
      type: channel.type,
      session: channel.session,
      config: channel.config || '',
      webhook_url: channel.webhook_url || ''
    });
    setIsEditModalOpen(true);
  };

  const handleUpdateChannel = async () => {
    if (!editingChannel) return;

    try {
      await updateChannelMutation.mutateAsync({
        id: editingChannel.id,
        channel: formData
      });

      setIsEditModalOpen(false);
      setEditingChannel(null);
      setFormData({
        name: '',
        type: 'whatsapp',
        session: '',
        config: '',
        webhook_url: ''
      });

      toast({
        title: "Sucesso",
        description: "Canal atualizado com sucesso!"
      });
    } catch (error) {
      toast({
        title: "Erro",
        description: "Erro ao atualizar canal",
        variant: "destructive"
      });
    }
  };

  const handleDeleteChannel = async (channelId: string, channelName: string) => {
    if (!confirm(`Tem certeza que deseja excluir o canal "${channelName}"? Esta ação não pode ser desfeita.`)) {
      return;
    }

    try {
      await deleteChannelMutation.mutateAsync(channelId);

      toast({
        title: "Sucesso",
        description: "Canal excluído com sucesso!"
      });
    } catch (error) {
      toast({
        title: "Erro",
        description: "Erro ao excluir canal",
        variant: "destructive"
      });
    }
  };

  const handleOpenAlerts = (channel: Channel) => {
    setSelectedChannelForAlerts(channel);
    setAlertsDialogOpen(true);
  };

  const handleOpenMigration = (channel: Channel) => {
    setSourceChannelForMigration(channel);
    setDestinationChannelId('');
    setIsMigrateModalOpen(true);
  };

  const handleMigrateConversations = async () => {
    if (!sourceChannelForMigration || !destinationChannelId) {
      toast({
        title: "Erro",
        description: "Selecione o canal de destino",
        variant: "destructive"
      });
      return;
    }

    setIsMigrating(true);
    try {
      const data = await migrateConversationsMutation.mutateAsync({
        channelId: sourceChannelForMigration.id,
        destinationChannelId: destinationChannelId
      });
      
      toast({
        title: "Sucesso",
        description: data.message,
        variant: "default"
      });

      setIsMigrateModalOpen(false);
      setSourceChannelForMigration(null);
      setDestinationChannelId('');
    } catch (error) {
      toast({
        title: "Erro",
        description: error instanceof Error ? error.message : "Erro ao migrar conversas",
        variant: "destructive"
      });
    } finally {
      setIsMigrating(false);
    }
  };

  const handleToggleActive = async (channelId: string, currentStatus: boolean) => {
    try {
      await updateChannelMutation.mutateAsync({
        id: channelId,
        channel: { is_active: !currentStatus }
      });

      toast({
        title: "Sucesso",
        description: `Canal ${!currentStatus ? 'ativado' : 'desativado'} com sucesso!`,
        variant: "default"
      });
    } catch (error) {
      toast({
        title: "Erro",
        description: error instanceof Error ? error.message : "Erro ao alterar status do canal",
        variant: "destructive"
      });
    }
  };

  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'connected':
        return 'bg-success text-success-foreground';
      case 'connecting':
        return 'bg-warning text-warning-foreground';
      default:
        return 'bg-destructive text-destructive-foreground';
    }
  };

  const getStatusText = (status?: string) => {
    switch (status) {
      case 'connected':
        return 'Conectado';
      case 'connecting':
        return 'Conectando';
      default:
        return 'Desconectado';
    }
  };

  if (channelsLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin" />
      </div>
    );
  }

  if (channelsError) {
    return (
      <Alert variant="destructive" className="max-w-2xl mx-auto">
        <AlertCircle className="w-4 h-4" />
        <AlertDescription>
          Erro ao carregar canais: {channelsError.message}
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-foreground mb-2">Gerenciar Canais</h1>
          <p className="text-muted-foreground">
            Gerencie os canais de WhatsApp da sua empresa
          </p>
        </div>
        <Dialog open={isCreateModalOpen} onOpenChange={(open) => {
          setIsCreateModalOpen(open);
          if (!open) {
            // Refetch channels when modal is closed
            refetchChannels();
          }
        }}>
          <DialogTrigger asChild>
            <Button className="bg-gradient-whatsapp">
              <Plus className="w-4 h-4 mr-2" />
              Novo Canal
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[500px] max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Criar Novo Canal</DialogTitle>
              <DialogDescription>
                Configure um novo canal de atendimento
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="name">Nome do Canal *</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Ex: WhatsApp Principal"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="type">Tipo</Label>
                <Select
                  value={formData.type}
                  onValueChange={(value) => setFormData({ ...formData, type: value })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="whatsapp">WhatsApp</SelectItem>
                    <SelectItem value="webchat">Web Chat</SelectItem>
                    <SelectItem value="telegram">Telegram</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="session">Session ID *</Label>
                <Input
                  id="session"
                  value={formData.session}
                  onChange={(e) => setFormData({ ...formData, session: e.target.value })}
                  placeholder="Ex: whatsapp-principal"
                  required
                />
                <p className="text-xs text-muted-foreground">
                  Identificador único para a sessão do WhatsApp (sem espaços ou caracteres especiais)
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="webhook_url">URL do Webhook (Opcional)</Label>
                <Input
                  id="webhook_url"
                  value={formData.webhook_url}
                  onChange={(e) => setFormData({ ...formData, webhook_url: e.target.value })}
                  placeholder="https://..."
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="config">Configuração (JSON)</Label>
                <Textarea
                  id="config"
                  value={formData.config}
                  onChange={(e) => setFormData({ ...formData, config: e.target.value })}
                  placeholder='{"phone_number_id": "123456789"}'
                  className="min-h-[100px]"
                />
              </div>
            </div>
            <div className="flex justify-end gap-3">
              <Button variant="outline" onClick={() => setIsCreateModalOpen(false)}>
                Cancelar
              </Button>
              <Button 
                onClick={handleCreateChannel}
                disabled={createChannelMutation.isPending}
              >
                {createChannelMutation.isPending ? (
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                ) : (
                  <Plus className="w-4 h-4 mr-2" />
                )}
                Criar Canal
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* Channels List */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {channels.map((channel) => (
          <Card key={channel.id} className="border-0 shadow-custom-md">
            <CardHeader className="pb-3">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-whatsapp/10 rounded-lg">
                    {channel.type === 'whatsapp' ? (
                      <Phone className="w-5 h-5 text-whatsapp" />
                    ) : (
                      <Smartphone className="w-5 h-5 text-primary" />
                    )}
                  </div>
                  <div>
                    <CardTitle className="text-lg">{channel.name}</CardTitle>
                    <CardDescription className="capitalize">
                      {channel.type}
                    </CardDescription>
                  </div>
                </div>
                <Badge className={getStatusColor(channel.status)}>
                  {getStatusText(channel.status)}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">ID:</span>
                  <span className="font-mono text-xs">
                    {channel.id.substring(0, 8)}...
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Session:</span>
                  <span className="font-mono text-xs">
                    {channel.session}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Criado:</span>
                  <span>
                    {new Date(channel.created_at).toLocaleDateString('pt-BR')}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Conversas:</span>
                  <span className="font-medium">
                    {channel.conversation_count || 0}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Ativo:</span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleToggleActive(channel.id, channel.is_active)}
                    className={`px-2 py-1 h-auto ${
                      channel.is_active 
                        ? 'text-green-600 hover:text-green-700' 
                        : 'text-gray-500 hover:text-gray-600'
                    }`}
                  >
                    <Power className={`w-4 h-4 mr-1 ${
                      channel.is_active ? 'text-green-600' : 'text-gray-400'
                    }`} />
                    {channel.is_active ? 'Ativado' : 'Desativado'}
                  </Button>
                </div>
              </div>

              <div className="flex flex-col gap-2">
                {channel.type === 'whatsapp' && channel.status !== 'connected' && (
                  <Button
                    onClick={() => handleGenerateQR(channel.id)}
                    variant="default"
                    size="sm"
                    className="w-full bg-whatsapp hover:bg-whatsapp/90"
                  >
                    <QrCode className="w-4 h-4 mr-2" />
                    Conectar WhatsApp
                  </Button>
                )}

                {channel.status === 'connected' && (
                  <Button
                    onClick={() => handleUpdateChannelStatus(channel.id, 'disconnected')}
                    variant="outline"
                    size="sm"
                    className="w-full"
                  >
                    <RefreshCw className="w-4 h-4 mr-2" />
                    Desconectar
                  </Button>
                )}

                <div className="flex flex-col gap-2">
                  <div className="flex gap-2">
                    <Button 
                      variant="outline" 
                      size="sm" 
                      className="flex-1"
                      onClick={() => handleEditChannel(channel)}
                    >
                      <Settings className="w-4 h-4 mr-1" />
                      Config
                    </Button>
                    <Button 
                      variant="outline" 
                      size="sm" 
                      className="flex-1"
                      onClick={() => handleOpenAlerts(channel)}
                    >
                      <Bell className="w-4 h-4 mr-1" />
                      Alertas
                    </Button>
                  </div>
                  <div className="flex gap-2">
                    <Button 
                      variant="outline" 
                      size="sm" 
                      className="flex-1"
                      onClick={() => handleOpenMigration(channel)}
                      disabled={
                        channels.length < 2 || 
                        (channel.is_active && channel.status === 'connected')
                      }
                      title={
                        channels.length < 2 
                          ? "É necessário ter pelo menos 2 canais para migrar conversas"
                          : (channel.is_active && channel.status === 'connected')
                          ? "Desconecte o canal antes de migrar as conversas"
                          : "Migrar conversas para outro canal"
                      }
                    >
                      <ArrowRightLeft className="w-4 h-4 mr-1" />
                      Migrar
                    </Button>
                    <Button 
                      variant="outline" 
                      size="sm" 
                      className="text-destructive hover:text-destructive"
                      onClick={() => handleDeleteChannel(channel.id, channel.name)}
                      disabled={
                        deleteChannelMutation.isPending || 
                        (channel.conversation_count && channel.conversation_count > 0)
                      }
                      title={
                        (channel.conversation_count && channel.conversation_count > 0)
                          ? `Cannot delete channel with ${channel.conversation_count} conversations. Migrate conversations first.`
                          : "Excluir canal"
                      }
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Empty state */}
      {channels.length === 0 && (
        <Card className="border-0 shadow-custom-md">
          <CardContent className="text-center py-12">
            <div className="w-24 h-24 bg-muted rounded-full mx-auto mb-4 flex items-center justify-center">
              <Phone className="w-12 h-12 text-muted-foreground" />
            </div>
            <h3 className="text-lg font-medium mb-2">Nenhum canal configurado</h3>
            <p className="text-muted-foreground mb-4">
              Crie seu primeiro canal para começar a receber mensagens
            </p>
            <Button onClick={() => setIsCreateModalOpen(true)} className="bg-gradient-whatsapp">
              <Plus className="w-4 h-4 mr-2" />
              Criar Primeiro Canal
            </Button>
          </CardContent>
        </Card>
      )}

      {/* QR Code Modal */}
      {qrState.showQR && (
        <Dialog open={qrState.showQR} onOpenChange={(open) => {
          if (!open) {
            setQrState({ ...qrState, showQR: false });
            // Refetch channels when modal is closed to update the list
            refetchChannels();
          }
        }}>
          <DialogContent className="sm:max-w-[500px] max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <QrCode className="w-5 h-5 text-whatsapp" />
                Conectar WhatsApp
              </DialogTitle>
              <DialogDescription>
                Escaneie o código QR com seu WhatsApp para conectar o canal
              </DialogDescription>
            </DialogHeader>
            
            <div className="flex flex-col items-center space-y-6 py-4">
              {qrError && (
                <Alert variant="destructive">
                  <AlertCircle className="w-4 h-4" />
                  <AlertDescription>
                    Erro ao carregar QR Code: {qrError.message}
                  </AlertDescription>
                </Alert>
              )}

              {/* Connection Status Indicator */}
              <div className="w-full max-w-md">
                <div className="flex items-center justify-center mb-4">
                  {sessionStatus?.status === 'WORKING' ? (
                    <Badge className="bg-green-500 text-white animate-pulse">
                      <CheckCircle className="w-3 h-3 mr-1" />
                      Conectado
                    </Badge>
                  ) : sessionStatus?.status === 'SCAN_QR_CODE' ? (
                    <Badge className="bg-blue-500 text-white animate-pulse">
                      <QrCode className="w-3 h-3 mr-1" />
                      Aguardando escaneamento
                    </Badge>
                  ) : sessionStatus?.status === 'STOPPED' ? (
                    <Badge className="bg-gray-500 text-white">
                      <Power className="w-3 h-3 mr-1" />
                      Desconectado
                    </Badge>
                  ) : sessionStatus?.status === 'FAILED' ? (
                    <Badge className="bg-red-500 text-white">
                      <AlertCircle className="w-3 h-3 mr-1" />
                      Falha na conexão
                    </Badge>
                  ) : isConnecting ? (
                    <Badge className="bg-yellow-500 text-white animate-pulse">
                      <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                      Iniciando conexão...
                    </Badge>
                  ) : (
                    <Badge variant="secondary">
                      Pronto para conectar
                    </Badge>
                  )}
                </div>
              </div>

              {/* Already connected state */}
              {qrState.channelId && sessionStatus?.status === 'WORKING' && (
                <>
                  <div className="w-64 h-64 bg-green-50 border-2 border-green-200 rounded-lg flex items-center justify-center">
                    <div className="text-center">
                      <CheckCircle className="w-16 h-16 text-green-600 mx-auto mb-4 animate-bounce" />
                      <p className="text-green-600 font-medium">WhatsApp Conectado!</p>
                      {sessionStatus.me && (
                        <p className="text-sm text-muted-foreground mt-2">
                          Conectado como: {sessionStatus.me.pushName}
                        </p>
                      )}
                    </div>
                  </div>
                  <div className="text-center space-y-2">
                    <p className="text-sm text-green-600 font-medium">
                      ✅ Conexão estabelecida com sucesso!
                    </p>
                    <p className="text-xs text-muted-foreground">
                      Você pode fechar este modal
                    </p>
                  </div>
                </>
              )}

              {/* Loading QR */}
              {qrState.channelId && qrLoading && sessionStatus?.status !== 'WORKING' && (
                <>
                  <div className="w-64 h-64 bg-muted rounded-lg flex items-center justify-center">
                    <div className="text-center">
                      <Loader2 className="w-16 h-16 text-muted-foreground mx-auto mb-4 animate-spin" />
                      <p className="text-muted-foreground">Carregando QR Code...</p>
                    </div>
                  </div>
                </>
              )}

              {/* QR Code Display */}
              {qrState.channelId && qrImageUrl && !qrError && !qrLoading && sessionStatus?.status !== 'WORKING' && (
                <>
                  <div className="w-64 h-64 bg-white rounded-lg flex items-center justify-center border-2 border-blue-200">
                    <img 
                      src={qrImageUrl} 
                      alt="WhatsApp QR Code" 
                      className="w-full h-full object-contain rounded-lg"
                    />
                  </div>
                  <div className="text-center space-y-3">
                    <div className="space-y-1">
                      <p className="text-sm font-medium text-foreground">
                        {sessionStatus?.status === 'SCAN_QR_CODE' 
                          ? '📱 Escaneie com seu WhatsApp'
                          : '🔄 Aguardando conexão...'
                        }
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {sessionStatus?.status === 'SCAN_QR_CODE' 
                          ? 'Abra o WhatsApp > Menu (3 pontos) > Dispositivos Conectados > Conectar um dispositivo'
                          : 'Aguardando resposta do servidor...'
                        }
                      </p>
                    </div>
                    
                    <div className="flex gap-2 justify-center">
                      <Button
                        onClick={() => refetchQR()}
                        variant="outline"
                        size="sm"
                      >
                        <RefreshCw className="w-4 h-4 mr-2" />
                        Novo QR
                      </Button>
                    </div>
                  </div>
                </>
              )}

              {/* Initial state or no channel selected */}
              {!qrState.channelId && (
                <div className="w-64 h-64 bg-muted rounded-lg flex items-center justify-center">
                  <div className="text-center">
                    <QrCode className="w-16 h-16 text-muted-foreground mx-auto mb-4" />
                    <p className="text-muted-foreground">Nenhum canal selecionado</p>
                  </div>
                </div>
              )}
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Edit Channel Modal */}
      {isEditModalOpen && (
        <Dialog open={isEditModalOpen} onOpenChange={(open) => {
          setIsEditModalOpen(open);
          if (!open) {
            // Refetch channels when modal is closed
            refetchChannels();
          }
        }}>
          <DialogContent className="sm:max-w-[500px] max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <Settings className="w-5 h-5 text-primary" />
                Editar Canal
              </DialogTitle>
              <DialogDescription>
                Atualize as configurações do canal WhatsApp
              </DialogDescription>
            </DialogHeader>
            
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="edit-name">Nome do Canal *</Label>
                <Input
                  id="edit-name"
                  placeholder="Ex: Atendimento Principal"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="edit-session">Session ID *</Label>
                <Input
                  id="edit-session"
                  placeholder="Ex: whatsapp-principal"
                  value={formData.session}
                  onChange={(e) => setFormData({ ...formData, session: e.target.value })}
                  required
                />
                <p className="text-xs text-muted-foreground">
                  Identificador único para o canal (somente letras, números e hífens)
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="edit-type">Tipo de Canal</Label>
                <Select 
                  value={formData.type} 
                  onValueChange={(value) => setFormData({ ...formData, type: value })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Selecione o tipo" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="whatsapp">WhatsApp</SelectItem>
                    <SelectItem value="telegram">Telegram</SelectItem>
                    <SelectItem value="sms">SMS</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="edit-webhook">Webhook URL</Label>
                <Input
                  id="edit-webhook"
                  placeholder="https://seu-dominio.com/webhook"
                  value={formData.webhook_url}
                  onChange={(e) => setFormData({ ...formData, webhook_url: e.target.value })}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="edit-config">Configurações (JSON)</Label>
                <Textarea
                  id="edit-config"
                  placeholder='{"timeout": 30000, "retries": 3}'
                  value={formData.config}
                  onChange={(e) => setFormData({ ...formData, config: e.target.value })}
                  rows={3}
                />
              </div>

              {/* WhatsApp QR Code Section */}
              {editingChannel && editingChannel.type === 'whatsapp' && (
                <div className="space-y-2 p-4 bg-muted/50 rounded-lg">
                  <Label className="text-sm font-medium">WhatsApp Connection</Label>
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => handleGenerateQR(editingChannel.id)}
                      className="flex-1"
                    >
                      <QrCode className="w-4 h-4 mr-2" />
                      Gerar QR Code
                    </Button>
                    <Badge variant={editingChannel.status === 'connected' ? 'default' : 'secondary'}>
                      {getStatusText(editingChannel.status)}
                    </Badge>
                  </div>
                </div>
              )}
            </div>

            <div className="flex justify-end space-x-2">
              <Button variant="outline" onClick={() => setIsEditModalOpen(false)}>
                Cancelar
              </Button>
              <Button 
                onClick={handleUpdateChannel}
                disabled={!formData.name || !formData.session || updateChannelMutation.isPending}
                className="bg-primary hover:bg-primary/90"
              >
                {updateChannelMutation.isPending ? "Salvando..." : "Salvar Alterações"}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Alerts Dialog */}
      {selectedChannelForAlerts && (
        <ChannelAlertsDialog
          channel={selectedChannelForAlerts}
          isOpen={alertsDialogOpen}
          onOpenChange={(open) => {
            setAlertsDialogOpen(open);
            if (!open) {
              setSelectedChannelForAlerts(null);
            }
          }}
        />
      )}

      {/* Migrate Conversations Dialog */}
      <Dialog open={isMigrateModalOpen} onOpenChange={(open) => {
        setIsMigrateModalOpen(open);
        if (!open) {
          setSourceChannelForMigration(null);
          setDestinationChannelId('');
        }
      }}>
        <DialogContent className="sm:max-w-md max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Migrar Conversas</DialogTitle>
            <DialogDescription>
              Migrar todas as conversas do canal "{sourceChannelForMigration?.name}" para outro canal.
              Esta ação não pode ser desfeita.
              <br />
              <br />
              <strong>Nota:</strong> Só é possível migrar conversas de canais desconectados ou inativos.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="destination-channel">Canal de Destino</Label>
              <Select value={destinationChannelId} onValueChange={setDestinationChannelId}>
                <SelectTrigger>
                  <SelectValue placeholder="Selecione o canal de destino" />
                </SelectTrigger>
                <SelectContent>
                  {channels
                    .filter(channel => channel.id !== sourceChannelForMigration?.id)
                    .map(channel => (
                      <SelectItem key={channel.id} value={channel.id}>
                        {channel.name} ({channel.status === 'connected' ? 'Conectado' : 'Desconectado'})
                      </SelectItem>
                    ))
                  }
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="flex justify-end gap-2">
            <Button 
              variant="outline" 
              onClick={() => setIsMigrateModalOpen(false)}
              disabled={isMigrating}
            >
              Cancelar
            </Button>
            <Button 
              onClick={handleMigrateConversations}
              disabled={!destinationChannelId || isMigrating}
            >
              {isMigrating ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Migrando...
                </>
              ) : (
                <>
                  <ArrowRightLeft className="w-4 h-4 mr-2" />
                  Migrar Conversas
                </>
              )}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
