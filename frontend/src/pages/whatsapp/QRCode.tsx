import { useState, useEffect } from "react";
import { QrCode, Smartphone, CheckCircle, AlertCircle, RefreshCw, Loader2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useAuth } from "@/contexts/AuthContext";
import { useWhatsAppQR, useWhatsAppSessionStatus } from "@/lib/api/hooks";

export default function WhatsAppQR() {
  const [connectionStatus, setConnectionStatus] = useState<'disconnected' | 'connecting' | 'connected'>('disconnected');
  const [showQR, setShowQR] = useState(false);
  const { user } = useAuth();
  
  // Check session status first
  const { data: sessionStatus, isLoading: statusLoading, error: statusError } = useWhatsAppSessionStatus();
  
  // Get QR code when requested - backend will handle tenant domain automatically
  const { data: qrImageUrl, isLoading: qrLoading, error: qrError, refetch: refetchQR } = useWhatsAppQR();

  // Update connection status based on session status
  useEffect(() => {
    if (sessionStatus) {
      if (sessionStatus.status === 'WORKING') {
        setConnectionStatus('connected');
        setShowQR(false);
      } else if (sessionStatus.status === 'SCAN_QR_CODE') {
        setConnectionStatus('disconnected');
      } else {
        setConnectionStatus('disconnected');
      }
    }
  }, [sessionStatus]);

  useEffect(() => {
    if (qrImageUrl && showQR) {
      setConnectionStatus('connecting');
    }
  }, [qrImageUrl, showQR]);

  const generateQR = () => {
    // Check if already connected
    if (sessionStatus?.status === 'WORKING') {
      alert('WhatsApp já está conectado! Não é necessário gerar um novo QR code.');
      return;
    }
    
    setShowQR(true);
    setConnectionStatus('connecting');
    refetchQR();
  };

  const refreshQR = () => {
    refetchQR();
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Header */}
      <div className="text-center">
        <h1 className="text-3xl font-bold text-foreground mb-2">Conectar WhatsApp</h1>
        <p className="text-muted-foreground">
          Escaneie o código QR para conectar sua conta do WhatsApp ao sistema
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* QR Code Section */}
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="text-center">
            <CardTitle className="flex items-center justify-center gap-2">
              <QrCode className="w-5 h-5 text-whatsapp" />
              Código QR
            </CardTitle>
            <CardDescription>
              Use seu WhatsApp para escanear o código abaixo
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col items-center space-y-6">
            {statusError && (
              <Alert variant="destructive">
                <AlertCircle className="w-4 h-4" />
                <AlertDescription>
                  Erro ao verificar status: {statusError.message}
                </AlertDescription>
              </Alert>
            )}

            {qrError && (
              <Alert variant="destructive">
                <AlertCircle className="w-4 h-4" />
                <AlertDescription>
                  Erro ao carregar QR Code: {qrError.message}
                </AlertDescription>
              </Alert>
            )}

            {/* Already connected state */}
            {sessionStatus?.status === 'WORKING' && (
              <>
                <div className="w-64 h-64 bg-success/10 rounded-lg flex items-center justify-center">
                  <div className="text-center">
                    <CheckCircle className="w-16 h-16 text-success mx-auto mb-4" />
                    <p className="text-success font-medium">WhatsApp Conectado!</p>
                    {sessionStatus.me && (
                      <p className="text-sm text-muted-foreground mt-2">
                        Conectado como: {sessionStatus.me.pushName}
                      </p>
                    )}
                  </div>
                </div>
                <Badge className="bg-success text-success-foreground">
                  Conexão Ativa
                </Badge>
              </>
            )}

            {/* Disconnected state - can generate QR */}
            {sessionStatus?.status !== 'WORKING' && connectionStatus === 'disconnected' && !showQR && (
              <>
                <div className="w-64 h-64 bg-muted rounded-lg flex items-center justify-center">
                  <div className="text-center">
                    <QrCode className="w-16 h-16 text-muted-foreground mx-auto mb-4" />
                    <p className="text-muted-foreground">Clique em "Gerar QR Code" para começar</p>
                  </div>
                </div>
                <Button 
                  onClick={generateQR}
                  className="bg-gradient-whatsapp"
                >
                  <QrCode className="w-4 h-4 mr-2" />
                  Gerar QR Code
                </Button>
              </>
            )}

            {showQR && qrLoading && (
              <>
                <div className="w-64 h-64 bg-muted rounded-lg flex items-center justify-center">
                  <div className="text-center">
                    <Loader2 className="w-16 h-16 text-muted-foreground mx-auto mb-4 animate-spin" />
                    <p className="text-muted-foreground">Carregando QR Code...</p>
                  </div>
                </div>
              </>
            )}

            {showQR && qrImageUrl && !qrError && (
              <>
                <div className="w-64 h-64 bg-white rounded-lg flex items-center justify-center border">
                  <img 
                    src={qrImageUrl} 
                    alt="WhatsApp QR Code" 
                    className="w-full h-full object-contain rounded-lg"
                  />
                </div>
                <div className="text-center space-y-2">
                  <Badge variant="secondary" className="mb-2">
                    Aguardando conexão...
                  </Badge>
                  <p className="text-sm text-muted-foreground">
                    Escaneie o código com seu WhatsApp
                  </p>
                  <Button
                    onClick={refreshQR}
                    variant="outline"
                    size="sm"
                    className="mt-2"
                  >
                    <RefreshCw className="w-4 h-4 mr-2" />
                    Atualizar QR
                  </Button>
                </div>
              </>
            )}

            {connectionStatus === 'connected' && (
              <>
                <div className="w-64 h-64 bg-success/10 rounded-lg flex items-center justify-center">
                  <div className="text-center">
                    <CheckCircle className="w-16 h-16 text-success mx-auto mb-4" />
                    <p className="text-success font-medium">Conectado com sucesso!</p>
                  </div>
                </div>
                <Badge className="bg-success text-success-foreground">
                  WhatsApp Conectado
                </Badge>
              </>
            )}
          </CardContent>
        </Card>

        {/* Instructions */}
        <Card className="border-0 shadow-custom-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Smartphone className="w-5 h-5 text-primary" />
              Como conectar
            </CardTitle>
            <CardDescription>
              Siga os passos abaixo para conectar seu WhatsApp
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-4">
              <div className="flex gap-3">
                <div className="w-8 h-8 bg-primary rounded-full flex items-center justify-center text-primary-foreground text-sm font-bold">
                  1
                </div>
                <div>
                  <p className="font-medium text-foreground">Abra o WhatsApp no seu celular</p>
                  <p className="text-sm text-muted-foreground">
                    Certifique-se de que está usando a versão mais recente
                  </p>
                </div>
              </div>

              <div className="flex gap-3">
                <div className="w-8 h-8 bg-primary rounded-full flex items-center justify-center text-primary-foreground text-sm font-bold">
                  2
                </div>
                <div>
                  <p className="font-medium text-foreground">Acesse as configurações</p>
                  <p className="text-sm text-muted-foreground">
                    Toque nos três pontos no canto superior direito
                  </p>
                </div>
              </div>

              <div className="flex gap-3">
                <div className="w-8 h-8 bg-primary rounded-full flex items-center justify-center text-primary-foreground text-sm font-bold">
                  3
                </div>
                <div>
                  <p className="font-medium text-foreground">Selecione "Aparelhos conectados"</p>
                  <p className="text-sm text-muted-foreground">
                    Em seguida, toque em "Conectar um aparelho"
                  </p>
                </div>
              </div>

              <div className="flex gap-3">
                <div className="w-8 h-8 bg-primary rounded-full flex items-center justify-center text-primary-foreground text-sm font-bold">
                  4
                </div>
                <div>
                  <p className="font-medium text-foreground">Escaneie o código QR</p>
                  <p className="text-sm text-muted-foreground">
                    Aponte a câmera para o código QR exibido acima
                  </p>
                </div>
              </div>
            </div>

            {connectionStatus === 'connected' && (
              <Alert className="border-success bg-success/5">
                <CheckCircle className="w-4 h-4 text-success" />
                <AlertDescription className="text-success">
                  Conexão estabelecida! Agora você pode receber e enviar mensagens através do sistema.
                </AlertDescription>
              </Alert>
            )}

            {connectionStatus === 'connecting' && (
              <Alert className="border-warning bg-warning/5">
                <AlertCircle className="w-4 h-4 text-warning" />
                <AlertDescription className="text-warning">
                  Aguardando conexão... Se o código não funcionar, tente gerar um novo QR Code.
                </AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Status Card */}
      <Card className="border-0 shadow-custom-md">
        <CardHeader>
          <CardTitle>Status da Conexão</CardTitle>
          <CardDescription>
            Informações sobre a conexão atual do WhatsApp
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="text-center">
              <div className={`w-16 h-16 rounded-full mx-auto mb-3 flex items-center justify-center ${
                connectionStatus === 'connected' ? 'bg-success/10' : 'bg-muted'
              }`}>
                <CheckCircle className={`w-8 h-8 ${
                  connectionStatus === 'connected' ? 'text-success' : 'text-muted-foreground'
                }`} />
              </div>
              <p className="font-medium text-foreground">Conexão</p>
              <p className={`text-sm ${
                connectionStatus === 'connected' ? 'text-success' : 'text-muted-foreground'
              }`}>
                {statusLoading ? 'Verificando...' : 
                 sessionStatus?.status === 'WORKING' ? 'Conectado' : 
                 sessionStatus?.status === 'SCAN_QR_CODE' ? 'Aguardando QR' : 'Desconectado'}
              </p>
              {sessionStatus?.me && (
                <p className="text-xs text-muted-foreground mt-1">
                  {sessionStatus.me.pushName}
                </p>
              )}
            </div>

            <div className="text-center">
              <div className="w-16 h-16 bg-muted rounded-full mx-auto mb-3 flex items-center justify-center">
                <span className="text-2xl font-bold text-muted-foreground">
                  {connectionStatus === 'connected' ? '24' : '0'}
                </span>
              </div>
              <p className="font-medium text-foreground">Mensagens Hoje</p>
              <p className="text-sm text-muted-foreground">
                {connectionStatus === 'connected' ? 'Últimas 24h' : 'Não conectado'}
              </p>
            </div>

            <div className="text-center">
              <div className="w-16 h-16 bg-muted rounded-full mx-auto mb-3 flex items-center justify-center">
                <span className="text-2xl font-bold text-muted-foreground">
                  {connectionStatus === 'connected' ? '8' : '0'}
                </span>
              </div>
              <p className="font-medium text-foreground">Conversas Ativas</p>
              <p className="text-sm text-muted-foreground">
                {connectionStatus === 'connected' ? 'Em andamento' : 'Não conectado'}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}