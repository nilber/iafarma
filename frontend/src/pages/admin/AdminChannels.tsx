import { useState, useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { 
  Phone, 
  QrCode, 
  Check, 
  X, 
  Trash2, 
  Edit, 
  Plus,
  Users,
  AlertCircle,
  RefreshCw,
  ExternalLink,
  Building2
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useToast } from "@/hooks/use-toast";
import { useAuth } from "@/contexts/AuthContext";
import { apiClient } from "@/lib/api/client";
import { Tenant, Channel } from "@/lib/api/types";
import { Switch } from "@/components/ui/switch";

interface ChannelFormData {
  name: string;
  type: string;
  session: string;
  is_active: boolean;
}

export default function AdminChannels() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const { toast } = useToast();
  
  const [selectedTenant, setSelectedTenant] = useState<string>("");
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(false);
  const [channelsLoading, setChannelsLoading] = useState(false);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null);
  const [formData, setFormData] = useState<ChannelFormData>({
    name: "",
    type: "whatsapp",
    session: "",
    is_active: true
  });
  const [qrModalOpen, setQrModalOpen] = useState(false);
  const [selectedChannelId, setSelectedChannelId] = useState<string | null>(null);
  const [qrCode, setQrCode] = useState<string | null>(null);
  const [qrLoading, setQrLoading] = useState(false);
  const [searchParams] = useSearchParams();

  // Verificar se é system admin
  useEffect(() => {
    if (user?.role !== 'system_admin') {
      navigate('/');
      return;
    }
  }, [user, navigate]);

  // Carregar tenants e selecionar automaticamente se há parâmetro na URL
  useEffect(() => {
    const loadTenants = async () => {
      try {
        setLoading(true);
        const response = await apiClient.getTenants();
        setTenants(response.data || []);
        
        // Verificar se há parâmetro tenant na URL
        const tenantParam = searchParams.get('tenant');
        if (tenantParam && response.data?.some(t => t.id === tenantParam)) {
          setSelectedTenant(tenantParam);
        }
      } catch (error) {
        console.error('Erro ao carregar tenants:', error);
        toast({
          title: "Erro",
          description: "Não foi possível carregar a lista de empresas.",
          variant: "destructive",
        });
      } finally {
        setLoading(false);
      }
    };

    loadTenants();
  }, [toast, searchParams]);

  // Carregar canais quando tenant for selecionado
  useEffect(() => {
    if (selectedTenant) {
      loadChannels();
    } else {
      setChannels([]);
    }
  }, [selectedTenant]);

  const loadChannels = async () => {
    if (!selectedTenant) return;

    try {
      setChannelsLoading(true);
      const response = await apiClient.getChannelsByTenant(selectedTenant);
      setChannels(response.data || []);
    } catch (error) {
      console.error('Erro ao carregar canais:', error);
      toast({
        title: "Erro",
        description: "Não foi possível carregar os canais desta empresa.",
        variant: "destructive",
      });
    } finally {
      setChannelsLoading(false);
    }
  };

  const handleCreateChannel = async () => {
    if (!selectedTenant) return;

    try {
      await apiClient.createChannelForTenant(selectedTenant, formData);
      
      toast({
        title: "Sucesso",
        description: "Canal criado com sucesso!",
      });
      
      setIsDialogOpen(false);
      resetForm();
      loadChannels();
    } catch (error: any) {
      console.error('Erro ao criar canal:', error);
      toast({
        title: "Erro",
        description: error.message || "Não foi possível criar o canal.",
        variant: "destructive",
      });
    }
  };

  const handleUpdateChannel = async () => {
    if (!selectedTenant || !editingChannel) return;

    try {
      await apiClient.updateChannelForTenant(selectedTenant, editingChannel.id, formData);
      
      toast({
        title: "Sucesso",
        description: "Canal atualizado com sucesso!",
      });
      
      setIsDialogOpen(false);
      resetForm();
      loadChannels();
    } catch (error: any) {
      console.error('Erro ao atualizar canal:', error);
      toast({
        title: "Erro",
        description: error.message || "Não foi possível atualizar o canal.",
        variant: "destructive",
      });
    }
  };

  const handleDeleteChannel = async (channelId: string) => {
    if (!selectedTenant) return;

    if (!confirm("Tem certeza que deseja excluir este canal? Esta ação não pode ser desfeita.")) {
      return;
    }

    try {
      await apiClient.deleteChannelForTenant(selectedTenant, channelId);
      
      toast({
        title: "Sucesso",
        description: "Canal excluído com sucesso!",
      });
      
      loadChannels();
    } catch (error: any) {
      console.error('Erro ao excluir canal:', error);
      toast({
        title: "Erro",
        description: error.message || "Não foi possível excluir o canal.",
        variant: "destructive",
      });
    }
  };

  const handleGenerateQR = async (channelId: string) => {
    if (!selectedTenant) return;

    try {
      setSelectedChannelId(channelId);
      setQrModalOpen(true);
      setQrLoading(true);
      setQrCode(null);

      // Gerar QR Code via API - incluir tenant_id para system_admin
      const qrCodeUrl = user?.role === 'system_admin' 
        ? await apiClient.getWhatsAppQR(channelId, selectedTenant)
        : await apiClient.getWhatsAppQR(channelId);
      setQrCode(qrCodeUrl);
      
      toast({
        title: "QR Code gerado",
        description: "Use seu celular para escanear o QR Code e conectar ao WhatsApp.",
      });
    } catch (error: any) {
      console.error('Erro ao gerar QR Code:', error);
      toast({
        title: "Erro",
        description: error.message || "Não foi possível gerar o QR Code.",
        variant: "destructive",
      });
      setQrModalOpen(false);
    } finally {
      setQrLoading(false);
    }
  };

  const handleEditChannel = (channel: Channel) => {
    setEditingChannel(channel);
    setFormData({
      name: channel.name,
      type: channel.type,
      session: channel.session,
      is_active: channel.is_active
    });
    setIsDialogOpen(true);
  };

  const resetForm = () => {
    setFormData({
      name: "",
      type: "whatsapp",
      session: "",
      is_active: true
    });
    setEditingChannel(null);
  };

  const handleCloseDialog = () => {
    setIsDialogOpen(false);
    resetForm();
  };

  const selectedTenantData = tenants.find(t => t.id === selectedTenant);

  if (user?.role !== 'system_admin') {
    return null;
  }

  return (
    <div className="container mx-auto py-6 space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Administrar Canais</h1>
          <p className="text-muted-foreground">
            Gerencie os canais WhatsApp das empresas do sistema
          </p>
        </div>
      </div>

      {/* Seleção de Tenant */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Building2 className="h-5 w-5" />
            Selecionar Empresa
          </CardTitle>
          <CardDescription>
            Escolha uma empresa para gerenciar seus canais WhatsApp
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Select value={selectedTenant} onValueChange={setSelectedTenant}>
            <SelectTrigger className="w-full max-w-md">
              <SelectValue placeholder="Selecione uma empresa..." />
            </SelectTrigger>
            <SelectContent>
              {tenants.map((tenant) => (
                <SelectItem key={tenant.id} value={tenant.id}>
                  {tenant.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      {/* Lista de Canais */}
      {selectedTenant && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Phone className="h-5 w-5" />
                  Canais WhatsApp - {selectedTenantData?.name}
                </CardTitle>
                <CardDescription>
                  Gerencie os canais WhatsApp desta empresa
                </CardDescription>
              </div>
              <div className="flex gap-2">
                <Button onClick={loadChannels} variant="outline" size="sm">
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Atualizar
                </Button>
                <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
                  <DialogTrigger asChild>
                    <Button onClick={() => resetForm()}>
                      <Plus className="h-4 w-4 mr-2" />
                      Novo Canal
                    </Button>
                  </DialogTrigger>
                  <DialogContent>
                    <DialogHeader>
                      <DialogTitle>
                        {editingChannel ? "Editar Canal" : "Novo Canal"}
                      </DialogTitle>
                      <DialogDescription>
                        {editingChannel 
                          ? "Edite as informações do canal WhatsApp." 
                          : "Crie um novo canal WhatsApp para esta empresa."
                        }
                      </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4">
                      <div>
                        <Label htmlFor="name">Nome do Canal</Label>
                        <Input
                          id="name"
                          value={formData.name}
                          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                          placeholder="Ex: Atendimento Principal"
                        />
                      </div>
                      <div>
                        <Label htmlFor="session">Session ID</Label>
                        <Input
                          id="session"
                          value={formData.session}
                          onChange={(e) => setFormData({ ...formData, session: e.target.value })}
                          placeholder="Ex: farma, loja1, atendimento..."
                        />
                      </div>
                      <div className="flex items-center space-x-2">
                        <Switch
                          id="is_active"
                          checked={formData.is_active}
                          onCheckedChange={(checked) => setFormData({ ...formData, is_active: checked })}
                        />
                        <Label htmlFor="is_active">Canal ativo</Label>
                      </div>
                      <div className="flex justify-end gap-2">
                        <Button variant="outline" onClick={handleCloseDialog}>
                          Cancelar
                        </Button>
                        <Button onClick={editingChannel ? handleUpdateChannel : handleCreateChannel}>
                          {editingChannel ? "Salvar" : "Criar Canal"}
                        </Button>
                      </div>
                    </div>
                  </DialogContent>
                </Dialog>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {channelsLoading ? (
              <div className="flex items-center justify-center py-8">
                <RefreshCw className="h-6 w-6 animate-spin mr-2" />
                Carregando canais...
              </div>
            ) : channels.length === 0 ? (
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  Nenhum canal encontrado para esta empresa.
                </AlertDescription>
              </Alert>
            ) : (
              <div className="grid gap-4">
                {channels.map((channel) => (
                  <Card key={channel.id} className="border">
                    <CardContent className="p-4">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="flex items-center justify-center w-10 h-10 bg-green-100 rounded-full">
                            <Phone className="h-5 w-5 text-green-600" />
                          </div>
                          <div>
                            <h3 className="font-semibold">{channel.name}</h3>
                            <p className="text-sm text-muted-foreground">
                              Session: {channel.session} | Type: {channel.type}
                            </p>
                            <div className="flex items-center gap-2 mt-1">
                              <Badge variant={channel.is_active ? "default" : "secondary"}>
                                {channel.is_active ? "Ativo" : "Inativo"}
                              </Badge>
                              <Badge variant={channel.status === 'connected' ? "default" : "destructive"}>
                                {channel.status === 'connected' ? "Conectado" : "Desconectado"}
                              </Badge>
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleGenerateQR(channel.id)}
                            disabled={qrLoading}
                            title="Gerar QR Code"
                          >
                            {qrLoading && selectedChannelId === channel.id ? (
                              <RefreshCw className="h-4 w-4 animate-spin" />
                            ) : (
                              <QrCode className="h-4 w-4" />
                            )}
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleEditChannel(channel)}
                          >
                            <Edit className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleDeleteChannel(channel.id)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* QR Code Modal */}
      <Dialog open={qrModalOpen} onOpenChange={setQrModalOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>QR Code do WhatsApp</DialogTitle>
            <DialogDescription>
              Escaneie este QR Code com o WhatsApp para conectar o canal.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col items-center justify-center py-6">
            {qrLoading ? (
              <div className="flex flex-col items-center gap-4">
                <RefreshCw className="h-8 w-8 animate-spin text-blue-600" />
                <p className="text-sm text-muted-foreground">
                  Gerando QR Code...
                </p>
              </div>
            ) : qrCode ? (
              <div className="flex flex-col items-center gap-4">
                <div className="p-4 bg-white rounded-lg border">
                  <img
                    src={`data:image/png;base64,${qrCode}`}
                    alt="QR Code WhatsApp"
                    className="w-64 h-64"
                  />
                </div>
                <p className="text-sm text-center text-muted-foreground max-w-sm">
                  Abra o WhatsApp no seu celular, vá em <strong>Configurações</strong> → <strong>Aparelhos conectados</strong> → <strong>Conectar um aparelho</strong> e escaneie este código.
                </p>
              </div>
            ) : (
              <div className="flex flex-col items-center gap-4">
                <AlertCircle className="h-8 w-8 text-orange-500" />
                <p className="text-sm text-muted-foreground">
                  Erro ao gerar QR Code. Tente novamente.
                </p>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}