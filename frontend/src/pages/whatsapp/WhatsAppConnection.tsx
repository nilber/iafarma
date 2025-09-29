import { useState, useEffect } from "react";
import { QrCode, RefreshCw, CheckCircle, XCircle, AlertTriangle, Bell } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useToast } from "@/hooks/use-toast";
import { useChannels } from "@/lib/api/hooks";
import { Channel } from "@/lib/api/types";
import { apiClient } from "@/lib/api/client";
import ChannelAlertsDialog from "@/components/alerts/ChannelAlertsDialog";

export default function WhatsAppConnection() {
  const { toast } = useToast();
  const { data: channelsData, isLoading, refetch } = useChannels();
  const [qrCode, setQrCode] = useState<string | null>(null);
  const [loadingQR, setLoadingQR] = useState(false);
  const [selectedChannelId, setSelectedChannelId] = useState<string | null>(null);
  const [alertsChannel, setAlertsChannel] = useState<Channel | null>(null);

  const channels = channelsData?.data || [];

  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'connected': return 'default';
      case 'connecting': return 'secondary';
      default: return 'destructive';
    }
  };

  const getStatusText = (status?: string) => {
    switch (status) {
      case 'connected': return 'Conectado';
      case 'connecting': return 'Conectando';
      default: return 'Desconectado';
    }
  };

  const handleGenerateQR = async (channelId: string) => {
    try {
      setLoadingQR(true);
      setSelectedChannelId(channelId);
      
      const qrCodeUrl = await apiClient.getWhatsAppQR(channelId);
      setQrCode(qrCodeUrl);
      
      toast({
        title: "QR Code gerado",
        description: "Escaneie o código QR com seu WhatsApp para conectar.",
      });
    } catch (error: any) {
      console.error('Erro ao gerar QR Code:', error);
      toast({
        title: "Erro",
        description: error.message || "Não foi possível gerar o QR Code.",
        variant: "destructive",
      });
    } finally {
      setLoadingQR(false);
    }
  };

  const handleDisconnect = async (channelId: string) => {
    if (!confirm("Tem certeza que deseja desconectar este canal?")) {
      return;
    }

    try {
      await apiClient.disconnectWhatsApp(channelId);
      
      toast({
        title: "Sucesso",
        description: "Canal desconectado com sucesso!",
      });
      
      refetch();
      setQrCode(null);
      setSelectedChannelId(null);
    } catch (error: any) {
      console.error('Erro ao desconectar canal:', error);
      toast({
        title: "Erro",
        description: error.message || "Não foi possível desconectar o canal.",
        variant: "destructive",
      });
    }
  };

  // Auto-refresh do status dos canais a cada 5 segundos
  useEffect(() => {
    const interval = setInterval(() => {
      refetch();
    }, 5000);

    return () => clearInterval(interval);
  }, [refetch]);

  if (isLoading) {
    return (
      <div className="container mx-auto py-6">
        <div className="flex items-center justify-center py-8">
          <RefreshCw className="h-6 w-6 animate-spin mr-2" />
          Carregando canais...
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-6 space-y-8">
            <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Canais e Alertas</h1>
          <p className="text-muted-foreground">
            Gerencie as conexões do WhatsApp e configure alertas para cada canal
          </p>
        </div>
        
        <Button onClick={() => refetch()} variant="outline">
          <RefreshCw className="h-4 w-4 mr-2" />
          Atualizar
        </Button>
      </div>

      {channels.length === 0 ? (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            Nenhum canal WhatsApp encontrado. Entre em contato com o administrador do sistema.
          </AlertDescription>
        </Alert>
      ) : (
        <div className="grid gap-6">
          {channels.map((channel: Channel) => (
            <Card key={channel.id} className="border shadow-sm hover:shadow-md transition-shadow">
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <CardTitle className="flex items-center gap-2 text-lg">
                      <QrCode className="h-5 w-5" />
                      {channel.name}
                    </CardTitle>
                    <CardDescription className="mt-1">
                      Session: {channel.session} | Type: {channel.type}
                    </CardDescription>
                  </div>
                  <div className="flex flex-col items-end gap-2">
                    <div className="flex items-center gap-2">
                      <Badge variant={channel.is_active ? "default" : "secondary"}>
                        {channel.is_active ? "Ativo" : "Inativo"}
                      </Badge>
                      <Badge variant={getStatusColor(channel.status)}>
                        {getStatusText(channel.status)}
                      </Badge>
                    </div>
                    <Button
                      variant={channel.status === 'connected' ? "default" : "outline"}
                      size="sm"
                      onClick={() => setAlertsChannel(channel)}
                      disabled={channel.status !== 'connected'}
                      className={channel.status === 'connected' ? 
                        "bg-blue-600 hover:bg-blue-700 text-white" : 
                        "opacity-50 cursor-not-allowed"
                      }
                    >
                      <Bell className="h-4 w-4 mr-1" />
                      {channel.status === 'connected' ? 'Configurar Alertas' : 'Alertas (Canal Desconectado)'}
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-3">
                <div className="space-y-4">
                  {channel.status === 'connected' ? (
                    <div className="bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800 rounded-lg p-3">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2 text-green-700 dark:text-green-300">
                          <CheckCircle className="h-5 w-5" />
                          <div>
                            <span className="font-medium">WhatsApp conectado</span>
                            <p className="text-sm text-green-600 dark:text-green-400 mt-1">
                              Canal funcionando normalmente e pronto para receber mensagens
                            </p>
                          </div>
                        </div>
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => handleDisconnect(channel.id)}
                        >
                          <XCircle className="h-4 w-4 mr-2" />
                          Desconectar
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <div className="space-y-4">
                      <div className="bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-lg p-3">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2 text-red-700 dark:text-red-300">
                            <XCircle className="h-5 w-5" />
                            <div>
                              <span className="font-medium">WhatsApp desconectado</span>
                              <p className="text-sm text-red-600 dark:text-red-400 mt-1">
                                Conecte o canal para receber mensagens e configurar alertas
                              </p>
                            </div>
                          </div>
                          <Button
                            onClick={() => handleGenerateQR(channel.id)}
                            disabled={loadingQR && selectedChannelId === channel.id}
                            className="bg-green-600 hover:bg-green-700 text-white"
                          >
                            {loadingQR && selectedChannelId === channel.id ? (
                              <>
                                <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                                Gerando...
                              </>
                            ) : (
                              <>
                                <QrCode className="h-4 w-4 mr-2" />
                                Conectar WhatsApp
                              </>
                            )}
                          </Button>
                        </div>
                      </div>

                      {qrCode && selectedChannelId === channel.id && (
                        <div className="bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded-lg p-6">
                          <div className="flex flex-col items-center space-y-4">
                            <div className="text-center">
                              <h3 className="font-semibold text-lg mb-2 text-blue-800 dark:text-blue-200">Escaneie o QR Code</h3>
                              <p className="text-sm text-blue-600 dark:text-blue-400 mb-4">
                                Abra o WhatsApp no seu celular e escaneie este código para conectar o canal
                              </p>
                            </div>
                            <div className="bg-white p-6 rounded-xl shadow-lg border">
                              <img 
                                src={qrCode} 
                                alt="QR Code WhatsApp" 
                                className="w-64 h-64 object-contain"
                              />
                            </div>
                            <Alert className="max-w-md">
                              <AlertTriangle className="h-4 w-4" />
                              <AlertDescription>
                                O QR Code expira em alguns minutos. Se não conseguir conectar, gere um novo código.
                              </AlertDescription>
                            </Alert>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
      
      {alertsChannel && (
        <ChannelAlertsDialog
          channel={alertsChannel}
          isOpen={true}
          onOpenChange={(open) => {
            if (!open) {
              setAlertsChannel(null);
            }
          }}
        />
      )}
    </div>
  );
}